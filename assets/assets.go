// +build dev

package assets

import (
	"go/build"
	"log"
	"net/http"
	"path/filepath"

	"github.com/shurcooL/go/gopherjs_http"
	"github.com/shurcooL/httpfs/union"
	"github.com/shurcooL/httpfs/vfsutil"
)

func importPathToDir(importPath string) string {
	p, err := build.Import(importPath, "", build.FindOnly)
	if err != nil {
		log.Fatalln(err)
	}
	return p.Dir
}

var Assets = union.New(map[string]http.FileSystem{
	//"/octicons": octicons.Assets,
	"/resume.js":  gopherjs_http.Package("github.com/shurcooL/resume"),
	"/resume.css": vfsutil.File(filepath.Join(importPathToDir("github.com/shurcooL/resume"), "style.css")),
})
