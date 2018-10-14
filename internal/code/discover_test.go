package code

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestDiscover(t *testing.T) {
	want := []*Directory{
		{
			ImportPath:   "dmitri.shuralyov.com/emptyrepo",
			RepoRoot:     "dmitri.shuralyov.com/emptyrepo",
			RepoPackages: 0,
		},
		{
			ImportPath:   "dmitri.shuralyov.com/kebabcase",
			RepoRoot:     "dmitri.shuralyov.com/kebabcase",
			RepoPackages: 1,
			Package: &Package{
				Name:     "kebabcase",
				Synopsis: "Package kebabcase provides a parser for identifier names using kebab-case naming convention.",
				DocHTML: `<p>
Package kebabcase provides a parser for identifier names
using kebab-case naming convention.
</p>
<p>
Reference: <a href="https://en.wikipedia.org/wiki/Naming_convention_(programming)#Multiple-word_identifiers">https://en.wikipedia.org/wiki/Naming_convention_(programming)#Multiple-word_identifiers</a>.
</p>
`,
			},
		},
		{
			ImportPath:   "dmitri.shuralyov.com/scratch",
			RepoRoot:     "dmitri.shuralyov.com/scratch",
			RepoPackages: 4,
			LicenseRoot:  "dmitri.shuralyov.com/scratch",
			Package: &Package{
				Name:     "scratch",
				Synopsis: "Package scratch is used for testing.",
				DocHTML: `<p>
Package scratch is used for testing.
</p>
`,
			},
		},
		{
			ImportPath:   "dmitri.shuralyov.com/scratch/hello",
			RepoRoot:     "dmitri.shuralyov.com/scratch",
			RepoPackages: 4,
			LicenseRoot:  "dmitri.shuralyov.com/scratch",
			Package: &Package{
				Name: "main",
			},
		},
		{
			ImportPath:   "dmitri.shuralyov.com/scratch/image",
			RepoRoot:     "dmitri.shuralyov.com/scratch",
			RepoPackages: 4,
			LicenseRoot:  "dmitri.shuralyov.com/scratch/image",
		},
		{
			ImportPath:   "dmitri.shuralyov.com/scratch/image/jpeg",
			RepoRoot:     "dmitri.shuralyov.com/scratch",
			RepoPackages: 4,
			LicenseRoot:  "dmitri.shuralyov.com/scratch/image",
			Package: &Package{
				Name:     "jpeg",
				Synopsis: "Package jpeg implements a tiny subset of a JPEG image decoder and encoder.",
				DocHTML: `<p>
Package jpeg implements a tiny subset of a JPEG image decoder and encoder.
</p>
<p>
JPEG is defined in ITU-T T.81: <a href="http://www.w3.org/Graphics/JPEG/itu-t81.pdf">http://www.w3.org/Graphics/JPEG/itu-t81.pdf</a>.
</p>
`,
			},
		},
		{
			ImportPath:   "dmitri.shuralyov.com/scratch/image/png",
			RepoRoot:     "dmitri.shuralyov.com/scratch",
			RepoPackages: 4,
			LicenseRoot:  "dmitri.shuralyov.com/scratch/image",
			Package: &Package{
				Name:     "png",
				Synopsis: "Package png implements a tiny subset of a PNG image decoder and encoder.",
				DocHTML: `<p>
Package png implements a tiny subset of a PNG image decoder and encoder.
</p>
<p>
The PNG specification is at <a href="http://www.w3.org/TR/PNG/">http://www.w3.org/TR/PNG/</a>.
</p>
`,
			},
		},
	}
	got, _, err := discover(filepath.Join("testdata", "repositories"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Error("not equal")
	}
}
