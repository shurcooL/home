// +build dev

package presentdata

import (
	"go/build"
	"log"
	"net/http"
	"os"

	"github.com/shurcooL/httpfs/filter"
	"github.com/shurcooL/httpfs/union"
)

func importPathToDir(importPath string) string {
	p, err := build.Import(importPath, "", build.FindOnly)
	if err != nil {
		log.Fatalln(err)
	}
	return p.Dir
}

// Assets contains static data for present format.
var Assets = union.New(map[string]http.FileSystem{
	"/static": filter.Keep(
		http.Dir(importPathToDir("golang.org/x/tools/cmd/present/static")),
		func(path string, fi os.FileInfo) bool {
			switch path {
			case "/", "/slides.js", "/styles.css":
				return true
			default:
				return false
			}
		},
	),
	"/templates": filter.Keep(
		http.Dir(importPathToDir("golang.org/x/tools/cmd/present/templates")),
		func(path string, fi os.FileInfo) bool {
			switch path {
			case "/", "/action.tmpl", "/slides.tmpl":
				return true
			default:
				return false
			}
		},
	),
})
