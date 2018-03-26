package code_test

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/shurcooL/home/internal/code"
)

func TestDiscover(t *testing.T) {
	want := []*code.Directory{
		{
			ImportPath: "dmitri.shuralyov.com/emptyrepo",
			RepoRoot:   "dmitri.shuralyov.com/emptyrepo",
		},
		{
			ImportPath: "dmitri.shuralyov.com/kebabcase",
			RepoRoot:   "dmitri.shuralyov.com/kebabcase",
			Package: &code.Package{
				Name:     "kebabcase",
				Synopsis: "Package kebabcase provides a parser for identifier names using kebab-case naming convention.",
				DocHTML: `<p>
Package kebabcase provides a parser for identifier names using kebab-case naming convention.
</p>
<p>
Reference: <a href="https://en.wikipedia.org/wiki/Naming_convention_(programming)#Multiple-word_identifiers">https://en.wikipedia.org/wiki/Naming_convention_(programming)#Multiple-word_identifiers</a>.</p>
`,
			},
		},
		{
			ImportPath:  "dmitri.shuralyov.com/scratch",
			RepoRoot:    "dmitri.shuralyov.com/scratch",
			LicenseRoot: "dmitri.shuralyov.com/scratch",
			Package: &code.Package{
				Name:     "scratch",
				Synopsis: "Package scratch is used for testing.",
				DocHTML:  "<p>\nPackage scratch is used for testing.</p>\n",
			},
		},
		{
			ImportPath:  "dmitri.shuralyov.com/scratch/hello",
			RepoRoot:    "dmitri.shuralyov.com/scratch",
			LicenseRoot: "dmitri.shuralyov.com/scratch",
			Package: &code.Package{
				Name: "main",
			},
		},
		{
			ImportPath:  "dmitri.shuralyov.com/scratch/image",
			RepoRoot:    "dmitri.shuralyov.com/scratch",
			LicenseRoot: "dmitri.shuralyov.com/scratch/image",
		},
		{
			ImportPath:  "dmitri.shuralyov.com/scratch/image/jpeg",
			RepoRoot:    "dmitri.shuralyov.com/scratch",
			LicenseRoot: "dmitri.shuralyov.com/scratch/image",
			Package: &code.Package{
				Name:     "jpeg",
				Synopsis: "Package jpeg implements a tiny subset of a JPEG image decoder and encoder.",
				DocHTML: `<p>
Package jpeg implements a tiny subset of a JPEG image decoder and encoder.
</p>
<p>
JPEG is defined in ITU-T T.81: <a href="http://www.w3.org/Graphics/JPEG/itu-t81.pdf">http://www.w3.org/Graphics/JPEG/itu-t81.pdf</a>.</p>
`,
			},
		},
		{
			ImportPath:  "dmitri.shuralyov.com/scratch/image/png",
			RepoRoot:    "dmitri.shuralyov.com/scratch",
			LicenseRoot: "dmitri.shuralyov.com/scratch/image",
			Package: &code.Package{
				Name:     "png",
				Synopsis: "Package png implements a tiny subset of a PNG image decoder and encoder.",
				DocHTML: `<p>
Package png implements a tiny subset of a PNG image decoder and encoder.
</p>
<p>
The PNG specification is at <a href="http://www.w3.org/TR/PNG/">http://www.w3.org/TR/PNG/</a>.</p>
`,
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
