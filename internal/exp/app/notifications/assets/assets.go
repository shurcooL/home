// +build dev

package assets

import (
	"fmt"
	"go/build"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/shurcooL/go/osutil"
	"github.com/shurcooL/httpfs/union"
	"github.com/shurcooL/httpfs/vfsutil"
)

// Assets contains assets for notifications app.
var Assets = union.New(map[string]http.FileSystem{
	"/wasm_exec.js":  vfsutil.File(filepath.Join(importPathToDir("github.com/shurcooL/home/internal/exp/app/notifications/_data"), "wasm_exec.js")),
	"/thread.css":    vfsutil.File(filepath.Join(importPathToDir("github.com/shurcooL/home/internal/exp/app/notifications/_data"), "thread.css")),
	"/stream.css":    vfsutil.File(filepath.Join(importPathToDir("github.com/shurcooL/home/internal/exp/app/notifications/_data"), "stream.css")),
	"/frontend.wasm": packageWasmFS{"github.com/shurcooL/home/internal/exp/app/notifications/frontend"},
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
	cmd := exec.Command("go", "build", "-o", temp, p.ImportPath)
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
