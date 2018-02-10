package code_test

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/shurcooL/home/internal/code"
)

func TestDiscover(t *testing.T) {
	want := []code.Directory{
		{
			ImportPath: "dmitri.shuralyov.com/emptyrepo",
			RepoRoot:   "dmitri.shuralyov.com/emptyrepo",
		},
		{
			ImportPath: "dmitri.shuralyov.com/kebabcase",
			RepoRoot:   "dmitri.shuralyov.com/kebabcase",
			Package: &code.Package{
				Name: "kebabcase",
				Doc:  "Package kebabcase provides a parser for identifier names using kebab-case naming convention.",
			},
		},
		{
			ImportPath: "dmitri.shuralyov.com/scratch",
			RepoRoot:   "dmitri.shuralyov.com/scratch",
			Package: &code.Package{
				Name: "scratch",
				Doc:  "Package scratch is used for testing.",
			},
		},
		{
			ImportPath: "dmitri.shuralyov.com/scratch/hello",
			RepoRoot:   "dmitri.shuralyov.com/scratch",
			Package: &code.Package{
				Name: "main",
			},
		},
		{
			ImportPath: "dmitri.shuralyov.com/scratch/image",
			RepoRoot:   "dmitri.shuralyov.com/scratch",
		},
		{
			ImportPath: "dmitri.shuralyov.com/scratch/image/jpeg",
			RepoRoot:   "dmitri.shuralyov.com/scratch",
			Package: &code.Package{
				Name: "jpeg",
				Doc:  "Package jpeg implements a tiny subset of a JPEG image decoder and encoder.",
			},
		},
		{
			ImportPath: "dmitri.shuralyov.com/scratch/image/png",
			RepoRoot:   "dmitri.shuralyov.com/scratch",
			Package: &code.Package{
				Name: "png",
				Doc:  "Package png implements a tiny subset of a PNG image decoder and encoder.",
			},
		},
	}
	got, err := code.Discover(filepath.Join("testdata", "repositories"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Sorted, want) {
		t.Error("not equal")
	}
}
