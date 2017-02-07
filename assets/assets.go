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

// Assets contains assets for home.
var Assets = union.New(map[string]http.FileSystem{
	"/assets": gopherjs_http.NewFS(http.Dir(importPathToDir("github.com/shurcooL/home/assets/_data"))),
	//"/octicons": octicons.Assets,
	"/resume.js":  gopherjs_http.Package("github.com/shurcooL/resume/frontend"),
	"/resume.css": vfsutil.File(filepath.Join(importPathToDir("github.com/shurcooL/resume/frontend"), "style.css")),
})
