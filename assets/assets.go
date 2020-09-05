// +build dev

package assets

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/build"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/shurcooL/go/gopherjs_http"
	"github.com/shurcooL/go/osutil"
	"github.com/shurcooL/httpfs/union"
)

// Assets contains assets for home.
var Assets = union.New(map[string]http.FileSystem{
	"/assets":   gopherjs_http.NewFS(http.Dir(importPathToDir("github.com/shurcooL/home/_data"))),
	"/spa.wasm": packageWasmFS{"github.com/shurcooL/home/internal/exp/cmd/spa"},
	"/wasm_exec_go1" + fmt.Sprint(goVersion) + ".js": &wasmExecFile{},
	"/issues":        http.Dir(importPathToDir("github.com/shurcooL/home/internal/exp/app/issuesapp/_data")),
	"/changes":       http.Dir(importPathToDir("github.com/shurcooL/home/internal/exp/app/changesapp/_data")),
	"/notifications": http.Dir(importPathToDir("github.com/shurcooL/home/internal/exp/app/notifsapp/_data")),
})

func importPathToDir(importPath string) string {
	p, err := build.Import(importPath, "", build.FindOnly)
	if err != nil {
		log.Fatalln(err)
	}
	return p.Dir
}

// packageWasmFS is an http.FileSystem that contains a single file at root,
// the result of building package ImportPath with GOOS=js and GOARCH=wasm.
type packageWasmFS struct {
	ImportPath string
}

func (p packageWasmFS) Open(name string) (http.File, error) {
	if name != "/" {
		return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrNotExist}
	}

	temp, err := temp()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("go", "build", "-tags=nethttpomithttp2", "-trimpath", "-o", temp, p.ImportPath)
	env := osutil.Environ(os.Environ())
	env.Set("GOOS", "js")
	env.Set("GOARCH", "wasm")
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		os.Remove(temp)
		return nil, fmt.Errorf("%q: %v\n\n%s", cmd.Args, err, out)
	}
	f, err := os.Open(temp)
	if err != nil {
		os.Remove(temp)
		return nil, err
	}
	return tempFile{File: f}, nil
}

// temp creates a new temporary
// file and returns its path.
func temp() (string, error) {
	t, err := ioutil.TempFile("", "")
	if err != nil {
		return "", err
	}
	if err := t.Close(); err != nil {
		os.Remove(t.Name())
		return "", err
	}
	return t.Name(), nil
}

// tempFile wraps a temporary *os.File.
// On Close, the file is closed and removed.
type tempFile struct {
	*os.File
}

func (f tempFile) Close() error {
	f.File.Close()
	os.Remove(f.File.Name())
	return nil
}

// wasmExecFile is an http.FileSystem that contains the
// $(go env GOROOT)/misc/wasm/wasm_exec.js file at root.
type wasmExecFile struct {
	once   sync.Once
	goroot string
	err    error
}

func (f *wasmExecFile) Open(name string) (http.File, error) {
	if name != "/" {
		return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrNotExist}
	}
	f.once.Do(func() { f.goroot, f.err = goroot() })
	if f.err != nil {
		return nil, f.err
	}
	return os.Open(filepath.Join(f.goroot, "misc", "wasm", "wasm_exec.js"))
}

// goroot returns the go env GOROOT value by invoking the go command.
func goroot() (string, error) {
	out, err := exec.Command("go", "env", "-json", "GOROOT").Output()
	if ee := (*exec.ExitError)(nil); errors.As(err, &ee) {
		return "", fmt.Errorf("go command exited unsuccessfully: %v\n%s", ee.ProcessState.String(), ee.Stderr)
	} else if err != nil {
		return "", err
	}
	var env struct{ GOROOT string }
	err = json.Unmarshal(out, &env)
	if err != nil {
		return "", err
	}
	return env.GOROOT, nil
}

// goVersion is the Go 1.x version used during the build.
var goVersion = len(build.Default.ReleaseTags)
