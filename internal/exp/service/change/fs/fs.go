// Package fs will implement change.Service using a virtual filesystem,
// once change.Service API is finalized. Until then, it uses hardcoded
// mock data to aid development and evaluation of change.Service API.
package fs

import (
	"context"
	"fmt"
	"os"
	"time"

	"dmitri.shuralyov.com/state"
	"github.com/shurcooL/home/internal/exp/service/change"
	"github.com/shurcooL/reactions"
	"github.com/shurcooL/users"
)

type Service struct {
	// Reactions, if not nil, is temporarily used as a place to store reactions.
	Reactions reactions.Service
}

var s = struct {
	changes map[string][]struct {
		change.Change
		Timeline []interface{}
		Commits  []change.Commit
		Diffs    map[string][]byte // Key is commit SHA. "all" is diff of all commits combined.
	}
}{
	changes: map[string][]struct {
		change.Change
		Timeline []interface{}
		Commits  []change.Commit
		Diffs    map[string][]byte
	}{
		"dmitri.shuralyov.com/font/woff2": {{
			Change: change.Change{
				ID:           1,
				State:        state.ChangeMerged,
				Title:        "Initial implementation of woff2.",
				Labels:       nil,
				Author:       dmitshur,
				CreatedAt:    time.Date(2018, 2, 12, 0, 9, 19, 621031866, time.UTC),
				Replies:      1,
				Commits:      3,
				ChangedFiles: 5,
			},
			Timeline: []interface{}{
				change.Comment{
					ID:        "0",
					User:      dmitshur,
					CreatedAt: time.Date(2018, 2, 12, 0, 9, 19, 621031866, time.UTC),
					Body: `Add initial parser implementation.

This is an initial implementation of a parser for the WOFF2 font
packaging format.

It is incomplete; further work will come later. The scope for this
milestone was to be able to parse .woff2 files for the needs of the
github.com/ConradIrwin/font/sfnt package.

At this time, the API is very low level and maps directly to the binary
format of the file, as described in [its specification](https://www.w3.org/TR/WOFF2/). This API is in
early development and is expected to change as further progress is made.

It successfully parses some Go font family .woff2 files that were
generated using the https://github.com/google/woff2 encoder
from the Go font source .ttf files located at
https://go.googlesource.com/image/+/master/font/gofont/ttfs/.

Add basic test coverage.

Helps https://github.com/ConradIrwin/font/issues/1.

For convenience, a ` + "`" + `godoc` + "`" + ` view of this change can be seen [here](https://redpen.io/rk9a75c358f45654a8).`,
				},
				change.Review{
					ID:        "1",
					User:      dmitshur,
					CreatedAt: time.Date(2018, 2, 20, 21, 49, 35, 536092503, time.UTC),
					State:     state.ReviewPlus2,
					Body:      "There have been some out-of-band review comments that I've addressed. This will do for the initial version.\n\nLGTM.",
				},
				change.TimelineItem{
					Actor:     dmitshur,
					CreatedAt: time.Date(2018, 2, 20, 21, 57, 47, 537746502, time.UTC),
					Payload: change.MergedEvent{
						CommitID:      "957792cbbdabb084d484a7dcfd1e5b1a739a0ced",
						CommitHTMLURL: "https://dmitri.shuralyov.com/font/woff2/...$commit/957792cbbdabb084d484a7dcfd1e5b1a739a0ced",
						RefName:       "master",
					},
				},
			},
			Commits: []change.Commit{{
				SHA: "d2568fb6f10921b2d0c84d58bad14b2fadb88aa7",
				Message: `Add initial parser implementation.

This is an initial implementation of a parser for the WOFF2 font
packaging format.

It is incomplete; further work will come later. The scope for this
milestone was to be able to parse .woff2 files for the needs of the
github.com/ConradIrwin/font/sfnt package.

At this time, the API is very low level and maps directly to the binary
format of the file, as described in its specification. This API is in
early development and is expected to change as further progress is made.

It successfully parses some Go font family .woff2 files that were
generated using the https://github.com/google/woff2 encoder
from the Go font source .ttf files located at
https://go.googlesource.com/image/+/master/font/gofont/ttfs/.

Add basic test coverage.

Helps https://github.com/ConradIrwin/font/issues/1.`,
				Author:     dmitshur,
				AuthorTime: time.Date(2018, 2, 11, 20, 10, 28, 0, time.UTC),
			}, {
				SHA: "61339d441b319cd6ca35d952522f86cc42ad4b6e",
				Message: `Update test for new API.

The API had changed earlier, the test wasn't updated for it. This
change fixes that, allowing tests to pass.`,
				Author:     dmitshur,
				AuthorTime: time.Date(2018, 2, 12, 20, 22, 29, 674711268, time.UTC),
			}, {
				SHA: "3b528c98b05508322be465a207f5ffd8258b8a96",
				Message: `Add comment describing null-padding check.

Also add a TODO comment to improve this check. It's not compliant with
spec as is. Addressing this will be a part of future changes.`,
				Author:     dmitshur,
				AuthorTime: time.Date(2018, 2, 20, 21, 39, 17, 660912242, time.UTC),
			}},
			Diffs: map[string][]byte{
				"all": []byte(diffAll),
				"d2568fb6f10921b2d0c84d58bad14b2fadb88aa7": []byte(diffCommit1),
				"61339d441b319cd6ca35d952522f86cc42ad4b6e": []byte(diffCommit2),
				"3b528c98b05508322be465a207f5ffd8258b8a96": []byte(diffCommit3),
			},
		}},
		"dmitri.shuralyov.com/gpu/mtl": {{
			Change: change.Change{
				ID:           1,
				State:        state.ChangeMerged,
				Title:        "Add minimal API to support interactive rendering in a window.",
				Labels:       nil,
				Author:       dmitshur,
				CreatedAt:    time.Date(2018, 10, 17, 2, 9, 9, 583606000, time.UTC),
				Replies:      1,
				Commits:      4,
				ChangedFiles: 10,
			},
			Timeline: []interface{}{
				change.Comment{
					ID:        "0",
					User:      dmitshur,
					CreatedAt: time.Date(2018, 10, 17, 2, 9, 9, 583606000, time.UTC),
					Edited: &change.Edited{
						By: dmitshur,
						At: time.Date(2018, 10, 21, 17, 25, 56, 868164000, time.UTC),
					},
					Body: `The goal of this change is to make it possible to use package mtl
to render to a window at interactive framerates (e.g., at 60 FPS,
assuming a 60 Hz display with vsync on). It adds the minimal API
that is needed.

A new movingtriangle example is added as a demonstration of this
functionality. It opens a window and renders a triangle that follows
the mouse cursor.

Much of the needed API comes from Core Animation, AppKit frameworks,
rather than Metal. Avoid adding that to mtl package; instead create
separate packages. For now, they are hidden in internal to avoid
committing to a public API and import path. After gaining more
confidence in the approach, they can be factored out and made public.`,
				},
				change.TimelineItem{
					Actor:     dmitshur,
					CreatedAt: time.Date(2018, 10, 18, 0, 54, 37, 790845000, time.UTC),
					Payload: change.RenamedEvent{
						From: "WIP: Add minimal API to support rendering to a window at 60 FPS.",
						To:   "WIP: Add minimal API to support interactive rendering in a window.",
					},
				},
				change.TimelineItem{
					Actor:     dmitshur,
					CreatedAt: time.Date(2018, 10, 21, 17, 25, 56, 868164000, time.UTC),
					Payload: change.RenamedEvent{
						From: "WIP: Add minimal API to support interactive rendering in a window.",
						To:   "Add minimal API to support interactive rendering in a window.",
					},
				},
				change.Review{
					ID:        "1",
					User:      hajimehoshi,
					CreatedAt: time.Date(2018, 10, 23, 3, 22, 57, 249312000, time.UTC),
					State:     state.ReviewPlus2,
					Body:      "I did a rough review and could not find a critical issue. 🙂",
				},
				change.TimelineItem{
					Actor:     dmitshur,
					CreatedAt: time.Date(2018, 10, 23, 3, 32, 2, 951463000, time.UTC),
					Payload: change.MergedEvent{
						CommitID:      "c4eb07ba2d711bc78bcd2606dd587d9267a61aa5",
						CommitHTMLURL: "https://dmitri.shuralyov.com/gpu/mtl/...$commit/c4eb07ba2d711bc78bcd2606dd587d9267a61aa5",
						RefName:       "master",
					},
				},
			},
			Commits: []change.Commit{{
				SHA: "fc76fa8984fb4a28ff383895e55e635e06bd32f0",
				Message: `WIP: Add minimal API to support rendering to a window at 60 FPS.

The goal of this change is to make it possible to use package mtl
to render to a window at 60 FPS. It tries to add the minimum viable
API that is needed.

A new movingtriangle example is added as a demonstration of this
functionality. It renders a triangle that follows the mouse cursor.

TODO: A lot of the newly added API comes from Core Animation, AppKit
frameworks, rather than Metal. As a result, they likely do not belong
in this package and should be factored out.`,
				Author:     dmitshur,
				AuthorTime: time.Date(2018, 10, 17, 2, 9, 9, 583606000, time.UTC),
			}, {
				SHA:        "da15ef360afe80e10274aecb1b3e1390144fde3c",
				Message:    "Add missing reference links.",
				Author:     dmitshur,
				AuthorTime: time.Date(2018, 10, 21, 3, 16, 43, 311370000, time.UTC),
			}, {
				SHA:        "d146c0ceb29d388d838337b3951f16dca31602e1",
				Message:    "Factor out Core Animation, Cocoa APIs into separate internal packages.",
				Author:     dmitshur,
				AuthorTime: time.Date(2018, 10, 21, 17, 22, 10, 566116000, time.UTC),
			}, {
				SHA: "c4eb07ba2d711bc78bcd2606dd587d9267a61aa5",
				Message: `Move internal/{ca,ns} into example/movingtriangle.

Also make minor tweaks to the documentation to make it more accurate.`,
				Author:     dmitshur,
				AuthorTime: time.Date(2018, 10, 23, 3, 20, 25, 631677000, time.UTC),
			}},
			Diffs: map[string][]byte{
				"all": []byte(diffMtlAll),
				"fc76fa8984fb4a28ff383895e55e635e06bd32f0": []byte(diffMtlCommit1),
				"da15ef360afe80e10274aecb1b3e1390144fde3c": []byte(diffMtlCommit2),
				"d146c0ceb29d388d838337b3951f16dca31602e1": []byte(diffMtlCommit3),
				"c4eb07ba2d711bc78bcd2606dd587d9267a61aa5": []byte(diffMtlCommit4),
			},
		}},
	},
}

// List changes.
func (*Service) List(ctx context.Context, repo string, opt change.ListOptions) ([]change.Change, error) {
	var counts func(s state.Change) bool
	switch opt.Filter {
	case change.FilterOpen:
		counts = func(s state.Change) bool { return s == state.ChangeOpen }
	case change.FilterClosedMerged:
		counts = func(s state.Change) bool { return s == state.ChangeClosed || s == state.ChangeMerged }
	case change.FilterAll:
		counts = func(s state.Change) bool { return true }
	default:
		// TODO: Map to 400 Bad Request HTTP error.
		return nil, fmt.Errorf("invalid change.ListOptions.Filter value: %q", opt.Filter)
	}
	var cs []change.Change
	for _, c := range s.changes[repo] {
		if !counts(c.State) {
			continue
		}
		cs = append(cs, c.Change)
	}
	return cs, nil
}

// Count changes.
func (*Service) Count(ctx context.Context, repo string, opt change.ListOptions) (uint64, error) {
	var counts func(s state.Change) bool
	switch opt.Filter {
	case change.FilterOpen:
		counts = func(s state.Change) bool { return s == state.ChangeOpen }
	case change.FilterClosedMerged:
		counts = func(s state.Change) bool { return s == state.ChangeClosed || s == state.ChangeMerged }
	case change.FilterAll:
		counts = func(s state.Change) bool { return true }
	default:
		// TODO: Map to 400 Bad Request HTTP error.
		return 0, fmt.Errorf("invalid change.ListOptions.Filter value: %q", opt.Filter)
	}
	var count uint64
	for _, c := range s.changes[repo] {
		if !counts(c.State) {
			continue
		}
		count++
	}
	return count, nil
}

// Get a change.
func (*Service) Get(ctx context.Context, repo string, id uint64) (change.Change, error) {
	if !hasChange(repo, id) {
		return change.Change{}, os.ErrNotExist
	}
	return s.changes[repo][id-1].Change, nil
}

// ListTimeline lists timeline items (change.Comment, change.Review, change.TimelineItem) for specified change id.
func (svc *Service) ListTimeline(ctx context.Context, repo string, id uint64, opt *change.ListTimelineOptions) ([]interface{}, error) {
	if !hasChange(repo, id) {
		return nil, os.ErrNotExist
	}
	if svc.Reactions == nil {
		return s.changes[repo][id-1].Timeline, nil
	}
	reactions, err := svc.Reactions.List(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("ListTimeline: Reactions.List: %v", err)
	}
	timeline := make([]interface{}, len(s.changes[repo][id-1].Timeline))
	copy(timeline, s.changes[repo][id-1].Timeline)
	switch {
	case repo == "dmitri.shuralyov.com/font/woff2" && id == 1:
		{
			t := timeline[0].(change.Comment)
			t.Reactions = reactions[t.ID]
			timeline[0] = t
		}
		{
			t := timeline[1].(change.Review)
			t.Reactions = reactions[t.ID]
			timeline[1] = t
		}
	case repo == "dmitri.shuralyov.com/gpu/mtl" && id == 1:
		{
			t := timeline[0].(change.Comment)
			t.Reactions = reactions[t.ID]
			timeline[0] = t
		}
		{
			t := timeline[3].(change.Review)
			t.Reactions = reactions[t.ID]
			timeline[3] = t
		}
	}
	return timeline, nil
}

// ListCommits lists change commits.
func (*Service) ListCommits(ctx context.Context, repo string, id uint64) ([]change.Commit, error) {
	if !hasChange(repo, id) {
		return nil, os.ErrNotExist
	}
	return s.changes[repo][id-1].Commits, nil
}

// Get a change diff.
func (*Service) GetDiff(ctx context.Context, repo string, id uint64, opt *change.GetDiffOptions) ([]byte, error) {
	if !hasChange(repo, id) {
		return nil, os.ErrNotExist
	}
	switch opt {
	case nil:
		return s.changes[repo][id-1].Diffs["all"], nil
	default:
		return s.changes[repo][id-1].Diffs[opt.Commit], nil
	}
}

func (s *Service) EditComment(ctx context.Context, repo string, id uint64, cr change.CommentRequest) (change.Comment, error) {
	if !hasChange(repo, id) {
		return change.Comment{}, os.ErrNotExist
	}
	if s.Reactions == nil {
		return change.Comment{}, fmt.Errorf("no place on backend to store reactions")
	}
	var comment change.Comment
	if cr.Reaction != nil {
		reactions, err := s.Reactions.Toggle(ctx, repo, cr.ID, reactions.ToggleRequest{Reaction: *cr.Reaction})
		if err != nil {
			return change.Comment{}, err
		}
		comment.Reactions = reactions
	}
	return comment, nil
}

func hasChange(repo string, id uint64) bool {
	return 1 <= id && id <= uint64(len(s.changes[repo]))
}

// fsChangeThreadType is the notification thread type for changes stored in a virtual filesystem.
const fsChangeThreadType = "Change"

// ThreadType returns the notification thread type for this service.
func (*Service) ThreadType(repo string) string { return fsChangeThreadType }

var (
	dmitshur = users.User{
		UserSpec: users.UserSpec{
			ID:     1924134,
			Domain: "github.com",
		},
		Login:     "dmitshur",
		Name:      "Dmitri Shuralyov",
		Email:     "dmitri@shuralyov.com",
		AvatarURL: "https://dmitri.shuralyov.com/avatar.jpg",
		HTMLURL:   "https://dmitri.shuralyov.com",
		SiteAdmin: true,
	}

	hajimehoshi = users.User{
		UserSpec: users.UserSpec{
			ID:     16950,
			Domain: "github.com",
		},
		Login:     "hajimehoshi",
		AvatarURL: "https://avatars2.githubusercontent.com/u/16950?v=4",
		HTMLURL:   "https://github.com/hajimehoshi",
	}
)

const diffAll = `diff --git a/Commit Message b/Commit Message
new file mode 100644
index 0000000..dfb31fe
--- /dev/null
+++ b/Commit Message
@@ -0,0 +1,27 @@
+Parent:     e9561aed (Initial commit.)
+Author:     Dmitri Shuralyov <dmitri@shuralyov.com>
+AuthorDate: Sun Feb 11 15:10:28 2018 -0500
+Commit:     Dmitri Shuralyov <dmitri@shuralyov.com>
+CommitDate: Sun Feb 11 15:19:24 2018 -0500
+
+Add initial parser implementation.
+
+This is an initial implementation of a parser for the WOFF2 font
+packaging format.
+
+It is incomplete; further work will come later. The scope for this
+milestone was to be able to parse .woff2 files for the needs of the
+github.com/ConradIrwin/font/sfnt package.
+
+At this time, the API is very low level and maps directly to the binary
+format of the file, as described in its specification. This API is in
+early development and is expected to change as further progress is made.
+
+It successfully parses some Go font family .woff2 files that were
+generated using the https://github.com/google/woff2 encoder
+from the Go font source .ttf files located at
+https://go.googlesource.com/image/+/master/font/gofont/ttfs/.
+
+Add basic test coverage.
+
+Helps https://github.com/ConradIrwin/font/issues/1.
diff --git a/doc.go b/doc.go
index fd35888..a751214 100644
--- a/doc.go
+++ b/doc.go
@@ -2,5 +2,3 @@
 //
 // The WOFF2 font packaging format is specified at https://www.w3.org/TR/WOFF2/.
 package woff2
-
-// TODO: Implement.
diff --git a/parse.go b/parse.go
new file mode 100644
index 0000000..498a4a8
--- /dev/null
+++ b/parse.go
@@ -0,0 +1,438 @@
+package woff2
+
+import (
+	"bytes"
+	"encoding/binary"
+	"fmt"
+	"io"
+
+	"github.com/dsnet/compress/brotli"
+)
+
+// File represents a parsed WOFF2 file.
+type File struct {
+	Header         Header
+	TableDirectory TableDirectory
+	// CollectionDirectory is present only if the font is a collection,
+	// as reported by Header.IsCollection.
+	CollectionDirectory *CollectionDirectory
+
+	// FontData is the concatenation of data for each table in the font.
+	// During storage, it's compressed using Brotli.
+	FontData []byte
+
+	ExtendedMetadata *ExtendedMetadata
+
+	// PrivateData is an optional block of private data for the font designer,
+	// foundry, or vendor to use.
+	PrivateData []byte
+}
+
+// Parse parses the WOFF2 data from r.
+func Parse(r io.Reader) (File, error) {
+	hdr, err := parseHeader(r)
+	if err != nil {
+		return File{}, err
+	}
+	td, err := parseTableDirectory(r, hdr)
+	if err != nil {
+		return File{}, err
+	}
+	cd, err := parseCollectionDirectory(r, hdr)
+	if err != nil {
+		return File{}, err
+	}
+	fd, err := parseCompressedFontData(r, hdr, td)
+	if err != nil {
+		return File{}, err
+	}
+	em, err := parseExtendedMetadata(r, hdr)
+	if err != nil {
+		return File{}, err
+	}
+	pd, err := parsePrivateData(r, hdr)
+	if err != nil {
+		return File{}, err
+	}
+
+	// Check for padding with a maximum of three null bytes.
+	// TODO: This check needs to be moved to Extended Metadata and Private Data blocks,
+	//       and made more precise (i.e., the beginning of those blocks must be 4-byte aligned, etc.).
+	n, err := io.Copy(discardZeroes{}, r)
+	if err != nil {
+		return File{}, fmt.Errorf("Parse: %v", err)
+	}
+	if n > 3 {
+		return File{}, fmt.Errorf("Parse: %d bytes left remaining, want no more than 3", n)
+	}
+
+	return File{
+		Header:              hdr,
+		TableDirectory:      td,
+		CollectionDirectory: cd,
+		FontData:            fd,
+		ExtendedMetadata:    em,
+		PrivateData:         pd,
+	}, nil
+}
+
+// discardZeroes is an io.Writer that returns an error if any non-zero bytes are written to it.
+type discardZeroes struct{}
+
+func (discardZeroes) Write(p []byte) (int, error) {
+	for _, b := range p {
+		if b != 0 {
+			return 0, fmt.Errorf("encountered non-zero byte %d", b)
+		}
+	}
+	return len(p), nil
+}
+
+// Header is the file header with basic font type and version,
+// along with offsets to metadata and private data blocks.
+type Header struct {
+	Signature           uint32 // The identifying signature; must be 0x774F4632 ('wOF2').
+	Flavor              uint32 // The "sfnt version" of the input font.
+	Length              uint32 // Total size of the WOFF file.
+	NumTables           uint16 // Number of entries in directory of font tables.
+	Reserved            uint16 // Reserved; set to 0.
+	TotalSfntSize       uint32 // Total size needed for the uncompressed font data, including the sfnt header, directory, and font tables (including padding).
+	TotalCompressedSize uint32 // Total length of the compressed data block.
+	MajorVersion        uint16 // Major version of the WOFF file.
+	MinorVersion        uint16 // Minor version of the WOFF file.
+	MetaOffset          uint32 // Offset to metadata block, from beginning of WOFF file.
+	MetaLength          uint32 // Length of compressed metadata block.
+	MetaOrigLength      uint32 // Uncompressed size of metadata block.
+	PrivOffset          uint32 // Offset to private data block, from beginning of WOFF file.
+	PrivLength          uint32 // Length of private data block.
+}
+
+func parseHeader(r io.Reader) (Header, error) {
+	var hdr Header
+	err := binary.Read(r, order, &hdr)
+	if err != nil {
+		return Header{}, err
+	}
+	if hdr.Signature != signature {
+		return Header{}, fmt.Errorf("parseHeader: invalid signature: got %#08x, want %#08x", hdr.Signature, signature)
+	}
+	return hdr, nil
+}
+
+// IsCollection reports whether this is a font collection, i.e.,
+// if the value of Flavor field is set to the TrueType Collection flavor 'ttcf'.
+func (hdr Header) IsCollection() bool {
+	return hdr.Flavor == ttcfFlavor
+}
+
+// TableDirectory is the directory of font tables, containing size and other info.
+type TableDirectory []TableDirectoryEntry
+
+func parseTableDirectory(r io.Reader, hdr Header) (TableDirectory, error) {
+	var td TableDirectory
+	for i := 0; i < int(hdr.NumTables); i++ {
+		var e TableDirectoryEntry
+
+		err := readU8(r, &e.Flags)
+		if err != nil {
+			return nil, err
+		}
+		if e.Flags&0x3f == 0x3f {
+			e.Tag = new(uint32)
+			err := readU32(r, e.Tag)
+			if err != nil {
+				return nil, err
+			}
+		}
+		err = readBase128(r, &e.OrigLength)
+		if err != nil {
+			return nil, err
+		}
+
+		switch tag, transformVersion := e.tag(), e.transformVersion(); tag {
+		case glyfTable, locaTable:
+			// 0 means transform for glyf/loca tables.
+			if transformVersion == 0 {
+				e.TransformLength = new(uint32)
+				err := readBase128(r, e.TransformLength)
+				if err != nil {
+					return nil, err
+				}
+
+				// The transform length of the transformed loca table MUST always be zero.
+				if tag == locaTable && *e.TransformLength != 0 {
+					return nil, fmt.Errorf("parseTableDirectory: 'loca' table has non-zero transform length %d", *e.TransformLength)
+				}
+			}
+		default:
+			// Non-0 means transform for other tables.
+			if transformVersion != 0 {
+				e.TransformLength = new(uint32)
+				err := readBase128(r, e.TransformLength)
+				if err != nil {
+					return nil, err
+				}
+			}
+		}
+
+		td = append(td, e)
+	}
+	return td, nil
+}
+
+// Table is a high-level representation of a table.
+type Table struct {
+	Tag    uint32
+	Offset int
+	Length int
+}
+
+// Tables returns the derived high-level information
+// about the tables in the table directory.
+func (td TableDirectory) Tables() []Table {
+	var ts []Table
+	var offset int
+	for _, t := range td {
+		length := int(t.length())
+		ts = append(ts, Table{
+			Tag:    t.tag(),
+			Offset: offset,
+			Length: length,
+		})
+		offset += length
+	}
+	return ts
+}
+
+// uncompressedSize computes the total uncompressed size
+// of the tables in the table directory.
+func (td TableDirectory) uncompressedSize() int64 {
+	var n int64
+	for _, t := range td {
+		n += int64(t.length())
+	}
+	return n
+}
+
+// TableDirectoryEntry is a table directory entry.
+type TableDirectoryEntry struct {
+	Flags           uint8   // Table type and flags.
+	Tag             *uint32 // 4-byte tag (optional).
+	OrigLength      uint32  // Length of original table.
+	TransformLength *uint32 // Transformed length (optional).
+}
+
+func (e TableDirectoryEntry) tag() uint32 {
+	switch e.Tag {
+	case nil:
+		return knownTableTags[e.Flags&0x3f] // Bits [0..5].
+	default:
+		return *e.Tag
+	}
+}
+
+func (e TableDirectoryEntry) transformVersion() uint8 {
+	return e.Flags >> 6 // Bits [6..7].
+}
+
+func (e TableDirectoryEntry) length() uint32 {
+	switch e.TransformLength {
+	case nil:
+		return e.OrigLength
+	default:
+		return *e.TransformLength
+	}
+}
+
+// CollectionDirectory is an optional table containing the font fragment descriptions
+// of font collection entries.
+type CollectionDirectory struct {
+	Header  CollectionHeader
+	Entries []CollectionFontEntry
+}
+
+// CollectionHeader is a part of CollectionDirectory.
+type CollectionHeader struct {
+	Version  uint32
+	NumFonts uint16
+}
+
+// CollectionFontEntry represents a CollectionFontEntry record.
+type CollectionFontEntry struct {
+	NumTables    uint16   // The number of tables in this font.
+	Flavor       uint32   // The "sfnt version" of the font.
+	TableIndices []uint16 // The indicies identifying an entry in the Table Directory for each table in this font.
+}
+
+func parseCollectionDirectory(r io.Reader, hdr Header) (*CollectionDirectory, error) {
+	// CollectionDirectory is present only if the input font is a collection.
+	if !hdr.IsCollection() {
+		return nil, nil
+	}
+
+	var cd CollectionDirectory
+	err := readU32(r, &cd.Header.Version)
+	if err != nil {
+		return nil, err
+	}
+	err = read255UShort(r, &cd.Header.NumFonts)
+	if err != nil {
+		return nil, err
+	}
+	for i := 0; i < int(cd.Header.NumFonts); i++ {
+		var e CollectionFontEntry
+
+		err := read255UShort(r, &e.NumTables)
+		if err != nil {
+			return nil, err
+		}
+		err = readU32(r, &e.Flavor)
+		if err != nil {
+			return nil, err
+		}
+		for j := 0; j < int(e.NumTables); j++ {
+			var tableIndex uint16
+			err := read255UShort(r, &tableIndex)
+			if err != nil {
+				return nil, err
+			}
+			if tableIndex >= hdr.NumTables {
+				return nil, fmt.Errorf("parseCollectionDirectory: tableIndex >= hdr.NumTables")
+			}
+			e.TableIndices = append(e.TableIndices, tableIndex)
+		}
+
+		cd.Entries = append(cd.Entries, e)
+	}
+	return &cd, nil
+}
+
+func parseCompressedFontData(r io.Reader, hdr Header, td TableDirectory) ([]byte, error) {
+	// Compressed font data.
+	br, err := brotli.NewReader(io.LimitReader(r, int64(hdr.TotalCompressedSize)), nil)
+	//br, err := brotli.NewReader(&exactReader{R: r, N: int64(hdr.TotalCompressedSize)}, nil)
+	if err != nil {
+		return nil, err
+	}
+	var buf bytes.Buffer
+	n, err := io.Copy(&buf, br)
+	if err != nil {
+		return nil, fmt.Errorf("parseCompressedFontData: io.Copy: %v", err)
+	}
+	err = br.Close()
+	if err != nil {
+		return nil, fmt.Errorf("parseCompressedFontData: br.Close: %v", err)
+	}
+	if uncompressedSize := td.uncompressedSize(); n != uncompressedSize {
+		return nil, fmt.Errorf("parseCompressedFontData: unexpected size of uncompressed data: got %d, want %d", n, uncompressedSize)
+	}
+	return buf.Bytes(), nil
+}
+
+// ExtendedMetadata is an optional block of extended metadata,
+// represented in XML format and compressed for storage in the WOFF2 file.
+type ExtendedMetadata struct{}
+
+func parseExtendedMetadata(r io.Reader, hdr Header) (*ExtendedMetadata, error) {
+	if hdr.MetaLength == 0 {
+		return nil, nil
+	}
+	return nil, fmt.Errorf("parseExtendedMetadata: not implemented")
+}
+
+func parsePrivateData(r io.Reader, hdr Header) ([]byte, error) {
+	if hdr.PrivLength == 0 {
+		return nil, nil
+	}
+	return nil, fmt.Errorf("parsePrivateData: not implemented")
+}
+
+// readU8 reads a UInt8 value.
+func readU8(r io.Reader, v *uint8) error {
+	return binary.Read(r, order, v)
+}
+
+// readU16 reads a UInt16 value.
+func readU16(r io.Reader, v *uint16) error {
+	return binary.Read(r, order, v)
+}
+
+// readU32 reads a UInt32 value.
+func readU32(r io.Reader, v *uint32) error {
+	return binary.Read(r, order, v)
+}
+
+// readBase128 reads a UIntBase128 value.
+func readBase128(r io.Reader, v *uint32) error {
+	var accum uint32
+	for i := 0; i < 5; i++ {
+		var data uint8
+		err := binary.Read(r, order, &data)
+		if err != nil {
+			return err
+		}
+
+		// Leading zeros are invalid.
+		if i == 0 && data == 0x80 {
+			return fmt.Errorf("leading zero is invalid")
+		}
+
+		// If any of top 7 bits are set then accum << 7 would overflow.
+		if accum&0xfe000000 != 0 {
+			return fmt.Errorf("top seven bits are set, about to overflow")
+		}
+
+		accum = (accum << 7) | uint32(data)&0x7f
+
+		// Spin until most significant bit of data byte is false.
+		if (data & 0x80) == 0 {
+			*v = accum
+			return nil
+		}
+	}
+	return fmt.Errorf("UIntBase128 sequence exceeds 5 bytes")
+}
+
+// read255UShort reads a 255UInt16 value.
+func read255UShort(r io.Reader, v *uint16) error {
+	const (
+		oneMoreByteCode1 = 255
+		oneMoreByteCode2 = 254
+		wordCode         = 253
+		lowestUCode      = 253
+	)
+	var code uint8
+	err := binary.Read(r, order, &code)
+	if err != nil {
+		return err
+	}
+	switch code {
+	case wordCode:
+		var value uint16
+		err := binary.Read(r, order, &value)
+		if err != nil {
+			return err
+		}
+		*v = value
+		return nil
+	case oneMoreByteCode1:
+		var value uint8
+		err := binary.Read(r, order, &value)
+		if err != nil {
+			return err
+		}
+		*v = uint16(value) + lowestUCode
+		return nil
+	case oneMoreByteCode2:
+		var value uint8
+		err := binary.Read(r, order, &value)
+		if err != nil {
+			return err
+		}
+		*v = uint16(value) + lowestUCode*2
+		return nil
+	default:
+		*v = uint16(code)
+		return nil
+	}
+}
+
+// WOFF2 uses big endian encoding.
+var order binary.ByteOrder = binary.BigEndian
diff --git a/parse_test.go b/parse_test.go
new file mode 100644
index 0000000..87545cf
--- /dev/null
+++ b/parse_test.go
@@ -0,0 +1,109 @@
+package woff2_test
+
+import (
+	"fmt"
+	"log"
+
+	"dmitri.shuralyov.com/font/woff2"
+	"github.com/shurcooL/gofontwoff"
+)
+
+func ExampleParse() {
+	f, err := gofontwoff.Assets.Open("/Go-Regular.woff2")
+	if err != nil {
+		log.Fatalln(err)
+	}
+	defer f.Close()
+
+	font, err := woff2.Parse(f)
+	if err != nil {
+		log.Fatalln(err)
+	}
+	Dump(font)
+
+	// Output:
+	//
+	// Signature:           0x774f4632
+	// Flavor:              0x00010000
+	// Length:              46132
+	// NumTables:           14
+	// Reserved:            0
+	// TotalSfntSize:       140308
+	// TotalCompressedSize: 46040
+	// MajorVersion:        1
+	// MinorVersion:        0
+	// MetaOffset:          0
+	// MetaLength:          0
+	// MetaOrigLength:      0
+	// PrivOffset:          0
+	// PrivLength:          0
+	//
+	// TableDirectory: 14 entries
+	// 	{Flags: 0x06, Tag: <nil>, OrigLength: 96, TransformLength: <nil>}
+	// 	{Flags: 0x00, Tag: <nil>, OrigLength: 1318, TransformLength: <nil>}
+	// 	{Flags: 0x08, Tag: <nil>, OrigLength: 176, TransformLength: <nil>}
+	// 	{Flags: 0x09, Tag: <nil>, OrigLength: 3437, TransformLength: <nil>}
+	// 	{Flags: 0x11, Tag: <nil>, OrigLength: 8, TransformLength: <nil>}
+	// 	{Flags: 0x0a, Tag: <nil>, OrigLength: 118912, TransformLength: 105020}
+	// 	{Flags: 0x0b, Tag: <nil>, OrigLength: 1334, TransformLength: 0}
+	// 	{Flags: 0x01, Tag: <nil>, OrigLength: 54, TransformLength: <nil>}
+	// 	{Flags: 0x02, Tag: <nil>, OrigLength: 36, TransformLength: <nil>}
+	// 	{Flags: 0x03, Tag: <nil>, OrigLength: 2662, TransformLength: <nil>}
+	// 	{Flags: 0x04, Tag: <nil>, OrigLength: 32, TransformLength: <nil>}
+	// 	{Flags: 0x05, Tag: <nil>, OrigLength: 6967, TransformLength: <nil>}
+	// 	{Flags: 0x07, Tag: <nil>, OrigLength: 4838, TransformLength: <nil>}
+	// 	{Flags: 0x0c, Tag: <nil>, OrigLength: 188, TransformLength: <nil>}
+	//
+	// CollectionDirectory: <nil>
+	// CompressedFontData: 124832 bytes (uncompressed size)
+	// ExtendedMetadata: <nil>
+	// PrivateData: []
+}
+
+func Dump(f woff2.File) {
+	dumpHeader(f.Header)
+	fmt.Println()
+	dumpTableDirectory(f.TableDirectory)
+	fmt.Println()
+	fmt.Println("CollectionDirectory:", f.CollectionDirectory)
+	fmt.Println("CompressedFontData:", len(f.FontData), "bytes (uncompressed size)")
+	fmt.Println("ExtendedMetadata:", f.ExtendedMetadata)
+	fmt.Println("PrivateData:", f.PrivateData)
+}
+
+func dumpHeader(hdr woff2.Header) {
+	fmt.Printf("Signature:           %#08x\n", hdr.Signature)
+	fmt.Printf("Flavor:              %#08x\n", hdr.Flavor)
+	fmt.Printf("Length:              %d\n", hdr.Length)
+	fmt.Printf("NumTables:           %d\n", hdr.NumTables)
+	fmt.Printf("Reserved:            %d\n", hdr.Reserved)
+	fmt.Printf("TotalSfntSize:       %d\n", hdr.TotalSfntSize)
+	fmt.Printf("TotalCompressedSize: %d\n", hdr.TotalCompressedSize)
+	fmt.Printf("MajorVersion:        %d\n", hdr.MajorVersion)
+	fmt.Printf("MinorVersion:        %d\n", hdr.MinorVersion)
+	fmt.Printf("MetaOffset:          %d\n", hdr.MetaOffset)
+	fmt.Printf("MetaLength:          %d\n", hdr.MetaLength)
+	fmt.Printf("MetaOrigLength:      %d\n", hdr.MetaOrigLength)
+	fmt.Printf("PrivOffset:          %d\n", hdr.PrivOffset)
+	fmt.Printf("PrivLength:          %d\n", hdr.PrivLength)
+}
+
+func dumpTableDirectory(td woff2.TableDirectory) {
+	fmt.Println("TableDirectory:", len(td), "entries")
+	for _, t := range td {
+		fmt.Printf("\t{")
+		fmt.Printf("Flags: %#02x, ", t.Flags)
+		if t.Tag != nil {
+			fmt.Printf("Tag: %v, ", *t.Tag)
+		} else {
+			fmt.Printf("Tag: <nil>, ")
+		}
+		fmt.Printf("OrigLength: %v, ", t.OrigLength)
+		if t.TransformLength != nil {
+			fmt.Printf("TransformLength: %v", *t.TransformLength)
+		} else {
+			fmt.Printf("TransformLength: <nil>")
+		}
+		fmt.Printf("}\n")
+	}
+}
diff --git a/tags.go b/tags.go
new file mode 100644
index 0000000..3c13a55
--- /dev/null
+++ b/tags.go
@@ -0,0 +1,79 @@
+package woff2
+
+const (
+	// signature is the WOFF 2.0 file identifying signature 'wOF2'.
+	signature = uint32('w'<<24 | 'O'<<16 | 'F'<<8 | '2')
+
+	// ttcfFlavor is the TrueType Collection flavor 'ttcf'.
+	ttcfFlavor = uint32('t'<<24 | 't'<<16 | 'c'<<8 | 'f')
+
+	glyfTable = uint32('g'<<24 | 'l'<<16 | 'y'<<8 | 'f')
+	locaTable = uint32('l'<<24 | 'o'<<16 | 'c'<<8 | 'a')
+)
+
+// knownTableTags is the "Known Table Tags" table.
+var knownTableTags = [...]uint32{
+	0:  uint32('c'<<24 | 'm'<<16 | 'a'<<8 | 'p'),
+	1:  uint32('h'<<24 | 'e'<<16 | 'a'<<8 | 'd'),
+	2:  uint32('h'<<24 | 'h'<<16 | 'e'<<8 | 'a'),
+	3:  uint32('h'<<24 | 'm'<<16 | 't'<<8 | 'x'),
+	4:  uint32('m'<<24 | 'a'<<16 | 'x'<<8 | 'p'),
+	5:  uint32('n'<<24 | 'a'<<16 | 'm'<<8 | 'e'),
+	6:  uint32('O'<<24 | 'S'<<16 | '/'<<8 | '2'),
+	7:  uint32('p'<<24 | 'o'<<16 | 's'<<8 | 't'),
+	8:  uint32('c'<<24 | 'v'<<16 | 't'<<8 | ' '),
+	9:  uint32('f'<<24 | 'p'<<16 | 'g'<<8 | 'm'),
+	10: uint32('g'<<24 | 'l'<<16 | 'y'<<8 | 'f'),
+	11: uint32('l'<<24 | 'o'<<16 | 'c'<<8 | 'a'),
+	12: uint32('p'<<24 | 'r'<<16 | 'e'<<8 | 'p'),
+	13: uint32('C'<<24 | 'F'<<16 | 'F'<<8 | ' '),
+	14: uint32('V'<<24 | 'O'<<16 | 'R'<<8 | 'G'),
+	15: uint32('E'<<24 | 'B'<<16 | 'D'<<8 | 'T'),
+	16: uint32('E'<<24 | 'B'<<16 | 'L'<<8 | 'C'),
+	17: uint32('g'<<24 | 'a'<<16 | 's'<<8 | 'p'),
+	18: uint32('h'<<24 | 'd'<<16 | 'm'<<8 | 'x'),
+	19: uint32('k'<<24 | 'e'<<16 | 'r'<<8 | 'n'),
+	20: uint32('L'<<24 | 'T'<<16 | 'S'<<8 | 'H'),
+	21: uint32('P'<<24 | 'C'<<16 | 'L'<<8 | 'T'),
+	22: uint32('V'<<24 | 'D'<<16 | 'M'<<8 | 'X'),
+	23: uint32('v'<<24 | 'h'<<16 | 'e'<<8 | 'a'),
+	24: uint32('v'<<24 | 'm'<<16 | 't'<<8 | 'x'),
+	25: uint32('B'<<24 | 'A'<<16 | 'S'<<8 | 'E'),
+	26: uint32('G'<<24 | 'D'<<16 | 'E'<<8 | 'F'),
+	27: uint32('G'<<24 | 'P'<<16 | 'O'<<8 | 'S'),
+	28: uint32('G'<<24 | 'S'<<16 | 'U'<<8 | 'B'),
+	29: uint32('E'<<24 | 'B'<<16 | 'S'<<8 | 'C'),
+	30: uint32('J'<<24 | 'S'<<16 | 'T'<<8 | 'F'),
+	31: uint32('M'<<24 | 'A'<<16 | 'T'<<8 | 'H'),
+	32: uint32('C'<<24 | 'B'<<16 | 'D'<<8 | 'T'),
+	33: uint32('C'<<24 | 'B'<<16 | 'L'<<8 | 'C'),
+	34: uint32('C'<<24 | 'O'<<16 | 'L'<<8 | 'R'),
+	35: uint32('C'<<24 | 'P'<<16 | 'A'<<8 | 'L'),
+	36: uint32('S'<<24 | 'V'<<16 | 'G'<<8 | ' '),
+	37: uint32('s'<<24 | 'b'<<16 | 'i'<<8 | 'x'),
+	38: uint32('a'<<24 | 'c'<<16 | 'n'<<8 | 't'),
+	39: uint32('a'<<24 | 'v'<<16 | 'a'<<8 | 'r'),
+	40: uint32('b'<<24 | 'd'<<16 | 'a'<<8 | 't'),
+	41: uint32('b'<<24 | 'l'<<16 | 'o'<<8 | 'c'),
+	42: uint32('b'<<24 | 's'<<16 | 'l'<<8 | 'n'),
+	43: uint32('c'<<24 | 'v'<<16 | 'a'<<8 | 'r'),
+	44: uint32('f'<<24 | 'd'<<16 | 's'<<8 | 'c'),
+	45: uint32('f'<<24 | 'e'<<16 | 'a'<<8 | 't'),
+	46: uint32('f'<<24 | 'm'<<16 | 't'<<8 | 'x'),
+	47: uint32('f'<<24 | 'v'<<16 | 'a'<<8 | 'r'),
+	48: uint32('g'<<24 | 'v'<<16 | 'a'<<8 | 'r'),
+	49: uint32('h'<<24 | 's'<<16 | 't'<<8 | 'y'),
+	50: uint32('j'<<24 | 'u'<<16 | 's'<<8 | 't'),
+	51: uint32('l'<<24 | 'c'<<16 | 'a'<<8 | 'r'),
+	52: uint32('m'<<24 | 'o'<<16 | 'r'<<8 | 't'),
+	53: uint32('m'<<24 | 'o'<<16 | 'r'<<8 | 'x'),
+	54: uint32('o'<<24 | 'p'<<16 | 'b'<<8 | 'd'),
+	55: uint32('p'<<24 | 'r'<<16 | 'o'<<8 | 'p'),
+	56: uint32('t'<<24 | 'r'<<16 | 'a'<<8 | 'k'),
+	57: uint32('Z'<<24 | 'a'<<16 | 'p'<<8 | 'f'),
+	58: uint32('S'<<24 | 'i'<<16 | 'l'<<8 | 'f'),
+	59: uint32('G'<<24 | 'l'<<16 | 'a'<<8 | 't'),
+	60: uint32('G'<<24 | 'l'<<16 | 'o'<<8 | 'c'),
+	61: uint32('F'<<24 | 'e'<<16 | 'a'<<8 | 't'),
+	62: uint32('S'<<24 | 'i'<<16 | 'l'<<8 | 'l'),
+}
diff --git a/tags_test.go b/tags_test.go
new file mode 100644
index 0000000..3980f39
--- /dev/null
+++ b/tags_test.go
@@ -0,0 +1,10 @@
+package woff2
+
+import "testing"
+
+func TestKnownTableTagsLength(t *testing.T) {
+	const want = 63
+	if got := len(knownTableTags); got != want {
+		t.Errorf("got len(knownTableTags): %v, want: %v", got, want)
+	}
+}
`

const diffCommit1 = `diff --git a/doc.go b/doc.go
index fd35888..a751214 100644
--- a/doc.go
+++ b/doc.go
@@ -2,5 +2,3 @@
 //
 // The WOFF2 font packaging format is specified at https://www.w3.org/TR/WOFF2/.
 package woff2
-
-// TODO: Implement.
diff --git a/parse.go b/parse.go
new file mode 100644
index 0000000..498a4a8
--- /dev/null
+++ b/parse.go
@@ -0,0 +1,438 @@
+package woff2
+
+import (
+	"bytes"
+	"encoding/binary"
+	"fmt"
+	"io"
+
+	"github.com/dsnet/compress/brotli"
+)
+
+// File represents a parsed WOFF2 file.
+type File struct {
+	Header         Header
+	TableDirectory TableDirectory
+	// CollectionDirectory is present only if the font is a collection,
+	// as reported by Header.IsCollection.
+	CollectionDirectory *CollectionDirectory
+
+	// FontData is the concatenation of data for each table in the font.
+	// During storage, it's compressed using Brotli.
+	FontData []byte
+
+	ExtendedMetadata *ExtendedMetadata
+
+	// PrivateData is an optional block of private data for the font designer,
+	// foundry, or vendor to use.
+	PrivateData []byte
+}
+
+// Parse parses the WOFF2 data from r.
+func Parse(r io.Reader) (File, error) {
+	hdr, err := parseHeader(r)
+	if err != nil {
+		return File{}, err
+	}
+	td, err := parseTableDirectory(r, hdr)
+	if err != nil {
+		return File{}, err
+	}
+	cd, err := parseCollectionDirectory(r, hdr)
+	if err != nil {
+		return File{}, err
+	}
+	fd, err := parseCompressedFontData(r, hdr, td)
+	if err != nil {
+		return File{}, err
+	}
+	em, err := parseExtendedMetadata(r, hdr)
+	if err != nil {
+		return File{}, err
+	}
+	pd, err := parsePrivateData(r, hdr)
+	if err != nil {
+		return File{}, err
+	}
+
+	n, err := io.Copy(discardZeroes{}, r)
+	if err != nil {
+		return File{}, fmt.Errorf("Parse: %v", err)
+	}
+	if n > 3 {
+		return File{}, fmt.Errorf("Parse: %d bytes left remaining, want no more than 3", n)
+	}
+
+	return File{
+		Header:              hdr,
+		TableDirectory:      td,
+		CollectionDirectory: cd,
+		FontData:            fd,
+		ExtendedMetadata:    em,
+		PrivateData:         pd,
+	}, nil
+}
+
+// discardZeroes is an io.Writer that returns an error if any non-zero bytes are written to it.
+type discardZeroes struct{}
+
+func (discardZeroes) Write(p []byte) (int, error) {
+	for _, b := range p {
+		if b != 0 {
+			return 0, fmt.Errorf("encountered non-zero byte %d", b)
+		}
+	}
+	return len(p), nil
+}
+
+// Header is the file header with basic font type and version,
+// along with offsets to metadata and private data blocks.
+type Header struct {
+	Signature           uint32 // The identifying signature; must be 0x774F4632 ('wOF2').
+	Flavor              uint32 // The "sfnt version" of the input font.
+	Length              uint32 // Total size of the WOFF file.
+	NumTables           uint16 // Number of entries in directory of font tables.
+	Reserved            uint16 // Reserved; set to 0.
+	TotalSfntSize       uint32 // Total size needed for the uncompressed font data, including the sfnt header, directory, and font tables (including padding).
+	TotalCompressedSize uint32 // Total length of the compressed data block.
+	MajorVersion        uint16 // Major version of the WOFF file.
+	MinorVersion        uint16 // Minor version of the WOFF file.
+	MetaOffset          uint32 // Offset to metadata block, from beginning of WOFF file.
+	MetaLength          uint32 // Length of compressed metadata block.
+	MetaOrigLength      uint32 // Uncompressed size of metadata block.
+	PrivOffset          uint32 // Offset to private data block, from beginning of WOFF file.
+	PrivLength          uint32 // Length of private data block.
+}
+
+func parseHeader(r io.Reader) (Header, error) {
+	var hdr Header
+	err := binary.Read(r, order, &hdr)
+	if err != nil {
+		return Header{}, err
+	}
+	if hdr.Signature != signature {
+		return Header{}, fmt.Errorf("parseHeader: invalid signature: got %#08x, want %#08x", hdr.Signature, signature)
+	}
+	return hdr, nil
+}
+
+// IsCollection reports whether this is a font collection, i.e.,
+// if the value of Flavor field is set to the TrueType Collection flavor 'ttcf'.
+func (hdr Header) IsCollection() bool {
+	return hdr.Flavor == ttcfFlavor
+}
+
+// TableDirectory is the directory of font tables, containing size and other info.
+type TableDirectory []TableDirectoryEntry
+
+func parseTableDirectory(r io.Reader, hdr Header) (TableDirectory, error) {
+	var td TableDirectory
+	for i := 0; i < int(hdr.NumTables); i++ {
+		var e TableDirectoryEntry
+
+		err := readU8(r, &e.Flags)
+		if err != nil {
+			return nil, err
+		}
+		if e.Flags&0x3f == 0x3f {
+			e.Tag = new(uint32)
+			err := readU32(r, e.Tag)
+			if err != nil {
+				return nil, err
+			}
+		}
+		err = readBase128(r, &e.OrigLength)
+		if err != nil {
+			return nil, err
+		}
+
+		switch tag, transformVersion := e.tag(), e.transformVersion(); tag {
+		case glyfTable, locaTable:
+			// 0 means transform for glyf/loca tables.
+			if transformVersion == 0 {
+				e.TransformLength = new(uint32)
+				err := readBase128(r, e.TransformLength)
+				if err != nil {
+					return nil, err
+				}
+
+				// The transform length of the transformed loca table MUST always be zero.
+				if tag == locaTable && *e.TransformLength != 0 {
+					return nil, fmt.Errorf("parseTableDirectory: 'loca' table has non-zero transform length %d", *e.TransformLength)
+				}
+			}
+		default:
+			// Non-0 means transform for other tables.
+			if transformVersion != 0 {
+				e.TransformLength = new(uint32)
+				err := readBase128(r, e.TransformLength)
+				if err != nil {
+					return nil, err
+				}
+			}
+		}
+
+		td = append(td, e)
+	}
+	return td, nil
+}
+
+// Table is a high-level representation of a table.
+type Table struct {
+	Tag    uint32
+	Offset int
+	Length int
+}
+
+// Tables returns the derived high-level information
+// about the tables in the table directory.
+func (td TableDirectory) Tables() []Table {
+	var ts []Table
+	var offset int
+	for _, t := range td {
+		length := int(t.length())
+		ts = append(ts, Table{
+			Tag:    t.tag(),
+			Offset: offset,
+			Length: length,
+		})
+		offset += length
+	}
+	return ts
+}
+
+// uncompressedSize computes the total uncompressed size
+// of the tables in the table directory.
+func (td TableDirectory) uncompressedSize() int64 {
+	var n int64
+	for _, t := range td {
+		n += int64(t.length())
+	}
+	return n
+}
+
+// TableDirectoryEntry is a table directory entry.
+type TableDirectoryEntry struct {
+	Flags           uint8   // Table type and flags.
+	Tag             *uint32 // 4-byte tag (optional).
+	OrigLength      uint32  // Length of original table.
+	TransformLength *uint32 // Transformed length (optional).
+}
+
+func (e TableDirectoryEntry) tag() uint32 {
+	switch e.Tag {
+	case nil:
+		return knownTableTags[e.Flags&0x3f] // Bits [0..5].
+	default:
+		return *e.Tag
+	}
+}
+
+func (e TableDirectoryEntry) transformVersion() uint8 {
+	return e.Flags >> 6 // Bits [6..7].
+}
+
+func (e TableDirectoryEntry) length() uint32 {
+	switch e.TransformLength {
+	case nil:
+		return e.OrigLength
+	default:
+		return *e.TransformLength
+	}
+}
+
+// CollectionDirectory is an optional table containing the font fragment descriptions
+// of font collection entries.
+type CollectionDirectory struct {
+	Header  CollectionHeader
+	Entries []CollectionFontEntry
+}
+
+// CollectionHeader is a part of CollectionDirectory.
+type CollectionHeader struct {
+	Version  uint32
+	NumFonts uint16
+}
+
+// CollectionFontEntry represents a CollectionFontEntry record.
+type CollectionFontEntry struct {
+	NumTables    uint16   // The number of tables in this font.
+	Flavor       uint32   // The "sfnt version" of the font.
+	TableIndices []uint16 // The indicies identifying an entry in the Table Directory for each table in this font.
+}
+
+func parseCollectionDirectory(r io.Reader, hdr Header) (*CollectionDirectory, error) {
+	// CollectionDirectory is present only if the input font is a collection.
+	if !hdr.IsCollection() {
+		return nil, nil
+	}
+
+	var cd CollectionDirectory
+	err := readU32(r, &cd.Header.Version)
+	if err != nil {
+		return nil, err
+	}
+	err = read255UShort(r, &cd.Header.NumFonts)
+	if err != nil {
+		return nil, err
+	}
+	for i := 0; i < int(cd.Header.NumFonts); i++ {
+		var e CollectionFontEntry
+
+		err := read255UShort(r, &e.NumTables)
+		if err != nil {
+			return nil, err
+		}
+		err = readU32(r, &e.Flavor)
+		if err != nil {
+			return nil, err
+		}
+		for j := 0; j < int(e.NumTables); j++ {
+			var tableIndex uint16
+			err := read255UShort(r, &tableIndex)
+			if err != nil {
+				return nil, err
+			}
+			if tableIndex >= hdr.NumTables {
+				return nil, fmt.Errorf("parseCollectionDirectory: tableIndex >= hdr.NumTables")
+			}
+			e.TableIndices = append(e.TableIndices, tableIndex)
+		}
+
+		cd.Entries = append(cd.Entries, e)
+	}
+	return &cd, nil
+}
+
+func parseCompressedFontData(r io.Reader, hdr Header, td TableDirectory) ([]byte, error) {
+	// Compressed font data.
+	br, err := brotli.NewReader(io.LimitReader(r, int64(hdr.TotalCompressedSize)), nil)
+	//br, err := brotli.NewReader(&exactReader{R: r, N: int64(hdr.TotalCompressedSize)}, nil)
+	if err != nil {
+		return nil, err
+	}
+	var buf bytes.Buffer
+	n, err := io.Copy(&buf, br)
+	if err != nil {
+		return nil, fmt.Errorf("parseCompressedFontData: io.Copy: %v", err)
+	}
+	err = br.Close()
+	if err != nil {
+		return nil, fmt.Errorf("parseCompressedFontData: br.Close: %v", err)
+	}
+	if uncompressedSize := td.uncompressedSize(); n != uncompressedSize {
+		return nil, fmt.Errorf("parseCompressedFontData: unexpected size of uncompressed data: got %d, want %d", n, uncompressedSize)
+	}
+	return buf.Bytes(), nil
+}
+
+// ExtendedMetadata is an optional block of extended metadata,
+// represented in XML format and compressed for storage in the WOFF2 file.
+type ExtendedMetadata struct{}
+
+func parseExtendedMetadata(r io.Reader, hdr Header) (*ExtendedMetadata, error) {
+	if hdr.MetaLength == 0 {
+		return nil, nil
+	}
+	return nil, fmt.Errorf("parseExtendedMetadata: not implemented")
+}
+
+func parsePrivateData(r io.Reader, hdr Header) ([]byte, error) {
+	if hdr.PrivLength == 0 {
+		return nil, nil
+	}
+	return nil, fmt.Errorf("parsePrivateData: not implemented")
+}
+
+// readU8 reads a UInt8 value.
+func readU8(r io.Reader, v *uint8) error {
+	return binary.Read(r, order, v)
+}
+
+// readU16 reads a UInt16 value.
+func readU16(r io.Reader, v *uint16) error {
+	return binary.Read(r, order, v)
+}
+
+// readU32 reads a UInt32 value.
+func readU32(r io.Reader, v *uint32) error {
+	return binary.Read(r, order, v)
+}
+
+// readBase128 reads a UIntBase128 value.
+func readBase128(r io.Reader, v *uint32) error {
+	var accum uint32
+	for i := 0; i < 5; i++ {
+		var data uint8
+		err := binary.Read(r, order, &data)
+		if err != nil {
+			return err
+		}
+
+		// Leading zeros are invalid.
+		if i == 0 && data == 0x80 {
+			return fmt.Errorf("leading zero is invalid")
+		}
+
+		// If any of top 7 bits are set then accum << 7 would overflow.
+		if accum&0xfe000000 != 0 {
+			return fmt.Errorf("top seven bits are set, about to overflow")
+		}
+
+		accum = (accum << 7) | uint32(data)&0x7f
+
+		// Spin until most significant bit of data byte is false.
+		if (data & 0x80) == 0 {
+			*v = accum
+			return nil
+		}
+	}
+	return fmt.Errorf("UIntBase128 sequence exceeds 5 bytes")
+}
+
+// read255UShort reads a 255UInt16 value.
+func read255UShort(r io.Reader, v *uint16) error {
+	const (
+		oneMoreByteCode1 = 255
+		oneMoreByteCode2 = 254
+		wordCode         = 253
+		lowestUCode      = 253
+	)
+	var code uint8
+	err := binary.Read(r, order, &code)
+	if err != nil {
+		return err
+	}
+	switch code {
+	case wordCode:
+		var value uint16
+		err := binary.Read(r, order, &value)
+		if err != nil {
+			return err
+		}
+		*v = value
+		return nil
+	case oneMoreByteCode1:
+		var value uint8
+		err := binary.Read(r, order, &value)
+		if err != nil {
+			return err
+		}
+		*v = uint16(value) + lowestUCode
+		return nil
+	case oneMoreByteCode2:
+		var value uint8
+		err := binary.Read(r, order, &value)
+		if err != nil {
+			return err
+		}
+		*v = uint16(value) + lowestUCode*2
+		return nil
+	default:
+		*v = uint16(code)
+		return nil
+	}
+}
+
+// WOFF2 uses big endian encoding.
+var order binary.ByteOrder = binary.BigEndian
diff --git a/parse_test.go b/parse_test.go
new file mode 100644
index 0000000..87545cf
--- /dev/null
+++ b/parse_test.go
@@ -0,0 +1,109 @@
+package woff2_test
+
+import (
+	"fmt"
+	"log"
+
+	"dmitri.shuralyov.com/font/woff2"
+	"github.com/shurcooL/gofontwoff"
+)
+
+func ExampleParse() {
+	f, err := gofontwoff.Assets.Open("/Go-Regular.woff2")
+	if err != nil {
+		log.Fatalln(err)
+	}
+	defer f.Close()
+
+	font, err := woff2.Parse(f)
+	if err != nil {
+		log.Fatalln(err)
+	}
+	Dump(font)
+
+	// Output:
+	//
+	// Signature:           0x774f4632
+	// Flavor:              0x00010000
+	// Length:              46132
+	// NumTables:           14
+	// Reserved:            0
+	// TotalSfntSize:       140308
+	// TotalCompressedSize: 46040
+	// MajorVersion:        1
+	// MinorVersion:        0
+	// MetaOffset:          0
+	// MetaLength:          0
+	// MetaOrigLength:      0
+	// PrivOffset:          0
+	// PrivLength:          0
+	//
+	// TableDirectory: 14 entries
+	// 	{Flags: 0x06, Tag: <nil>, OrigLength: 96, TransformLength: <nil>}
+	// 	{Flags: 0x00, Tag: <nil>, OrigLength: 1318, TransformLength: <nil>}
+	// 	{Flags: 0x08, Tag: <nil>, OrigLength: 176, TransformLength: <nil>}
+	// 	{Flags: 0x09, Tag: <nil>, OrigLength: 3437, TransformLength: <nil>}
+	// 	{Flags: 0x11, Tag: <nil>, OrigLength: 8, TransformLength: <nil>}
+	// 	{Flags: 0x0a, Tag: <nil>, OrigLength: 118912, TransformLength: 105020}
+	// 	{Flags: 0x0b, Tag: <nil>, OrigLength: 1334, TransformLength: 0}
+	// 	{Flags: 0x01, Tag: <nil>, OrigLength: 54, TransformLength: <nil>}
+	// 	{Flags: 0x02, Tag: <nil>, OrigLength: 36, TransformLength: <nil>}
+	// 	{Flags: 0x03, Tag: <nil>, OrigLength: 2662, TransformLength: <nil>}
+	// 	{Flags: 0x04, Tag: <nil>, OrigLength: 32, TransformLength: <nil>}
+	// 	{Flags: 0x05, Tag: <nil>, OrigLength: 6967, TransformLength: <nil>}
+	// 	{Flags: 0x07, Tag: <nil>, OrigLength: 4838, TransformLength: <nil>}
+	// 	{Flags: 0x0c, Tag: <nil>, OrigLength: 188, TransformLength: <nil>}
+	//
+	// CollectionDirectory: <nil>
+	// CompressedFontData: 124832 bytes (uncompressed size)
+	// ExtendedMetadata: <nil>
+	// PrivateData: []
+}
+
+func Dump(f woff2.File) {
+	dumpHeader(f.Header)
+	fmt.Println()
+	dumpTableDirectory(f.TableDirectory)
+	fmt.Println()
+	fmt.Println("CollectionDirectory:", f.CollectionDirectory)
+	fmt.Println("CompressedFontData:", len(f.CompressedFontData.Data), "bytes (uncompressed size)")
+	fmt.Println("ExtendedMetadata:", f.ExtendedMetadata)
+	fmt.Println("PrivateData:", f.PrivateData)
+}
+
+func dumpHeader(hdr woff2.Header) {
+	fmt.Printf("Signature:           %#08x\n", hdr.Signature)
+	fmt.Printf("Flavor:              %#08x\n", hdr.Flavor)
+	fmt.Printf("Length:              %d\n", hdr.Length)
+	fmt.Printf("NumTables:           %d\n", hdr.NumTables)
+	fmt.Printf("Reserved:            %d\n", hdr.Reserved)
+	fmt.Printf("TotalSfntSize:       %d\n", hdr.TotalSfntSize)
+	fmt.Printf("TotalCompressedSize: %d\n", hdr.TotalCompressedSize)
+	fmt.Printf("MajorVersion:        %d\n", hdr.MajorVersion)
+	fmt.Printf("MinorVersion:        %d\n", hdr.MinorVersion)
+	fmt.Printf("MetaOffset:          %d\n", hdr.MetaOffset)
+	fmt.Printf("MetaLength:          %d\n", hdr.MetaLength)
+	fmt.Printf("MetaOrigLength:      %d\n", hdr.MetaOrigLength)
+	fmt.Printf("PrivOffset:          %d\n", hdr.PrivOffset)
+	fmt.Printf("PrivLength:          %d\n", hdr.PrivLength)
+}
+
+func dumpTableDirectory(td woff2.TableDirectory) {
+	fmt.Println("TableDirectory:", len(td), "entries")
+	for _, t := range td {
+		fmt.Printf("\t{")
+		fmt.Printf("Flags: %#02x, ", t.Flags)
+		if t.Tag != nil {
+			fmt.Printf("Tag: %v, ", *t.Tag)
+		} else {
+			fmt.Printf("Tag: <nil>, ")
+		}
+		fmt.Printf("OrigLength: %v, ", t.OrigLength)
+		if t.TransformLength != nil {
+			fmt.Printf("TransformLength: %v", *t.TransformLength)
+		} else {
+			fmt.Printf("TransformLength: <nil>")
+		}
+		fmt.Printf("}\n")
+	}
+}
diff --git a/tags.go b/tags.go
new file mode 100644
index 0000000..3c13a55
--- /dev/null
+++ b/tags.go
@@ -0,0 +1,79 @@
+package woff2
+
+const (
+	// signature is the WOFF 2.0 file identifying signature 'wOF2'.
+	signature = uint32('w'<<24 | 'O'<<16 | 'F'<<8 | '2')
+
+	// ttcfFlavor is the TrueType Collection flavor 'ttcf'.
+	ttcfFlavor = uint32('t'<<24 | 't'<<16 | 'c'<<8 | 'f')
+
+	glyfTable = uint32('g'<<24 | 'l'<<16 | 'y'<<8 | 'f')
+	locaTable = uint32('l'<<24 | 'o'<<16 | 'c'<<8 | 'a')
+)
+
+// knownTableTags is the "Known Table Tags" table.
+var knownTableTags = [...]uint32{
+	0:  uint32('c'<<24 | 'm'<<16 | 'a'<<8 | 'p'),
+	1:  uint32('h'<<24 | 'e'<<16 | 'a'<<8 | 'd'),
+	2:  uint32('h'<<24 | 'h'<<16 | 'e'<<8 | 'a'),
+	3:  uint32('h'<<24 | 'm'<<16 | 't'<<8 | 'x'),
+	4:  uint32('m'<<24 | 'a'<<16 | 'x'<<8 | 'p'),
+	5:  uint32('n'<<24 | 'a'<<16 | 'm'<<8 | 'e'),
+	6:  uint32('O'<<24 | 'S'<<16 | '/'<<8 | '2'),
+	7:  uint32('p'<<24 | 'o'<<16 | 's'<<8 | 't'),
+	8:  uint32('c'<<24 | 'v'<<16 | 't'<<8 | ' '),
+	9:  uint32('f'<<24 | 'p'<<16 | 'g'<<8 | 'm'),
+	10: uint32('g'<<24 | 'l'<<16 | 'y'<<8 | 'f'),
+	11: uint32('l'<<24 | 'o'<<16 | 'c'<<8 | 'a'),
+	12: uint32('p'<<24 | 'r'<<16 | 'e'<<8 | 'p'),
+	13: uint32('C'<<24 | 'F'<<16 | 'F'<<8 | ' '),
+	14: uint32('V'<<24 | 'O'<<16 | 'R'<<8 | 'G'),
+	15: uint32('E'<<24 | 'B'<<16 | 'D'<<8 | 'T'),
+	16: uint32('E'<<24 | 'B'<<16 | 'L'<<8 | 'C'),
+	17: uint32('g'<<24 | 'a'<<16 | 's'<<8 | 'p'),
+	18: uint32('h'<<24 | 'd'<<16 | 'm'<<8 | 'x'),
+	19: uint32('k'<<24 | 'e'<<16 | 'r'<<8 | 'n'),
+	20: uint32('L'<<24 | 'T'<<16 | 'S'<<8 | 'H'),
+	21: uint32('P'<<24 | 'C'<<16 | 'L'<<8 | 'T'),
+	22: uint32('V'<<24 | 'D'<<16 | 'M'<<8 | 'X'),
+	23: uint32('v'<<24 | 'h'<<16 | 'e'<<8 | 'a'),
+	24: uint32('v'<<24 | 'm'<<16 | 't'<<8 | 'x'),
+	25: uint32('B'<<24 | 'A'<<16 | 'S'<<8 | 'E'),
+	26: uint32('G'<<24 | 'D'<<16 | 'E'<<8 | 'F'),
+	27: uint32('G'<<24 | 'P'<<16 | 'O'<<8 | 'S'),
+	28: uint32('G'<<24 | 'S'<<16 | 'U'<<8 | 'B'),
+	29: uint32('E'<<24 | 'B'<<16 | 'S'<<8 | 'C'),
+	30: uint32('J'<<24 | 'S'<<16 | 'T'<<8 | 'F'),
+	31: uint32('M'<<24 | 'A'<<16 | 'T'<<8 | 'H'),
+	32: uint32('C'<<24 | 'B'<<16 | 'D'<<8 | 'T'),
+	33: uint32('C'<<24 | 'B'<<16 | 'L'<<8 | 'C'),
+	34: uint32('C'<<24 | 'O'<<16 | 'L'<<8 | 'R'),
+	35: uint32('C'<<24 | 'P'<<16 | 'A'<<8 | 'L'),
+	36: uint32('S'<<24 | 'V'<<16 | 'G'<<8 | ' '),
+	37: uint32('s'<<24 | 'b'<<16 | 'i'<<8 | 'x'),
+	38: uint32('a'<<24 | 'c'<<16 | 'n'<<8 | 't'),
+	39: uint32('a'<<24 | 'v'<<16 | 'a'<<8 | 'r'),
+	40: uint32('b'<<24 | 'd'<<16 | 'a'<<8 | 't'),
+	41: uint32('b'<<24 | 'l'<<16 | 'o'<<8 | 'c'),
+	42: uint32('b'<<24 | 's'<<16 | 'l'<<8 | 'n'),
+	43: uint32('c'<<24 | 'v'<<16 | 'a'<<8 | 'r'),
+	44: uint32('f'<<24 | 'd'<<16 | 's'<<8 | 'c'),
+	45: uint32('f'<<24 | 'e'<<16 | 'a'<<8 | 't'),
+	46: uint32('f'<<24 | 'm'<<16 | 't'<<8 | 'x'),
+	47: uint32('f'<<24 | 'v'<<16 | 'a'<<8 | 'r'),
+	48: uint32('g'<<24 | 'v'<<16 | 'a'<<8 | 'r'),
+	49: uint32('h'<<24 | 's'<<16 | 't'<<8 | 'y'),
+	50: uint32('j'<<24 | 'u'<<16 | 's'<<8 | 't'),
+	51: uint32('l'<<24 | 'c'<<16 | 'a'<<8 | 'r'),
+	52: uint32('m'<<24 | 'o'<<16 | 'r'<<8 | 't'),
+	53: uint32('m'<<24 | 'o'<<16 | 'r'<<8 | 'x'),
+	54: uint32('o'<<24 | 'p'<<16 | 'b'<<8 | 'd'),
+	55: uint32('p'<<24 | 'r'<<16 | 'o'<<8 | 'p'),
+	56: uint32('t'<<24 | 'r'<<16 | 'a'<<8 | 'k'),
+	57: uint32('Z'<<24 | 'a'<<16 | 'p'<<8 | 'f'),
+	58: uint32('S'<<24 | 'i'<<16 | 'l'<<8 | 'f'),
+	59: uint32('G'<<24 | 'l'<<16 | 'a'<<8 | 't'),
+	60: uint32('G'<<24 | 'l'<<16 | 'o'<<8 | 'c'),
+	61: uint32('F'<<24 | 'e'<<16 | 'a'<<8 | 't'),
+	62: uint32('S'<<24 | 'i'<<16 | 'l'<<8 | 'l'),
+}
diff --git a/tags_test.go b/tags_test.go
new file mode 100644
index 0000000..3980f39
--- /dev/null
+++ b/tags_test.go
@@ -0,0 +1,10 @@
+package woff2
+
+import "testing"
+
+func TestKnownTableTagsLength(t *testing.T) {
+	const want = 63
+	if got := len(knownTableTags); got != want {
+		t.Errorf("got len(knownTableTags): %v, want: %v", got, want)
+	}
+}
`

const diffCommit2 = `diff --git a/parse_test.go b/parse_test.go
index 87545cf..14e2830 100644
--- a/parse_test.go
+++ b/parse_test.go
@@ -66,7 +66,7 @@ func Dump(f woff2.File) {
 	dumpTableDirectory(f.TableDirectory)
 	fmt.Println()
 	fmt.Println("CollectionDirectory:", f.CollectionDirectory)
-	fmt.Println("CompressedFontData:", len(f.CompressedFontData.Data), "bytes (uncompressed size)")
+	fmt.Println("CompressedFontData:", len(f.FontData), "bytes (uncompressed size)")
 	fmt.Println("ExtendedMetadata:", f.ExtendedMetadata)
 	fmt.Println("PrivateData:", f.PrivateData)
 }
`

const diffCommit3 = `diff --git a/parse.go b/parse.go
index 498a4a8..0004d27 100644
--- a/parse.go
+++ b/parse.go
@@ -55,6 +55,9 @@ func Parse(r io.Reader) (File, error) {
 		return File{}, err
 	}

+	// Check for padding with a maximum of three null bytes.
+	// TODO: This check needs to be moved to Extended Metadata and Private Data blocks,
+	//       and made more precise (i.e., the beginning of those blocks must be 4-byte aligned, etc.).
 	n, err := io.Copy(discardZeroes{}, r)
 	if err != nil {
 		return File{}, fmt.Errorf("Parse: %v", err)
`

const diffMtlAll = `diff --git a/Commit Message b/Commit Message
new file mode 100644
index 0000000..dfb31fe
--- /dev/null
+++ b/Commit Message
@@ -0,0 +1,27 @@
+Parent:     0cf138a8 (cmd/mtlinfo: Add a tool to list all Metal devices, supported feature sets.)
+Author:     Dmitri Shuralyov <dmitri@shuralyov.com>
+AuthorDate: Sat Jun 23 01:07:53 2018 -0400
+Commit:     Dmitri Shuralyov <dmitri@shuralyov.com>
+CommitDate: Sat Oct 20 23:15:25 2018 -0400
+
+Add minimal API to support interactive rendering in a window.
+
+The goal of this change is to make it possible to use package mtl
+to render to a window at interactive framerates (e.g., at 60 FPS,
+assuming a 60 Hz display with vsync on). It adds the minimal API
+that is needed.
+
+A new movingtriangle example is added as a demonstration of this
+functionality. It opens a window and renders a triangle that follows
+the mouse cursor.
+
+Much of the needed API comes from Core Animation, AppKit frameworks,
+rather than Metal. Avoid adding that to mtl package; instead create
+separate packages. For now, they are hidden in internal to avoid
+committing to a public API and import path. After gaining more
+confidence in the approach, they can be factored out and made public.
diff --git a/example/movingtriangle/internal/ca/ca.go b/example/movingtriangle/internal/ca/ca.go
new file mode 100644
index 0000000..d2ff39d
--- /dev/null
+++ b/example/movingtriangle/internal/ca/ca.go
@@ -0,0 +1,137 @@
+// +build darwin
+
+// Package ca provides access to Apple's Core Animation API (https://developer.apple.com/documentation/quartzcore).
+//
+// This package is in very early stages of development.
+// It's a minimal implementation with scope limited to
+// supporting the movingtriangle example.
+package ca
+
+import (
+	"errors"
+	"unsafe"
+
+	"dmitri.shuralyov.com/gpu/mtl"
+)
+
+/*
+#cgo LDFLAGS: -framework QuartzCore -framework Foundation
+#include "ca.h"
+*/
+import "C"
+
+// Layer is an object that manages image-based content and
+// allows you to perform animations on that content.
+//
+// Reference: https://developer.apple.com/documentation/quartzcore/calayer.
+type Layer interface {
+	// Layer returns the underlying CALayer * pointer.
+	Layer() unsafe.Pointer
+}
+
+// MetalLayer is a Core Animation Metal layer, a layer that manages a pool of Metal drawables.
+//
+// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer.
+type MetalLayer struct {
+	metalLayer unsafe.Pointer
+}
+
+// MakeMetalLayer creates a new Core Animation Metal layer.
+//
+// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer.
+func MakeMetalLayer() MetalLayer {
+	return MetalLayer{C.MakeMetalLayer()}
+}
+
+// Layer implements the Layer interface.
+func (ml MetalLayer) Layer() unsafe.Pointer { return ml.metalLayer }
+
+// PixelFormat returns the pixel format of textures for rendering layer content.
+//
+// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478155-pixelformat.
+func (ml MetalLayer) PixelFormat() mtl.PixelFormat {
+	return mtl.PixelFormat(C.MetalLayer_PixelFormat(ml.metalLayer))
+}
+
+// SetDevice sets the Metal device responsible for the layer's drawable resources.
+//
+// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478163-device.
+func (ml MetalLayer) SetDevice(device mtl.Device) {
+	C.MetalLayer_SetDevice(ml.metalLayer, device.Device())
+}
+
+// SetPixelFormat controls the pixel format of textures for rendering layer content.
+//
+// The pixel format for a Metal layer must be PixelFormatBGRA8UNorm, PixelFormatBGRA8UNormSRGB,
+// PixelFormatRGBA16Float, PixelFormatBGRA10XR, or PixelFormatBGRA10XRSRGB.
+// SetPixelFormat panics for other values.
+//
+// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478155-pixelformat.
+func (ml MetalLayer) SetPixelFormat(pf mtl.PixelFormat) {
+	e := C.MetalLayer_SetPixelFormat(ml.metalLayer, C.uint16_t(pf))
+	if e != nil {
+		panic(errors.New(C.GoString(e)))
+	}
+}
+
+// SetMaximumDrawableCount controls the number of Metal drawables in the resource pool
+// managed by Core Animation.
+//
+// It can set to 2 or 3 only. SetMaximumDrawableCount panics for other values.
+//
+// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/2938720-maximumdrawablecount.
+func (ml MetalLayer) SetMaximumDrawableCount(count int) {
+	e := C.MetalLayer_SetMaximumDrawableCount(ml.metalLayer, C.uint_t(count))
+	if e != nil {
+		panic(errors.New(C.GoString(e)))
+	}
+}
+
+// SetDisplaySyncEnabled controls whether the Metal layer and its drawables
+// are synchronized with the display's refresh rate.
+//
+// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/2887087-displaysyncenabled.
+func (ml MetalLayer) SetDisplaySyncEnabled(enabled bool) {
+	switch enabled {
+	case true:
+		C.MetalLayer_SetDisplaySyncEnabled(ml.metalLayer, 1)
+	case false:
+		C.MetalLayer_SetDisplaySyncEnabled(ml.metalLayer, 0)
+	}
+}
+
+// SetDrawableSize sets the size, in pixels, of textures for rendering layer content.
+//
+// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478174-drawablesize.
+func (ml MetalLayer) SetDrawableSize(width, height int) {
+	C.MetalLayer_SetDrawableSize(ml.metalLayer, C.double(width), C.double(height))
+}
+
+// NextDrawable returns a Metal drawable.
+//
+// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478172-nextdrawable.
+func (ml MetalLayer) NextDrawable() (MetalDrawable, error) {
+	md := C.MetalLayer_NextDrawable(ml.metalLayer)
+	if md == nil {
+		return MetalDrawable{}, errors.New("nextDrawable returned nil")
+	}
+
+	return MetalDrawable{md}, nil
+}
+
+// MetalDrawable is a displayable resource that can be rendered or written to by Metal.
+//
+// Reference: https://developer.apple.com/documentation/quartzcore/cametaldrawable.
+type MetalDrawable struct {
+	metalDrawable unsafe.Pointer
+}
+
+// Drawable implements the mtl.Drawable interface.
+func (md MetalDrawable) Drawable() unsafe.Pointer { return md.metalDrawable }
+
+// Texture returns a Metal texture object representing the drawable object's content.
+//
+// Reference: https://developer.apple.com/documentation/quartzcore/cametaldrawable/1478159-texture.
+func (md MetalDrawable) Texture() mtl.Texture {
+	return mtl.NewTexture(C.MetalDrawable_Texture(md.metalDrawable))
+}
diff --git a/example/movingtriangle/internal/ca/ca.h b/example/movingtriangle/internal/ca/ca.h
new file mode 100644
index 0000000..809898b
--- /dev/null
+++ b/example/movingtriangle/internal/ca/ca.h
@@ -0,0 +1,17 @@
+// +build darwin
+
+typedef signed char BOOL;
+typedef unsigned long uint_t;
+typedef unsigned short uint16_t;
+
+void * MakeMetalLayer();
+
+uint16_t     MetalLayer_PixelFormat(void * metalLayer);
+void         MetalLayer_SetDevice(void * metalLayer, void * device);
+const char * MetalLayer_SetPixelFormat(void * metalLayer, uint16_t pixelFormat);
+const char * MetalLayer_SetMaximumDrawableCount(void * metalLayer, uint_t maximumDrawableCount);
+void         MetalLayer_SetDisplaySyncEnabled(void * metalLayer, BOOL displaySyncEnabled);
+void         MetalLayer_SetDrawableSize(void * metalLayer, double width, double height);
+void *       MetalLayer_NextDrawable(void * metalLayer);
+
+void * MetalDrawable_Texture(void * drawable);
diff --git a/example/movingtriangle/internal/ca/ca.m b/example/movingtriangle/internal/ca/ca.m
new file mode 100644
index 0000000..45d14f7
--- /dev/null
+++ b/example/movingtriangle/internal/ca/ca.m
@@ -0,0 +1,54 @@
+// +build darwin
+
+#import <QuartzCore/QuartzCore.h>
+#include "ca.h"
+
+void * MakeMetalLayer() {
+	return [[CAMetalLayer alloc] init];
+}
+
+uint16_t MetalLayer_PixelFormat(void * metalLayer) {
+	return ((CAMetalLayer *)metalLayer).pixelFormat;
+}
+
+void MetalLayer_SetDevice(void * metalLayer, void * device) {
+	((CAMetalLayer *)metalLayer).device = (id<MTLDevice>)device;
+}
+
+const char * MetalLayer_SetPixelFormat(void * metalLayer, uint16_t pixelFormat) {
+	@try {
+		((CAMetalLayer *)metalLayer).pixelFormat = (MTLPixelFormat)pixelFormat;
+	}
+	@catch (NSException * exception) {
+		return exception.reason.UTF8String;
+	}
+	return NULL;
+}
+
+const char * MetalLayer_SetMaximumDrawableCount(void * metalLayer, uint_t maximumDrawableCount) {
+	if (@available(macOS 10.13.2, *)) {
+		@try {
+			((CAMetalLayer *)metalLayer).maximumDrawableCount = (NSUInteger)maximumDrawableCount;
+		}
+		@catch (NSException * exception) {
+			return exception.reason.UTF8String;
+		}
+	}
+	return NULL;
+}
+
+void MetalLayer_SetDisplaySyncEnabled(void * metalLayer, BOOL displaySyncEnabled) {
+	((CAMetalLayer *)metalLayer).displaySyncEnabled = displaySyncEnabled;
+}
+
+void MetalLayer_SetDrawableSize(void * metalLayer, double width, double height) {
+	((CAMetalLayer *)metalLayer).drawableSize = (CGSize){width, height};
+}
+
+void * MetalLayer_NextDrawable(void * metalLayer) {
+	return [(CAMetalLayer *)metalLayer nextDrawable];
+}
+
+void * MetalDrawable_Texture(void * metalDrawable) {
+	return ((id<CAMetalDrawable>)metalDrawable).texture;
+}
diff --git a/example/movingtriangle/internal/ns/ns.go b/example/movingtriangle/internal/ns/ns.go
new file mode 100644
index 0000000..e8d2993
--- /dev/null
+++ b/example/movingtriangle/internal/ns/ns.go
@@ -0,0 +1,65 @@
+// +build darwin
+
+// Package ns provides access to Apple's AppKit API (https://developer.apple.com/documentation/appkit).
+//
+// This package is in very early stages of development.
+// It's a minimal implementation with scope limited to
+// supporting the movingtriangle example.
+package ns
+
+import (
+	"unsafe"
+
+	"dmitri.shuralyov.com/gpu/mtl/example/movingtriangle/internal/ca"
+)
+
+/*
+#include "ns.h"
+*/
+import "C"
+
+// Window is a window that an app displays on the screen.
+//
+// Reference: https://developer.apple.com/documentation/appkit/nswindow.
+type Window struct {
+	window unsafe.Pointer
+}
+
+// NewWindow returns a Window that wraps an existing NSWindow * pointer.
+func NewWindow(window unsafe.Pointer) Window {
+	return Window{window}
+}
+
+// ContentView returns the window's content view, the highest accessible View
+// in the window's view hierarchy.
+//
+// Reference: https://developer.apple.com/documentation/appkit/nswindow/1419160-contentview.
+func (w Window) ContentView() View {
+	return View{C.Window_ContentView(w.window)}
+}
+
+// View is the infrastructure for drawing, printing, and handling events in an app.
+//
+// Reference: https://developer.apple.com/documentation/appkit/nsview.
+type View struct {
+	view unsafe.Pointer
+}
+
+// SetLayer sets v.layer to l.
+//
+// Reference: https://developer.apple.com/documentation/appkit/nsview/1483298-layer.
+func (v View) SetLayer(l ca.Layer) {
+	C.View_SetLayer(v.view, l.Layer())
+}
+
+// SetWantsLayer sets v.wantsLayer to wantsLayer.
+//
+// Reference: https://developer.apple.com/documentation/appkit/nsview/1483695-wantslayer.
+func (v View) SetWantsLayer(wantsLayer bool) {
+	switch wantsLayer {
+	case true:
+		C.View_SetWantsLayer(v.view, 1)
+	case false:
+		C.View_SetWantsLayer(v.view, 0)
+	}
+}
diff --git a/example/movingtriangle/internal/ns/ns.h b/example/movingtriangle/internal/ns/ns.h
new file mode 100644
index 0000000..42ceb6a
--- /dev/null
+++ b/example/movingtriangle/internal/ns/ns.h
@@ -0,0 +1,8 @@
+// +build darwin
+
+typedef signed char BOOL;
+
+void * Window_ContentView(void * window);
+
+void View_SetLayer(void * view, void * layer);
+void View_SetWantsLayer(void * view, BOOL wantsLayer);
diff --git a/example/movingtriangle/internal/ns/ns.m b/example/movingtriangle/internal/ns/ns.m
new file mode 100644
index 0000000..937836d
--- /dev/null
+++ b/example/movingtriangle/internal/ns/ns.m
@@ -0,0 +1,16 @@
+// +build darwin
+
+#import <Cocoa/Cocoa.h>
+#include "ns.h"
+
+void * Window_ContentView(void * window) {
+	return ((NSWindow *)window).contentView;
+}
+
+void View_SetLayer(void * view, void * layer) {
+	((NSView *)view).layer = (CALayer *)layer;
+}
+
+void View_SetWantsLayer(void * view, BOOL wantsLayer) {
+	((NSView *)view).wantsLayer = wantsLayer;
+}
diff --git a/example/movingtriangle/main.go b/example/movingtriangle/main.go
new file mode 100644
index 0000000..cf2aa35
--- /dev/null
+++ b/example/movingtriangle/main.go
@@ -0,0 +1,198 @@
+// +build darwin
+
+// movingtriangle is an example Metal program that displays a moving triangle in a window.
+// It opens a window and renders a triangle that follows the mouse cursor.
+package main
+
+import (
+	"flag"
+	"fmt"
+	"log"
+	"os"
+	"runtime"
+	"time"
+	"unsafe"
+
+	"dmitri.shuralyov.com/gpu/mtl"
+	"dmitri.shuralyov.com/gpu/mtl/example/movingtriangle/internal/ca"
+	"dmitri.shuralyov.com/gpu/mtl/example/movingtriangle/internal/ns"
+	"github.com/go-gl/glfw/v3.2/glfw"
+	"golang.org/x/image/math/f32"
+)
+
+func init() {
+	runtime.LockOSThread()
+}
+
+func main() {
+	flag.Usage = func() {
+		fmt.Fprintln(os.Stderr, "Usage: movingtriangle")
+		flag.PrintDefaults()
+	}
+	flag.Parse()
+
+	err := run()
+	if err != nil {
+		log.Fatalln(err)
+	}
+}
+
+func run() error {
+	device, err := mtl.CreateSystemDefaultDevice()
+	if err != nil {
+		return err
+	}
+	fmt.Println("Metal device:", device.Name)
+
+	err = glfw.Init()
+	if err != nil {
+		return err
+	}
+	defer glfw.Terminate()
+
+	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
+	window, err := glfw.CreateWindow(640, 480, "Metal Example", nil, nil)
+	if err != nil {
+		return err
+	}
+	defer window.Destroy()
+
+	ml := ca.MakeMetalLayer()
+	ml.SetDevice(device)
+	ml.SetPixelFormat(mtl.PixelFormatBGRA8UNorm)
+	ml.SetDrawableSize(window.GetFramebufferSize())
+	ml.SetMaximumDrawableCount(3)
+	ml.SetDisplaySyncEnabled(true)
+	cocoaWindow := ns.NewWindow(unsafe.Pointer(window.GetCocoaWindow()))
+	cocoaWindow.ContentView().SetLayer(ml)
+	cocoaWindow.ContentView().SetWantsLayer(true)
+
+	// Set callbacks.
+	window.SetFramebufferSizeCallback(func(_ *glfw.Window, width, height int) {
+		ml.SetDrawableSize(width, height)
+	})
+	var windowSize = [2]int32{640, 480}
+	window.SetSizeCallback(func(_ *glfw.Window, width, height int) {
+		windowSize[0], windowSize[1] = int32(width), int32(height)
+	})
+	var pos [2]float32
+	window.SetCursorPosCallback(func(_ *glfw.Window, x, y float64) {
+		pos[0], pos[1] = float32(x), float32(y)
+	})
+
+	// Create a render pipeline state.
+	const source = ` + "`" + `#include <metal_stdlib>
+
+using namespace metal;
+
+struct Vertex {
+	float4 position [[position]];
+	float4 color;
+};
+
+vertex Vertex VertexShader(
+	uint vertexID [[vertex_id]],
+	device Vertex * vertices [[buffer(0)]],
+	constant int2 * windowSize [[buffer(1)]],
+	constant float2 * pos [[buffer(2)]]
+) {
+	Vertex out = vertices[vertexID];
+	out.position.xy += *pos;
+	float2 viewportSize = float2(*windowSize);
+	out.position.xy = float2(-1 + out.position.x / (0.5 * viewportSize.x),
+	                          1 - out.position.y / (0.5 * viewportSize.y));
+	return out;
+}
+
+fragment float4 FragmentShader(Vertex in [[stage_in]]) {
+	return in.color;
+}
+` + "`" + `
+	lib, err := device.MakeLibrary(source, mtl.CompileOptions{})
+	if err != nil {
+		return err
+	}
+	vs, err := lib.MakeFunction("VertexShader")
+	if err != nil {
+		return err
+	}
+	fs, err := lib.MakeFunction("FragmentShader")
+	if err != nil {
+		return err
+	}
+	var rpld mtl.RenderPipelineDescriptor
+	rpld.VertexFunction = vs
+	rpld.FragmentFunction = fs
+	rpld.ColorAttachments[0].PixelFormat = ml.PixelFormat()
+	rps, err := device.MakeRenderPipelineState(rpld)
+	if err != nil {
+		return err
+	}
+
+	// Create a vertex buffer.
+	type Vertex struct {
+		Position f32.Vec4
+		Color    f32.Vec4
+	}
+	vertexData := [...]Vertex{
+		{f32.Vec4{0, 0, 0, 1}, f32.Vec4{1, 0, 0, 1}},
+		{f32.Vec4{300, 100, 0, 1}, f32.Vec4{0, 1, 0, 1}},
+		{f32.Vec4{0, 100, 0, 1}, f32.Vec4{0, 0, 1, 1}},
+	}
+	vertexBuffer := device.MakeBuffer(unsafe.Pointer(&vertexData[0]), unsafe.Sizeof(vertexData), mtl.ResourceStorageModeManaged)
+
+	cq := device.MakeCommandQueue()
+
+	frame := startFPSCounter()
+
+	for !window.ShouldClose() {
+		glfw.PollEvents()
+
+		// Create a drawable to render into.
+		drawable, err := ml.NextDrawable()
+		if err != nil {
+			return err
+		}
+
+		cb := cq.MakeCommandBuffer()
+
+		// Encode all render commands.
+		var rpd mtl.RenderPassDescriptor
+		rpd.ColorAttachments[0].LoadAction = mtl.LoadActionClear
+		rpd.ColorAttachments[0].StoreAction = mtl.StoreActionStore
+		rpd.ColorAttachments[0].ClearColor = mtl.ClearColor{Red: 0.35, Green: 0.65, Blue: 0.85, Alpha: 1}
+		rpd.ColorAttachments[0].Texture = drawable.Texture()
+		rce := cb.MakeRenderCommandEncoder(rpd)
+		rce.SetRenderPipelineState(rps)
+		rce.SetVertexBuffer(vertexBuffer, 0, 0)
+		rce.SetVertexBytes(unsafe.Pointer(&windowSize[0]), unsafe.Sizeof(windowSize), 1)
+		rce.SetVertexBytes(unsafe.Pointer(&pos[0]), unsafe.Sizeof(pos), 2)
+		rce.DrawPrimitives(mtl.PrimitiveTypeTriangle, 0, 3)
+		rce.EndEncoding()
+
+		cb.PresentDrawable(drawable)
+		cb.Commit()
+
+		frame <- struct{}{}
+	}
+
+	return nil
+}
+
+func startFPSCounter() chan struct{} {
+	frame := make(chan struct{}, 4)
+	go func() {
+		second := time.Tick(time.Second)
+		frames := 0
+		for {
+			select {
+			case <-second:
+				fmt.Println("fps:", frames)
+				frames = 0
+			case <-frame:
+				frames++
+			}
+		}
+	}()
+	return frame
+}
diff --git a/mtl.go b/mtl.go
index feff4bb..9c66681 100644
--- a/mtl.go
+++ b/mtl.go
@@ -15,7 +15,6 @@ import (
 )

 /*
-#cgo CFLAGS: -x objective-c
 #cgo LDFLAGS: -framework Metal -framework Foundation
 #include <stdlib.h>
 #include "mtl.h"
@@ -49,8 +48,9 @@ type PixelFormat uint8
 // The data formats that describe the organization and characteristics
 // of individual pixels in a texture.
 const (
-	PixelFormatRGBA8UNorm PixelFormat = 70 // Ordinary format with four 8-bit normalized unsigned integer components in RGBA order.
-	PixelFormatBGRA8UNorm PixelFormat = 80 // Ordinary format with four 8-bit normalized unsigned integer components in BGRA order.
+	PixelFormatRGBA8UNorm     PixelFormat = 70 // Ordinary format with four 8-bit normalized unsigned integer components in RGBA order.
+	PixelFormatBGRA8UNorm     PixelFormat = 80 // Ordinary format with four 8-bit normalized unsigned integer components in BGRA order.
+	PixelFormatBGRA8UNormSRGB PixelFormat = 81 // Ordinary format with four 8-bit normalized unsigned integer components in BGRA order with conversion between sRGB and linear space.
 )

 // PrimitiveType defines geometric primitive types for drawing commands.
@@ -193,6 +193,7 @@ const (
 //
 // Reference: https://developer.apple.com/documentation/metal/mtlresource.
 type Resource interface {
+	// resource returns the underlying id<MTLResource> pointer.
 	resource() unsafe.Pointer
 }

@@ -215,7 +216,7 @@ type RenderPipelineDescriptor struct {
 //
 // Reference: https://developer.apple.com/documentation/metal/mtlrenderpipelinecolorattachmentdescriptor.
 type RenderPipelineColorAttachmentDescriptor struct {
-	// PixelFormat is the pixel format of the color attachment’s texture.
+	// PixelFormat is the pixel format of the color attachment's texture.
 	PixelFormat PixelFormat
 }

@@ -327,6 +328,9 @@ func CopyAllDevices() []Device {
 	return ds
 }

+// Device returns the underlying id<MTLDevice> pointer.
+func (d Device) Device() unsafe.Pointer { return d.device }
+
 // SupportsFeatureSet reports whether device d supports feature set fs.
 //
 // Reference: https://developer.apple.com/documentation/metal/mtldevice/1433418-supportsfeatureset.
@@ -405,6 +409,14 @@ type CompileOptions struct {
 	// TODO.
 }

+// Drawable is a displayable resource that can be rendered or written to.
+//
+// Reference: https://developer.apple.com/documentation/metal/mtldrawable.
+type Drawable interface {
+	// Drawable returns the underlying id<MTLDrawable> pointer.
+	Drawable() unsafe.Pointer
+}
+
 // CommandQueue is a queue that organizes the order
 // in which command buffers are executed by the GPU.
 //
@@ -428,6 +440,13 @@ type CommandBuffer struct {
 	commandBuffer unsafe.Pointer
 }

+// PresentDrawable registers a drawable presentation to occur as soon as possible.
+//
+// Reference: https://developer.apple.com/documentation/metal/mtlcommandbuffer/1443029-presentdrawable.
+func (cb CommandBuffer) PresentDrawable(d Drawable) {
+	C.CommandBuffer_PresentDrawable(cb.commandBuffer, d.Drawable())
+}
+
 // Commit commits this command buffer for execution as soon as possible.
 //
 // Reference: https://developer.apple.com/documentation/metal/mtlcommandbuffer/1443003-commit.
@@ -507,6 +526,13 @@ func (rce RenderCommandEncoder) SetVertexBuffer(buf Buffer, offset, index int) {
 	C.RenderCommandEncoder_SetVertexBuffer(rce.commandEncoder, buf.buffer, C.uint_t(offset), C.uint_t(index))
 }

+// SetVertexBytes sets a block of data for the vertex function.
+//
+// Reference: https://developer.apple.com/documentation/metal/mtlrendercommandencoder/1515846-setvertexbytes.
+func (rce RenderCommandEncoder) SetVertexBytes(bytes unsafe.Pointer, length uintptr, index int) {
+	C.RenderCommandEncoder_SetVertexBytes(rce.commandEncoder, bytes, C.size_t(length), C.uint_t(index))
+}
+
 // DrawPrimitives renders one instance of primitives using vertex data
 // in contiguous array elements.
 //
@@ -557,6 +583,8 @@ func (l Library) MakeFunction(name string) (Function, error) {
 type Texture struct {
 	texture unsafe.Pointer

+	// TODO: Change these fields into methods.
+
 	// Width is the width of the texture image for the base level mipmap, in pixels.
 	Width int

@@ -564,6 +592,12 @@ type Texture struct {
 	Height int
 }

+// NewTexture returns a Texture that wraps an existing id<MTLTexture> pointer.
+func NewTexture(texture unsafe.Pointer) Texture {
+	return Texture{texture: texture}
+}
+
+// resource implements the Resource interface.
 func (t Texture) resource() unsafe.Pointer { return t.texture }

 // GetBytes copies a block of pixels from the storage allocation of texture
diff --git a/mtl.h b/mtl.h
index 6ac8b18..f7c4c67 100644
--- a/mtl.h
+++ b/mtl.h
@@ -86,6 +86,7 @@ void *                     Device_MakeTexture(void * device, struct TextureDescr

 void * CommandQueue_MakeCommandBuffer(void * commandQueue);

+void   CommandBuffer_PresentDrawable(void * commandBuffer, void * drawable);
 void   CommandBuffer_Commit(void * commandBuffer);
 void   CommandBuffer_WaitUntilCompleted(void * commandBuffer);
 void * CommandBuffer_MakeRenderCommandEncoder(void * commandBuffer, struct RenderPassDescriptor descriptor);
@@ -95,6 +96,7 @@ void CommandEncoder_EndEncoding(void * commandEncoder);

 void RenderCommandEncoder_SetRenderPipelineState(void * renderCommandEncoder, void * renderPipelineState);
 void RenderCommandEncoder_SetVertexBuffer(void * renderCommandEncoder, void * buffer, uint_t offset, uint_t index);
+void RenderCommandEncoder_SetVertexBytes(void * renderCommandEncoder, const void * bytes, size_t length, uint_t index);
 void RenderCommandEncoder_DrawPrimitives(void * renderCommandEncoder, uint8_t primitiveType, uint_t vertexStart, uint_t vertexCount);

 void BlitCommandEncoder_Synchronize(void * blitCommandEncoder, void * resource);
diff --git a/mtl.m b/mtl.m
index b3126d6..4296744 100644
--- a/mtl.m
+++ b/mtl.m
@@ -1,7 +1,7 @@
 // +build darwin

-#include <stdlib.h>
 #import <Metal/Metal.h>
+#include <stdlib.h>
 #include "mtl.h"

 struct Device CreateSystemDefaultDevice() {
@@ -100,6 +100,10 @@ struct RenderPipelineState Device_MakeRenderPipelineState(void * device, struct
 	return [(id<MTLCommandQueue>)commandQueue commandBuffer];
 }

+void CommandBuffer_PresentDrawable(void * commandBuffer, void * drawable) {
+	[(id<MTLCommandBuffer>)commandBuffer presentDrawable:(id<MTLDrawable>)drawable];
+}
+
 void CommandBuffer_Commit(void * commandBuffer) {
 	[(id<MTLCommandBuffer>)commandBuffer commit];
 }
@@ -136,14 +140,20 @@ void RenderCommandEncoder_SetRenderPipelineState(void * renderCommandEncoder, vo

 void RenderCommandEncoder_SetVertexBuffer(void * renderCommandEncoder, void * buffer, uint_t offset, uint_t index) {
 	[(id<MTLRenderCommandEncoder>)renderCommandEncoder setVertexBuffer:(id<MTLBuffer>)buffer
-	                                                            offset:offset
-	                                                           atIndex:index];
+	                                                            offset:(NSUInteger)offset
+	                                                           atIndex:(NSUInteger)index];
+}
+
+void RenderCommandEncoder_SetVertexBytes(void * renderCommandEncoder, const void * bytes, size_t length, uint_t index) {
+	[(id<MTLRenderCommandEncoder>)renderCommandEncoder setVertexBytes:bytes
+	                                                           length:(NSUInteger)length
+	                                                          atIndex:(NSUInteger)index];
 }

 void RenderCommandEncoder_DrawPrimitives(void * renderCommandEncoder, uint8_t primitiveType, uint_t vertexStart, uint_t vertexCount) {
-	[(id<MTLRenderCommandEncoder>)renderCommandEncoder drawPrimitives:primitiveType
-	                                                      vertexStart:vertexStart
-	                                                      vertexCount:vertexCount];
+	[(id<MTLRenderCommandEncoder>)renderCommandEncoder drawPrimitives:(MTLPrimitiveType)primitiveType
+	                                                      vertexStart:(NSUInteger)vertexStart
+	                                                      vertexCount:(NSUInteger)vertexCount];
 }

 void BlitCommandEncoder_Synchronize(void * blitCommandEncoder, void * resource) {
`

const diffMtlCommit1 = `diff --git a/example/movingtriangle/main.go b/example/movingtriangle/main.go
new file mode 100644
index 0000000..18b5e03
--- /dev/null
+++ b/example/movingtriangle/main.go
@@ -0,0 +1,196 @@
+// +build darwin
+
+// movingtriangle is an example Metal program that displays a moving triangle in a window.
+package main
+
+import (
+	"flag"
+	"fmt"
+	"log"
+	"os"
+	"runtime"
+	"time"
+	"unsafe"
+
+	"dmitri.shuralyov.com/gpu/mtl"
+	"github.com/go-gl/glfw/v3.2/glfw"
+	"golang.org/x/image/math/f32"
+)
+
+func usage() {
+	fmt.Fprintln(os.Stderr, "Usage: movingtriangle")
+	flag.PrintDefaults()
+}
+
+func init() {
+	runtime.LockOSThread()
+}
+
+func main() {
+	flag.Usage = usage
+	flag.Parse()
+
+	err := run()
+	if err != nil {
+		log.Fatalln(err)
+	}
+}
+
+func run() error {
+	device, err := mtl.CreateSystemDefaultDevice()
+	if err != nil {
+		return err
+	}
+	fmt.Println("Metal device:", device.Name)
+
+	err = glfw.Init()
+	if err != nil {
+		return err
+	}
+	defer glfw.Terminate()
+
+	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
+	window, err := glfw.CreateWindow(640, 480, "Metal Example", nil, nil)
+	if err != nil {
+		return err
+	}
+	defer window.Destroy()
+
+	layer := mtl.MakeLayer()
+	layer.SetDevice(device)
+	layer.SetPixelFormat(mtl.PixelFormatBGRA8UNorm)
+	layer.SetDrawableSize(window.GetFramebufferSize())
+	layer.SetMaximumDrawableCount(3)
+	layer.SetDisplaySyncEnabled(true)
+	mtl.SetWindowContentViewLayer(window.GetCocoaWindow(), layer)
+	mtl.SetWindowContentViewWantsLayer(window.GetCocoaWindow(), true)
+
+	// Set callbacks.
+	window.SetFramebufferSizeCallback(func(_ *glfw.Window, width, height int) {
+		layer.SetDrawableSize(width, height)
+	})
+	var windowSize = [2]int32{640, 480}
+	window.SetSizeCallback(func(_ *glfw.Window, width, height int) {
+		windowSize[0], windowSize[1] = int32(width), int32(height)
+	})
+	var pos [2]float32
+	window.SetCursorPosCallback(func(_ *glfw.Window, x, y float64) {
+		pos[0], pos[1] = float32(x), float32(y)
+	})
+
+	// Create a render pipeline state.
+	const source = ` + "`" + `#include <metal_stdlib>
+
+using namespace metal;
+
+struct Vertex {
+	float4 position [[position]];
+	float4 color;
+};
+
+vertex Vertex VertexShader(
+	uint vertexID [[vertex_id]],
+	device Vertex * vertices [[buffer(0)]],
+	constant int2 * windowSize [[buffer(1)]],
+	constant float2 * pos [[buffer(2)]]
+) {
+	Vertex out = vertices[vertexID];
+	out.position.xy += *pos;
+	float2 viewportSize = float2(*windowSize);
+	out.position.xy = float2(-1 + out.position.x / (0.5 * viewportSize.x),
+	                          1 - out.position.y / (0.5 * viewportSize.y));
+	return out;
+}
+
+fragment float4 FragmentShader(Vertex in [[stage_in]]) {
+	return in.color;
+}
+` + "`" + `
+	lib, err := device.MakeLibrary(source, mtl.CompileOptions{})
+	if err != nil {
+		return err
+	}
+	vs, err := lib.MakeFunction("VertexShader")
+	if err != nil {
+		return err
+	}
+	fs, err := lib.MakeFunction("FragmentShader")
+	if err != nil {
+		return err
+	}
+	var rpld mtl.RenderPipelineDescriptor
+	rpld.VertexFunction = vs
+	rpld.FragmentFunction = fs
+	rpld.ColorAttachments[0].PixelFormat = layer.PixelFormat()
+	rps, err := device.MakeRenderPipelineState(rpld)
+	if err != nil {
+		return err
+	}
+
+	// Create a vertex buffer.
+	type Vertex struct {
+		Position f32.Vec4
+		Color    f32.Vec4
+	}
+	vertexData := [...]Vertex{
+		{f32.Vec4{0, 0, 0, 1}, f32.Vec4{1, 0, 0, 1}},
+		{f32.Vec4{300, 100, 0, 1}, f32.Vec4{0, 1, 0, 1}},
+		{f32.Vec4{0, 100, 0, 1}, f32.Vec4{0, 0, 1, 1}},
+	}
+	vertexBuffer := device.MakeBuffer(unsafe.Pointer(&vertexData[0]), unsafe.Sizeof(vertexData), mtl.ResourceStorageModeManaged)
+
+	cq := device.MakeCommandQueue()
+
+	frame := startFPSCounter()
+
+	for !window.ShouldClose() {
+		glfw.PollEvents()
+
+		// Create a drawable to render into.
+		drawable, err := layer.NextDrawable()
+		if err != nil {
+			return err
+		}
+
+		cb := cq.MakeCommandBuffer()
+
+		// Encode all render commands.
+		var rpd mtl.RenderPassDescriptor
+		rpd.ColorAttachments[0].LoadAction = mtl.LoadActionClear
+		rpd.ColorAttachments[0].StoreAction = mtl.StoreActionStore
+		rpd.ColorAttachments[0].ClearColor = mtl.ClearColor{Red: 0.35, Green: 0.65, Blue: 0.85, Alpha: 1}
+		rpd.ColorAttachments[0].Texture = drawable.Texture()
+		rce := cb.MakeRenderCommandEncoder(rpd)
+		rce.SetRenderPipelineState(rps)
+		rce.SetVertexBuffer(vertexBuffer, 0, 0)
+		rce.SetVertexBytes(unsafe.Pointer(&windowSize[0]), unsafe.Sizeof(windowSize), 1)
+		rce.SetVertexBytes(unsafe.Pointer(&pos[0]), unsafe.Sizeof(pos), 2)
+		rce.DrawPrimitives(mtl.PrimitiveTypeTriangle, 0, 3)
+		rce.EndEncoding()
+
+		cb.Present(drawable)
+		cb.Commit()
+
+		frame <- struct{}{}
+	}
+
+	return nil
+}
+
+func startFPSCounter() chan struct{} {
+	frame := make(chan struct{}, 4)
+	go func() {
+		second := time.Tick(time.Second)
+		frames := 0
+		for {
+			select {
+			case <-second:
+				fmt.Println("fps:", frames)
+				frames = 0
+			case <-frame:
+				frames++
+			}
+		}
+	}()
+	return frame
+}
diff --git a/mtl.go b/mtl.go
index feff4bb..5ff54c5 100644
--- a/mtl.go
+++ b/mtl.go
@@ -16,7 +16,7 @@ import (

 /*
 #cgo CFLAGS: -x objective-c
-#cgo LDFLAGS: -framework Metal -framework Foundation
+#cgo LDFLAGS: -framework Metal -framework QuartzCore -framework Foundation
 #include <stdlib.h>
 #include "mtl.h"
 struct Library Go_Device_MakeLibrary(void * device, _GoString_ source) {
@@ -25,6 +25,124 @@ struct Library Go_Device_MakeLibrary(void * device, _GoString_ source) {
 */
 import "C"

+// Layer is a Core Animation Metal layer, a layer that manages a pool of Metal drawables.
+//
+// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer.
+type Layer struct {
+	layer unsafe.Pointer
+}
+
+// MakeLayer creates a new Core Animation Metal layer.
+//
+// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer.
+func MakeLayer() Layer {
+	return Layer{C.MakeLayer()}
+}
+
+// PixelFormat returns the pixel format of textures for rendering layer content.
+//
+// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478155-pixelformat.
+func (l Layer) PixelFormat() PixelFormat {
+	return PixelFormat(C.Layer_PixelFormat(l.layer))
+}
+
+// SetDevice sets the Metal device responsible for the layer's drawable resources.
+//
+// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478163-device.
+func (l Layer) SetDevice(device Device) {
+	C.Layer_SetDevice(l.layer, device.device)
+}
+
+// SetPixelFormat controls the pixel format of textures for rendering layer content.
+//
+// The pixel format for a Metal layer must be PixelFormatBGRA8UNorm, PixelFormatBGRA8UNormSRGB,
+// PixelFormatRGBA16Float, PixelFormatBGRA10XR, or PixelFormatBGRA10XRSRGB.
+// SetPixelFormat panics for other values.
+func (l Layer) SetPixelFormat(pf PixelFormat) {
+	e := C.Layer_SetPixelFormat(l.layer, C.uint16_t(pf))
+	if e != nil {
+		panic(errors.New(C.GoString(e)))
+	}
+}
+
+// SetMaximumDrawableCount controls the number of Metal drawables in the resource pool
+// managed by Core Animation.
+//
+// It can set to 2 or 3 only. SetMaximumDrawableCount panics for other values.
+func (l Layer) SetMaximumDrawableCount(count int) {
+	e := C.Layer_SetMaximumDrawableCount(l.layer, C.uint_t(count))
+	if e != nil {
+		panic(errors.New(C.GoString(e)))
+	}
+}
+
+// SetDisplaySyncEnabled controls whether the Metal layer and its drawables
+// are synchronized with the display's refresh rate.
+func (l Layer) SetDisplaySyncEnabled(enabled bool) {
+	switch enabled {
+	case true:
+		C.Layer_SetDisplaySyncEnabled(l.layer, 1)
+	case false:
+		C.Layer_SetDisplaySyncEnabled(l.layer, 0)
+	}
+}
+
+// SetDrawableSize sets the size, in pixels, of textures for rendering layer content.
+//
+// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478174-drawablesize.
+func (l Layer) SetDrawableSize(width, height int) {
+	C.Layer_SetDrawableSize(l.layer, C.double(width), C.double(height))
+}
+
+// NextDrawable returns a Metal drawable.
+//
+// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478172-nextdrawable.
+func (l Layer) NextDrawable() (Drawable, error) {
+	d := C.Layer_NextDrawable(l.layer)
+	if d == nil {
+		return Drawable{}, errors.New("nextDrawable returned nil")
+	}
+
+	return Drawable{d}, nil
+}
+
+// Drawable is a displayable resource that can be rendered or written to.
+//
+// Reference: https://developer.apple.com/documentation/metal/mtldrawable.
+type Drawable struct {
+	drawable unsafe.Pointer
+}
+
+// Texture returns a Metal texture object representing the drawable object's content.
+//
+// Reference: https://developer.apple.com/documentation/quartzcore/cametaldrawable/1478159-texture.
+func (d Drawable) Texture() Texture {
+	return Texture{
+		texture: C.Drawable_Texture(d.drawable),
+		Width:   0, // TODO: Fetch dimensions of actually created texture.
+		Height:  0, // TODO: Fetch dimensions of actually created texture.
+	}
+}
+
+// SetWindowContentViewLayer sets cocoaWindow's contentView's layer to layer.
+//
+// Reference: https://developer.apple.com/documentation/appkit/nsview/1483298-layer.
+func SetWindowContentViewLayer(cocoaWindow uintptr, l Layer) {
+	C.SetWindowContentViewLayer(unsafe.Pointer(cocoaWindow), l.layer)
+}
+
+// SetWindowContentViewWantsLayer sets cocoaWindow's contentView's wantsLayer to wantsLayer.
+//
+// Reference: https://developer.apple.com/documentation/appkit/nsview/1483695-wantslayer.
+func SetWindowContentViewWantsLayer(cocoaWindow uintptr, wantsLayer bool) {
+	switch wantsLayer {
+	case true:
+		C.SetWindowContentViewWantsLayer(unsafe.Pointer(cocoaWindow), 1)
+	case false:
+		C.SetWindowContentViewWantsLayer(unsafe.Pointer(cocoaWindow), 0)
+	}
+}
+
 // FeatureSet defines a specific platform, hardware, and software configuration.
 //
 // Reference: https://developer.apple.com/documentation/metal/mtlfeatureset.
@@ -49,8 +167,9 @@ type PixelFormat uint8
 // The data formats that describe the organization and characteristics
 // of individual pixels in a texture.
 const (
-	PixelFormatRGBA8UNorm PixelFormat = 70 // Ordinary format with four 8-bit normalized unsigned integer components in RGBA order.
-	PixelFormatBGRA8UNorm PixelFormat = 80 // Ordinary format with four 8-bit normalized unsigned integer components in BGRA order.
+	PixelFormatRGBA8UNorm     PixelFormat = 70 // Ordinary format with four 8-bit normalized unsigned integer components in RGBA order.
+	PixelFormatBGRA8UNorm     PixelFormat = 80 // Ordinary format with four 8-bit normalized unsigned integer components in BGRA order.
+	PixelFormatBGRA8UNormSRGB PixelFormat = 81 // Ordinary format with four 8-bit normalized unsigned integer components in BGRA order with conversion between sRGB and linear space.
 )

 // PrimitiveType defines geometric primitive types for drawing commands.
@@ -215,7 +334,7 @@ type RenderPipelineDescriptor struct {
 //
 // Reference: https://developer.apple.com/documentation/metal/mtlrenderpipelinecolorattachmentdescriptor.
 type RenderPipelineColorAttachmentDescriptor struct {
-	// PixelFormat is the pixel format of the color attachment’s texture.
+	// PixelFormat is the pixel format of the color attachment's texture.
 	PixelFormat PixelFormat
 }

@@ -428,6 +547,13 @@ type CommandBuffer struct {
 	commandBuffer unsafe.Pointer
 }

+// Present registers a drawable presentation to occur as soon as possible.
+//
+// Reference: https://developer.apple.com/documentation/metal/mtlcommandbuffer/1443029-presentdrawable.
+func (cb CommandBuffer) Present(d Drawable) {
+	C.CommandBuffer_Present(cb.commandBuffer, d.drawable)
+}
+
 // Commit commits this command buffer for execution as soon as possible.
 //
 // Reference: https://developer.apple.com/documentation/metal/mtlcommandbuffer/1443003-commit.
@@ -507,6 +633,13 @@ func (rce RenderCommandEncoder) SetVertexBuffer(buf Buffer, offset, index int) {
 	C.RenderCommandEncoder_SetVertexBuffer(rce.commandEncoder, buf.buffer, C.uint_t(offset), C.uint_t(index))
 }

+// SetVertexBytes sets a block of data for the vertex function.
+//
+// Reference: https://developer.apple.com/documentation/metal/mtlrendercommandencoder/1515846-setvertexbytes.
+func (rce RenderCommandEncoder) SetVertexBytes(bytes unsafe.Pointer, length uintptr, index int) {
+	C.RenderCommandEncoder_SetVertexBytes(rce.commandEncoder, bytes, C.size_t(length), C.uint_t(index))
+}
+
 // DrawPrimitives renders one instance of primitives using vertex data
 // in contiguous array elements.
 //
diff --git a/mtl.h b/mtl.h
index 6ac8b18..e8924ab 100644
--- a/mtl.h
+++ b/mtl.h
@@ -74,6 +74,21 @@ struct Region {
 	struct Size   Size;
 };

+void * MakeLayer();
+
+uint16_t     Layer_PixelFormat(void * layer);
+void         Layer_SetDevice(void * layer, void * device);
+const char * Layer_SetPixelFormat(void * layer, uint16_t pixelFormat);
+const char * Layer_SetMaximumDrawableCount(void * layer, uint_t maximumDrawableCount);
+void         Layer_SetDisplaySyncEnabled(void * layer, BOOL displaySyncEnabled);
+void         Layer_SetDrawableSize(void * layer, double width, double height);
+void *       Layer_NextDrawable(void * layer);
+
+void * Drawable_Texture(void * drawable);
+
+void SetWindowContentViewLayer(void * cocoaWindow, void * layer);
+void SetWindowContentViewWantsLayer(void * cocoaWindow, BOOL wantsLayer);
+
 struct Device CreateSystemDefaultDevice();
 struct Devices CopyAllDevices();

@@ -86,6 +101,7 @@ void *                     Device_MakeTexture(void * device, struct TextureDescr

 void * CommandQueue_MakeCommandBuffer(void * commandQueue);

+void   CommandBuffer_Present(void * commandBuffer, void * drawable);
 void   CommandBuffer_Commit(void * commandBuffer);
 void   CommandBuffer_WaitUntilCompleted(void * commandBuffer);
 void * CommandBuffer_MakeRenderCommandEncoder(void * commandBuffer, struct RenderPassDescriptor descriptor);
@@ -95,6 +111,7 @@ void CommandEncoder_EndEncoding(void * commandEncoder);

 void RenderCommandEncoder_SetRenderPipelineState(void * renderCommandEncoder, void * renderPipelineState);
 void RenderCommandEncoder_SetVertexBuffer(void * renderCommandEncoder, void * buffer, uint_t offset, uint_t index);
+void RenderCommandEncoder_SetVertexBytes(void * renderCommandEncoder, const void * bytes, size_t length, uint_t index);
 void RenderCommandEncoder_DrawPrimitives(void * renderCommandEncoder, uint8_t primitiveType, uint_t vertexStart, uint_t vertexCount);

 void BlitCommandEncoder_Synchronize(void * blitCommandEncoder, void * resource);
diff --git a/mtl.m b/mtl.m
index b3126d6..7b0fd96 100644
--- a/mtl.m
+++ b/mtl.m
@@ -1,9 +1,71 @@
 // +build darwin

-#include <stdlib.h>
 #import <Metal/Metal.h>
+#import <QuartzCore/QuartzCore.h>
+#import <Cocoa/Cocoa.h>
+
+#include <stdlib.h>
+
 #include "mtl.h"

+void * MakeLayer() {
+	return [[CAMetalLayer alloc] init];
+}
+
+uint16_t Layer_PixelFormat(void * layer) {
+	return ((CAMetalLayer *)layer).pixelFormat;
+}
+
+void Layer_SetDevice(void * layer, void * device) {
+	((CAMetalLayer *)layer).device = (id<MTLDevice>)device;
+}
+
+const char * Layer_SetPixelFormat(void * layer, uint16_t pixelFormat) {
+	@try {
+		((CAMetalLayer *)layer).pixelFormat = (MTLPixelFormat)pixelFormat;
+	}
+	@catch (NSException * exception) {
+		return exception.reason.UTF8String;
+	}
+	return NULL;
+}
+
+const char * Layer_SetMaximumDrawableCount(void * layer, uint_t maximumDrawableCount) {
+	if (@available(macOS 10.13.2, *)) {
+		@try {
+			((CAMetalLayer *)layer).maximumDrawableCount = (NSUInteger)maximumDrawableCount;
+		}
+		@catch (NSException * exception) {
+			return exception.reason.UTF8String;
+		}
+	}
+	return NULL;
+}
+
+void Layer_SetDisplaySyncEnabled(void * layer, BOOL displaySyncEnabled) {
+	((CAMetalLayer *)layer).displaySyncEnabled = displaySyncEnabled;
+}
+
+void Layer_SetDrawableSize(void * layer, double width, double height) {
+	((CAMetalLayer *)layer).drawableSize = (CGSize){width, height};
+}
+
+void * Layer_NextDrawable(void * layer) {
+	return [(CAMetalLayer *)layer nextDrawable];
+}
+
+void * Drawable_Texture(void * drawable) {
+	return ((id<CAMetalDrawable>)drawable).texture;
+}
+
+void SetWindowContentViewLayer(void * cocoaWindow, void * layer) {
+	((NSWindow *)cocoaWindow).contentView.layer = (CAMetalLayer *)layer;
+}
+
+void SetWindowContentViewWantsLayer(void * cocoaWindow, BOOL wantsLayer) {
+	((NSWindow *)cocoaWindow).contentView.wantsLayer = wantsLayer;
+}
+
 struct Device CreateSystemDefaultDevice() {
 	id<MTLDevice> device = MTLCreateSystemDefaultDevice();
 	if (!device) {
@@ -100,6 +162,10 @@ struct RenderPipelineState Device_MakeRenderPipelineState(void * device, struct
 	return [(id<MTLCommandQueue>)commandQueue commandBuffer];
 }

+void CommandBuffer_Present(void * commandBuffer, void * drawable) {
+	[(id<MTLCommandBuffer>)commandBuffer presentDrawable:(id<CAMetalDrawable>)drawable];
+}
+
 void CommandBuffer_Commit(void * commandBuffer) {
 	[(id<MTLCommandBuffer>)commandBuffer commit];
 }
@@ -136,14 +202,20 @@ void RenderCommandEncoder_SetRenderPipelineState(void * renderCommandEncoder, vo

 void RenderCommandEncoder_SetVertexBuffer(void * renderCommandEncoder, void * buffer, uint_t offset, uint_t index) {
 	[(id<MTLRenderCommandEncoder>)renderCommandEncoder setVertexBuffer:(id<MTLBuffer>)buffer
-	                                                            offset:offset
-	                                                           atIndex:index];
+	                                                            offset:(NSUInteger)offset
+	                                                           atIndex:(NSUInteger)index];
+}
+
+void RenderCommandEncoder_SetVertexBytes(void * renderCommandEncoder, const void * bytes, size_t length, uint_t index) {
+	[(id<MTLRenderCommandEncoder>)renderCommandEncoder setVertexBytes:bytes
+	                                                           length:(NSUInteger)length
+	                                                          atIndex:(NSUInteger)index];
 }

 void RenderCommandEncoder_DrawPrimitives(void * renderCommandEncoder, uint8_t primitiveType, uint_t vertexStart, uint_t vertexCount) {
-	[(id<MTLRenderCommandEncoder>)renderCommandEncoder drawPrimitives:primitiveType
-	                                                      vertexStart:vertexStart
-	                                                      vertexCount:vertexCount];
+	[(id<MTLRenderCommandEncoder>)renderCommandEncoder drawPrimitives:(MTLPrimitiveType)primitiveType
+	                                                      vertexStart:(NSUInteger)vertexStart
+	                                                      vertexCount:(NSUInteger)vertexCount];
 }

 void BlitCommandEncoder_Synchronize(void * blitCommandEncoder, void * resource) {
`

const diffMtlCommit2 = `diff --git a/mtl.go b/mtl.go
index 5ff54c5..4e3b9a7 100644
--- a/mtl.go
+++ b/mtl.go
@@ -58,6 +58,8 @@ func (l Layer) SetDevice(device Device) {
 // The pixel format for a Metal layer must be PixelFormatBGRA8UNorm, PixelFormatBGRA8UNormSRGB,
 // PixelFormatRGBA16Float, PixelFormatBGRA10XR, or PixelFormatBGRA10XRSRGB.
 // SetPixelFormat panics for other values.
+//
+// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478155-pixelformat.
 func (l Layer) SetPixelFormat(pf PixelFormat) {
 	e := C.Layer_SetPixelFormat(l.layer, C.uint16_t(pf))
 	if e != nil {
@@ -69,6 +71,8 @@ func (l Layer) SetPixelFormat(pf PixelFormat) {
 // managed by Core Animation.
 //
 // It can set to 2 or 3 only. SetMaximumDrawableCount panics for other values.
+//
+// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/2938720-maximumdrawablecount.
 func (l Layer) SetMaximumDrawableCount(count int) {
 	e := C.Layer_SetMaximumDrawableCount(l.layer, C.uint_t(count))
 	if e != nil {
@@ -78,6 +82,8 @@ func (l Layer) SetMaximumDrawableCount(count int) {

 // SetDisplaySyncEnabled controls whether the Metal layer and its drawables
 // are synchronized with the display's refresh rate.
+//
+// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/2887087-displaysyncenabled.
 func (l Layer) SetDisplaySyncEnabled(enabled bool) {
 	switch enabled {
 	case true:
`

const diffMtlCommit3 = `diff --git a/example/movingtriangle/main.go b/example/movingtriangle/main.go
index 18b5e03..b09a63d 100644
--- a/example/movingtriangle/main.go
+++ b/example/movingtriangle/main.go
@@ -13,21 +13,21 @@ import (
 	"unsafe"

 	"dmitri.shuralyov.com/gpu/mtl"
+	"dmitri.shuralyov.com/gpu/mtl/internal/ca"
+	"dmitri.shuralyov.com/gpu/mtl/internal/ns"
 	"github.com/go-gl/glfw/v3.2/glfw"
 	"golang.org/x/image/math/f32"
 )

-func usage() {
-	fmt.Fprintln(os.Stderr, "Usage: movingtriangle")
-	flag.PrintDefaults()
-}
-
 func init() {
 	runtime.LockOSThread()
 }

 func main() {
-	flag.Usage = usage
+	flag.Usage = func() {
+		fmt.Fprintln(os.Stderr, "Usage: movingtriangle")
+		flag.PrintDefaults()
+	}
 	flag.Parse()

 	err := run()
@@ -56,18 +56,19 @@ func run() error {
 	}
 	defer window.Destroy()

-	layer := mtl.MakeLayer()
-	layer.SetDevice(device)
-	layer.SetPixelFormat(mtl.PixelFormatBGRA8UNorm)
-	layer.SetDrawableSize(window.GetFramebufferSize())
-	layer.SetMaximumDrawableCount(3)
-	layer.SetDisplaySyncEnabled(true)
-	mtl.SetWindowContentViewLayer(window.GetCocoaWindow(), layer)
-	mtl.SetWindowContentViewWantsLayer(window.GetCocoaWindow(), true)
+	ml := ca.MakeMetalLayer()
+	ml.SetDevice(device)
+	ml.SetPixelFormat(mtl.PixelFormatBGRA8UNorm)
+	ml.SetDrawableSize(window.GetFramebufferSize())
+	ml.SetMaximumDrawableCount(3)
+	ml.SetDisplaySyncEnabled(true)
+	cocoaWindow := ns.NewWindow(unsafe.Pointer(window.GetCocoaWindow()))
+	cocoaWindow.ContentView().SetLayer(ml)
+	cocoaWindow.ContentView().SetWantsLayer(true)

 	// Set callbacks.
 	window.SetFramebufferSizeCallback(func(_ *glfw.Window, width, height int) {
-		layer.SetDrawableSize(width, height)
+		ml.SetDrawableSize(width, height)
 	})
 	var windowSize = [2]int32{640, 480}
 	window.SetSizeCallback(func(_ *glfw.Window, width, height int) {
@@ -121,7 +122,7 @@ fragment float4 FragmentShader(Vertex in [[stage_in]]) {
 	var rpld mtl.RenderPipelineDescriptor
 	rpld.VertexFunction = vs
 	rpld.FragmentFunction = fs
-	rpld.ColorAttachments[0].PixelFormat = layer.PixelFormat()
+	rpld.ColorAttachments[0].PixelFormat = ml.PixelFormat()
 	rps, err := device.MakeRenderPipelineState(rpld)
 	if err != nil {
 		return err
@@ -147,7 +148,7 @@ fragment float4 FragmentShader(Vertex in [[stage_in]]) {
 		glfw.PollEvents()

 		// Create a drawable to render into.
-		drawable, err := layer.NextDrawable()
+		drawable, err := ml.NextDrawable()
 		if err != nil {
 			return err
 		}
@@ -168,7 +169,7 @@ fragment float4 FragmentShader(Vertex in [[stage_in]]) {
 		rce.DrawPrimitives(mtl.PrimitiveTypeTriangle, 0, 3)
 		rce.EndEncoding()

-		cb.Present(drawable)
+		cb.PresentDrawable(drawable)
 		cb.Commit()

 		frame <- struct{}{}
diff --git a/internal/ca/ca.go b/internal/ca/ca.go
new file mode 100644
index 0000000..87afcc6
--- /dev/null
+++ b/internal/ca/ca.go
@@ -0,0 +1,137 @@
+// +build darwin
+
+// Package ca provides access to Apple's Core Animation API (https://developer.apple.com/documentation/quartzcore).
+//
+// This package is in very early stages of development.
+// It's a minimal implementation with scope limited to
+// supporting the ../../example/movingtriangle command.
+package ca
+
+import (
+	"errors"
+	"unsafe"
+
+	"dmitri.shuralyov.com/gpu/mtl"
+)
+
+/*
+#cgo LDFLAGS: -framework QuartzCore -framework Foundation
+#include "ca.h"
+*/
+import "C"
+
+// Layer is an object that manages image-based content and
+// allows you to perform animations on that content.
+//
+// Reference: https://developer.apple.com/documentation/quartzcore/calayer.
+type Layer interface {
+	// Layer returns the underlying CALayer * pointer.
+	Layer() unsafe.Pointer
+}
+
+// MetalLayer is a Core Animation Metal layer, a layer that manages a pool of Metal drawables.
+//
+// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer.
+type MetalLayer struct {
+	metalLayer unsafe.Pointer
+}
+
+// MakeMetalLayer creates a new Core Animation Metal layer.
+//
+// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer.
+func MakeMetalLayer() MetalLayer {
+	return MetalLayer{C.MakeMetalLayer()}
+}
+
+// Layer implements the Layer interface.
+func (ml MetalLayer) Layer() unsafe.Pointer { return ml.metalLayer }
+
+// PixelFormat returns the pixel format of textures for rendering layer content.
+//
+// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478155-pixelformat.
+func (ml MetalLayer) PixelFormat() mtl.PixelFormat {
+	return mtl.PixelFormat(C.MetalLayer_PixelFormat(ml.metalLayer))
+}
+
+// SetDevice sets the Metal device responsible for the layer's drawable resources.
+//
+// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478163-device.
+func (ml MetalLayer) SetDevice(device mtl.Device) {
+	C.MetalLayer_SetDevice(ml.metalLayer, device.Device())
+}
+
+// SetPixelFormat controls the pixel format of textures for rendering layer content.
+//
+// The pixel format for a Metal layer must be PixelFormatBGRA8UNorm, PixelFormatBGRA8UNormSRGB,
+// PixelFormatRGBA16Float, PixelFormatBGRA10XR, or PixelFormatBGRA10XRSRGB.
+// SetPixelFormat panics for other values.
+//
+// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478155-pixelformat.
+func (ml MetalLayer) SetPixelFormat(pf mtl.PixelFormat) {
+	e := C.MetalLayer_SetPixelFormat(ml.metalLayer, C.uint16_t(pf))
+	if e != nil {
+		panic(errors.New(C.GoString(e)))
+	}
+}
+
+// SetMaximumDrawableCount controls the number of Metal drawables in the resource pool
+// managed by Core Animation.
+//
+// It can set to 2 or 3 only. SetMaximumDrawableCount panics for other values.
+//
+// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/2938720-maximumdrawablecount.
+func (ml MetalLayer) SetMaximumDrawableCount(count int) {
+	e := C.MetalLayer_SetMaximumDrawableCount(ml.metalLayer, C.uint_t(count))
+	if e != nil {
+		panic(errors.New(C.GoString(e)))
+	}
+}
+
+// SetDisplaySyncEnabled controls whether the Metal layer and its drawables
+// are synchronized with the display's refresh rate.
+//
+// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/2887087-displaysyncenabled.
+func (ml MetalLayer) SetDisplaySyncEnabled(enabled bool) {
+	switch enabled {
+	case true:
+		C.MetalLayer_SetDisplaySyncEnabled(ml.metalLayer, 1)
+	case false:
+		C.MetalLayer_SetDisplaySyncEnabled(ml.metalLayer, 0)
+	}
+}
+
+// SetDrawableSize sets the size, in pixels, of textures for rendering layer content.
+//
+// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478174-drawablesize.
+func (ml MetalLayer) SetDrawableSize(width, height int) {
+	C.MetalLayer_SetDrawableSize(ml.metalLayer, C.double(width), C.double(height))
+}
+
+// NextDrawable returns a Metal drawable.
+//
+// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478172-nextdrawable.
+func (ml MetalLayer) NextDrawable() (MetalDrawable, error) {
+	md := C.MetalLayer_NextDrawable(ml.metalLayer)
+	if md == nil {
+		return MetalDrawable{}, errors.New("nextDrawable returned nil")
+	}
+
+	return MetalDrawable{md}, nil
+}
+
+// MetalDrawable is a displayable resource that can be rendered or written to by Metal.
+//
+// Reference: https://developer.apple.com/documentation/quartzcore/cametaldrawable.
+type MetalDrawable struct {
+	metalDrawable unsafe.Pointer
+}
+
+// Drawable implements the mtl.Drawable interface.
+func (md MetalDrawable) Drawable() unsafe.Pointer { return md.metalDrawable }
+
+// Texture returns a Metal texture object representing the drawable object's content.
+//
+// Reference: https://developer.apple.com/documentation/quartzcore/cametaldrawable/1478159-texture.
+func (md MetalDrawable) Texture() mtl.Texture {
+	return mtl.NewTexture(C.MetalDrawable_Texture(md.metalDrawable))
+}
diff --git a/internal/ca/ca.h b/internal/ca/ca.h
new file mode 100644
index 0000000..809898b
--- /dev/null
+++ b/internal/ca/ca.h
@@ -0,0 +1,17 @@
+// +build darwin
+
+typedef signed char BOOL;
+typedef unsigned long uint_t;
+typedef unsigned short uint16_t;
+
+void * MakeMetalLayer();
+
+uint16_t     MetalLayer_PixelFormat(void * metalLayer);
+void         MetalLayer_SetDevice(void * metalLayer, void * device);
+const char * MetalLayer_SetPixelFormat(void * metalLayer, uint16_t pixelFormat);
+const char * MetalLayer_SetMaximumDrawableCount(void * metalLayer, uint_t maximumDrawableCount);
+void         MetalLayer_SetDisplaySyncEnabled(void * metalLayer, BOOL displaySyncEnabled);
+void         MetalLayer_SetDrawableSize(void * metalLayer, double width, double height);
+void *       MetalLayer_NextDrawable(void * metalLayer);
+
+void * MetalDrawable_Texture(void * drawable);
diff --git a/internal/ca/ca.m b/internal/ca/ca.m
new file mode 100644
index 0000000..45d14f7
--- /dev/null
+++ b/internal/ca/ca.m
@@ -0,0 +1,54 @@
+// +build darwin
+
+#import <QuartzCore/QuartzCore.h>
+#include "ca.h"
+
+void * MakeMetalLayer() {
+	return [[CAMetalLayer alloc] init];
+}
+
+uint16_t MetalLayer_PixelFormat(void * metalLayer) {
+	return ((CAMetalLayer *)metalLayer).pixelFormat;
+}
+
+void MetalLayer_SetDevice(void * metalLayer, void * device) {
+	((CAMetalLayer *)metalLayer).device = (id<MTLDevice>)device;
+}
+
+const char * MetalLayer_SetPixelFormat(void * metalLayer, uint16_t pixelFormat) {
+	@try {
+		((CAMetalLayer *)metalLayer).pixelFormat = (MTLPixelFormat)pixelFormat;
+	}
+	@catch (NSException * exception) {
+		return exception.reason.UTF8String;
+	}
+	return NULL;
+}
+
+const char * MetalLayer_SetMaximumDrawableCount(void * metalLayer, uint_t maximumDrawableCount) {
+	if (@available(macOS 10.13.2, *)) {
+		@try {
+			((CAMetalLayer *)metalLayer).maximumDrawableCount = (NSUInteger)maximumDrawableCount;
+		}
+		@catch (NSException * exception) {
+			return exception.reason.UTF8String;
+		}
+	}
+	return NULL;
+}
+
+void MetalLayer_SetDisplaySyncEnabled(void * metalLayer, BOOL displaySyncEnabled) {
+	((CAMetalLayer *)metalLayer).displaySyncEnabled = displaySyncEnabled;
+}
+
+void MetalLayer_SetDrawableSize(void * metalLayer, double width, double height) {
+	((CAMetalLayer *)metalLayer).drawableSize = (CGSize){width, height};
+}
+
+void * MetalLayer_NextDrawable(void * metalLayer) {
+	return [(CAMetalLayer *)metalLayer nextDrawable];
+}
+
+void * MetalDrawable_Texture(void * metalDrawable) {
+	return ((id<CAMetalDrawable>)metalDrawable).texture;
+}
diff --git a/internal/ns/ns.go b/internal/ns/ns.go
new file mode 100644
index 0000000..b81157d
--- /dev/null
+++ b/internal/ns/ns.go
@@ -0,0 +1,65 @@
+// +build darwin
+
+// Package ns provides access to Apple's Cocoa API (https://developer.apple.com/documentation/appkit).
+//
+// This package is in very early stages of development.
+// It's a minimal implementation with scope limited to
+// supporting the ../../example/movingtriangle command.
+package ns
+
+import (
+	"unsafe"
+
+	"dmitri.shuralyov.com/gpu/mtl/internal/ca"
+)
+
+/*
+#include "ns.h"
+*/
+import "C"
+
+// Window is a window that an app displays on the screen.
+//
+// Reference: https://developer.apple.com/documentation/appkit/nswindow.
+type Window struct {
+	window unsafe.Pointer
+}
+
+// NewWindow returns a Window that wraps an existing NSWindow * pointer.
+func NewWindow(window unsafe.Pointer) Window {
+	return Window{window}
+}
+
+// ContentView returns the window's content view, the highest accessible View
+// in the window's view hierarchy.
+//
+// Reference: https://developer.apple.com/documentation/appkit/nswindow/1419160-contentview.
+func (w Window) ContentView() View {
+	return View{C.Window_ContentView(w.window)}
+}
+
+// View is the infrastructure for drawing, printing, and handling events in an app.
+//
+// Reference: https://developer.apple.com/documentation/appkit/nsview.
+type View struct {
+	view unsafe.Pointer
+}
+
+// SetLayer sets v.layer to l.
+//
+// Reference: https://developer.apple.com/documentation/appkit/nsview/1483298-layer.
+func (v View) SetLayer(l ca.Layer) {
+	C.View_SetLayer(v.view, l.Layer())
+}
+
+// SetWantsLayer sets v.wantsLayer to wantsLayer.
+//
+// Reference: https://developer.apple.com/documentation/appkit/nsview/1483695-wantslayer.
+func (v View) SetWantsLayer(wantsLayer bool) {
+	switch wantsLayer {
+	case true:
+		C.View_SetWantsLayer(v.view, 1)
+	case false:
+		C.View_SetWantsLayer(v.view, 0)
+	}
+}
diff --git a/internal/ns/ns.h b/internal/ns/ns.h
new file mode 100644
index 0000000..42ceb6a
--- /dev/null
+++ b/internal/ns/ns.h
@@ -0,0 +1,8 @@
+// +build darwin
+
+typedef signed char BOOL;
+
+void * Window_ContentView(void * window);
+
+void View_SetLayer(void * view, void * layer);
+void View_SetWantsLayer(void * view, BOOL wantsLayer);
diff --git a/internal/ns/ns.m b/internal/ns/ns.m
new file mode 100644
index 0000000..937836d
--- /dev/null
+++ b/internal/ns/ns.m
@@ -0,0 +1,16 @@
+// +build darwin
+
+#import <Cocoa/Cocoa.h>
+#include "ns.h"
+
+void * Window_ContentView(void * window) {
+	return ((NSWindow *)window).contentView;
+}
+
+void View_SetLayer(void * view, void * layer) {
+	((NSView *)view).layer = (CALayer *)layer;
+}
+
+void View_SetWantsLayer(void * view, BOOL wantsLayer) {
+	((NSView *)view).wantsLayer = wantsLayer;
+}
diff --git a/mtl.go b/mtl.go
index 4e3b9a7..9c66681 100644
--- a/mtl.go
+++ b/mtl.go
@@ -15,8 +15,7 @@ import (
 )

 /*
-#cgo CFLAGS: -x objective-c
-#cgo LDFLAGS: -framework Metal -framework QuartzCore -framework Foundation
+#cgo LDFLAGS: -framework Metal -framework Foundation
 #include <stdlib.h>
 #include "mtl.h"
 struct Library Go_Device_MakeLibrary(void * device, _GoString_ source) {
@@ -25,130 +24,6 @@ struct Library Go_Device_MakeLibrary(void * device, _GoString_ source) {
 */
 import "C"

-// Layer is a Core Animation Metal layer, a layer that manages a pool of Metal drawables.
-//
-// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer.
-type Layer struct {
-	layer unsafe.Pointer
-}
-
-// MakeLayer creates a new Core Animation Metal layer.
-//
-// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer.
-func MakeLayer() Layer {
-	return Layer{C.MakeLayer()}
-}
-
-// PixelFormat returns the pixel format of textures for rendering layer content.
-//
-// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478155-pixelformat.
-func (l Layer) PixelFormat() PixelFormat {
-	return PixelFormat(C.Layer_PixelFormat(l.layer))
-}
-
-// SetDevice sets the Metal device responsible for the layer's drawable resources.
-//
-// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478163-device.
-func (l Layer) SetDevice(device Device) {
-	C.Layer_SetDevice(l.layer, device.device)
-}
-
-// SetPixelFormat controls the pixel format of textures for rendering layer content.
-//
-// The pixel format for a Metal layer must be PixelFormatBGRA8UNorm, PixelFormatBGRA8UNormSRGB,
-// PixelFormatRGBA16Float, PixelFormatBGRA10XR, or PixelFormatBGRA10XRSRGB.
-// SetPixelFormat panics for other values.
-//
-// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478155-pixelformat.
-func (l Layer) SetPixelFormat(pf PixelFormat) {
-	e := C.Layer_SetPixelFormat(l.layer, C.uint16_t(pf))
-	if e != nil {
-		panic(errors.New(C.GoString(e)))
-	}
-}
-
-// SetMaximumDrawableCount controls the number of Metal drawables in the resource pool
-// managed by Core Animation.
-//
-// It can set to 2 or 3 only. SetMaximumDrawableCount panics for other values.
-//
-// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/2938720-maximumdrawablecount.
-func (l Layer) SetMaximumDrawableCount(count int) {
-	e := C.Layer_SetMaximumDrawableCount(l.layer, C.uint_t(count))
-	if e != nil {
-		panic(errors.New(C.GoString(e)))
-	}
-}
-
-// SetDisplaySyncEnabled controls whether the Metal layer and its drawables
-// are synchronized with the display's refresh rate.
-//
-// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/2887087-displaysyncenabled.
-func (l Layer) SetDisplaySyncEnabled(enabled bool) {
-	switch enabled {
-	case true:
-		C.Layer_SetDisplaySyncEnabled(l.layer, 1)
-	case false:
-		C.Layer_SetDisplaySyncEnabled(l.layer, 0)
-	}
-}
-
-// SetDrawableSize sets the size, in pixels, of textures for rendering layer content.
-//
-// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478174-drawablesize.
-func (l Layer) SetDrawableSize(width, height int) {
-	C.Layer_SetDrawableSize(l.layer, C.double(width), C.double(height))
-}
-
-// NextDrawable returns a Metal drawable.
-//
-// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478172-nextdrawable.
-func (l Layer) NextDrawable() (Drawable, error) {
-	d := C.Layer_NextDrawable(l.layer)
-	if d == nil {
-		return Drawable{}, errors.New("nextDrawable returned nil")
-	}
-
-	return Drawable{d}, nil
-}
-
-// Drawable is a displayable resource that can be rendered or written to.
-//
-// Reference: https://developer.apple.com/documentation/metal/mtldrawable.
-type Drawable struct {
-	drawable unsafe.Pointer
-}
-
-// Texture returns a Metal texture object representing the drawable object's content.
-//
-// Reference: https://developer.apple.com/documentation/quartzcore/cametaldrawable/1478159-texture.
-func (d Drawable) Texture() Texture {
-	return Texture{
-		texture: C.Drawable_Texture(d.drawable),
-		Width:   0, // TODO: Fetch dimensions of actually created texture.
-		Height:  0, // TODO: Fetch dimensions of actually created texture.
-	}
-}
-
-// SetWindowContentViewLayer sets cocoaWindow's contentView's layer to layer.
-//
-// Reference: https://developer.apple.com/documentation/appkit/nsview/1483298-layer.
-func SetWindowContentViewLayer(cocoaWindow uintptr, l Layer) {
-	C.SetWindowContentViewLayer(unsafe.Pointer(cocoaWindow), l.layer)
-}
-
-// SetWindowContentViewWantsLayer sets cocoaWindow's contentView's wantsLayer to wantsLayer.
-//
-// Reference: https://developer.apple.com/documentation/appkit/nsview/1483695-wantslayer.
-func SetWindowContentViewWantsLayer(cocoaWindow uintptr, wantsLayer bool) {
-	switch wantsLayer {
-	case true:
-		C.SetWindowContentViewWantsLayer(unsafe.Pointer(cocoaWindow), 1)
-	case false:
-		C.SetWindowContentViewWantsLayer(unsafe.Pointer(cocoaWindow), 0)
-	}
-}
-
 // FeatureSet defines a specific platform, hardware, and software configuration.
 //
 // Reference: https://developer.apple.com/documentation/metal/mtlfeatureset.
@@ -318,6 +193,7 @@ const (
 //
 // Reference: https://developer.apple.com/documentation/metal/mtlresource.
 type Resource interface {
+	// resource returns the underlying id<MTLResource> pointer.
 	resource() unsafe.Pointer
 }

@@ -452,6 +328,9 @@ func CopyAllDevices() []Device {
 	return ds
 }

+// Device returns the underlying id<MTLDevice> pointer.
+func (d Device) Device() unsafe.Pointer { return d.device }
+
 // SupportsFeatureSet reports whether device d supports feature set fs.
 //
 // Reference: https://developer.apple.com/documentation/metal/mtldevice/1433418-supportsfeatureset.
@@ -530,6 +409,14 @@ type CompileOptions struct {
 	// TODO.
 }

+// Drawable is a displayable resource that can be rendered or written to.
+//
+// Reference: https://developer.apple.com/documentation/metal/mtldrawable.
+type Drawable interface {
+	// Drawable returns the underlying id<MTLDrawable> pointer.
+	Drawable() unsafe.Pointer
+}
+
 // CommandQueue is a queue that organizes the order
 // in which command buffers are executed by the GPU.
 //
@@ -553,11 +440,11 @@ type CommandBuffer struct {
 	commandBuffer unsafe.Pointer
 }

-// Present registers a drawable presentation to occur as soon as possible.
+// PresentDrawable registers a drawable presentation to occur as soon as possible.
 //
 // Reference: https://developer.apple.com/documentation/metal/mtlcommandbuffer/1443029-presentdrawable.
-func (cb CommandBuffer) Present(d Drawable) {
-	C.CommandBuffer_Present(cb.commandBuffer, d.drawable)
+func (cb CommandBuffer) PresentDrawable(d Drawable) {
+	C.CommandBuffer_PresentDrawable(cb.commandBuffer, d.Drawable())
 }

 // Commit commits this command buffer for execution as soon as possible.
@@ -696,6 +583,8 @@ func (l Library) MakeFunction(name string) (Function, error) {
 type Texture struct {
 	texture unsafe.Pointer

+	// TODO: Change these fields into methods.
+
 	// Width is the width of the texture image for the base level mipmap, in pixels.
 	Width int

@@ -703,6 +592,12 @@ type Texture struct {
 	Height int
 }

+// NewTexture returns a Texture that wraps an existing id<MTLTexture> pointer.
+func NewTexture(texture unsafe.Pointer) Texture {
+	return Texture{texture: texture}
+}
+
+// resource implements the Resource interface.
 func (t Texture) resource() unsafe.Pointer { return t.texture }

 // GetBytes copies a block of pixels from the storage allocation of texture
diff --git a/mtl.h b/mtl.h
index e8924ab..f7c4c67 100644
--- a/mtl.h
+++ b/mtl.h
@@ -74,21 +74,6 @@ struct Region {
 	struct Size   Size;
 };

-void * MakeLayer();
-
-uint16_t     Layer_PixelFormat(void * layer);
-void         Layer_SetDevice(void * layer, void * device);
-const char * Layer_SetPixelFormat(void * layer, uint16_t pixelFormat);
-const char * Layer_SetMaximumDrawableCount(void * layer, uint_t maximumDrawableCount);
-void         Layer_SetDisplaySyncEnabled(void * layer, BOOL displaySyncEnabled);
-void         Layer_SetDrawableSize(void * layer, double width, double height);
-void *       Layer_NextDrawable(void * layer);
-
-void * Drawable_Texture(void * drawable);
-
-void SetWindowContentViewLayer(void * cocoaWindow, void * layer);
-void SetWindowContentViewWantsLayer(void * cocoaWindow, BOOL wantsLayer);
-
 struct Device CreateSystemDefaultDevice();
 struct Devices CopyAllDevices();

@@ -101,7 +86,7 @@ void *                     Device_MakeTexture(void * device, struct TextureDescr

 void * CommandQueue_MakeCommandBuffer(void * commandQueue);

-void   CommandBuffer_Present(void * commandBuffer, void * drawable);
+void   CommandBuffer_PresentDrawable(void * commandBuffer, void * drawable);
 void   CommandBuffer_Commit(void * commandBuffer);
 void   CommandBuffer_WaitUntilCompleted(void * commandBuffer);
 void * CommandBuffer_MakeRenderCommandEncoder(void * commandBuffer, struct RenderPassDescriptor descriptor);
diff --git a/mtl.m b/mtl.m
index 7b0fd96..4296744 100644
--- a/mtl.m
+++ b/mtl.m
@@ -1,71 +1,9 @@
 // +build darwin

 #import <Metal/Metal.h>
-#import <QuartzCore/QuartzCore.h>
-#import <Cocoa/Cocoa.h>
-
 #include <stdlib.h>
-
 #include "mtl.h"

-void * MakeLayer() {
-	return [[CAMetalLayer alloc] init];
-}
-
-uint16_t Layer_PixelFormat(void * layer) {
-	return ((CAMetalLayer *)layer).pixelFormat;
-}
-
-void Layer_SetDevice(void * layer, void * device) {
-	((CAMetalLayer *)layer).device = (id<MTLDevice>)device;
-}
-
-const char * Layer_SetPixelFormat(void * layer, uint16_t pixelFormat) {
-	@try {
-		((CAMetalLayer *)layer).pixelFormat = (MTLPixelFormat)pixelFormat;
-	}
-	@catch (NSException * exception) {
-		return exception.reason.UTF8String;
-	}
-	return NULL;
-}
-
-const char * Layer_SetMaximumDrawableCount(void * layer, uint_t maximumDrawableCount) {
-	if (@available(macOS 10.13.2, *)) {
-		@try {
-			((CAMetalLayer *)layer).maximumDrawableCount = (NSUInteger)maximumDrawableCount;
-		}
-		@catch (NSException * exception) {
-			return exception.reason.UTF8String;
-		}
-	}
-	return NULL;
-}
-
-void Layer_SetDisplaySyncEnabled(void * layer, BOOL displaySyncEnabled) {
-	((CAMetalLayer *)layer).displaySyncEnabled = displaySyncEnabled;
-}
-
-void Layer_SetDrawableSize(void * layer, double width, double height) {
-	((CAMetalLayer *)layer).drawableSize = (CGSize){width, height};
-}
-
-void * Layer_NextDrawable(void * layer) {
-	return [(CAMetalLayer *)layer nextDrawable];
-}
-
-void * Drawable_Texture(void * drawable) {
-	return ((id<CAMetalDrawable>)drawable).texture;
-}
-
-void SetWindowContentViewLayer(void * cocoaWindow, void * layer) {
-	((NSWindow *)cocoaWindow).contentView.layer = (CAMetalLayer *)layer;
-}
-
-void SetWindowContentViewWantsLayer(void * cocoaWindow, BOOL wantsLayer) {
-	((NSWindow *)cocoaWindow).contentView.wantsLayer = wantsLayer;
-}
-
 struct Device CreateSystemDefaultDevice() {
 	id<MTLDevice> device = MTLCreateSystemDefaultDevice();
 	if (!device) {
@@ -162,8 +100,8 @@ struct RenderPipelineState Device_MakeRenderPipelineState(void * device, struct
 	return [(id<MTLCommandQueue>)commandQueue commandBuffer];
 }

-void CommandBuffer_Present(void * commandBuffer, void * drawable) {
-	[(id<MTLCommandBuffer>)commandBuffer presentDrawable:(id<CAMetalDrawable>)drawable];
+void CommandBuffer_PresentDrawable(void * commandBuffer, void * drawable) {
+	[(id<MTLCommandBuffer>)commandBuffer presentDrawable:(id<MTLDrawable>)drawable];
 }

 void CommandBuffer_Commit(void * commandBuffer) {
`

const diffMtlCommit4 = `diff --git a/internal/ca/ca.go b/example/movingtriangle/internal/ca/ca.go
similarity index 98%
rename from internal/ca/ca.go
rename to example/movingtriangle/internal/ca/ca.go
index 87afcc6..d2ff39d 100644
--- a/internal/ca/ca.go
+++ b/example/movingtriangle/internal/ca/ca.go
@@ -4,7 +4,7 @@
 //
 // This package is in very early stages of development.
 // It's a minimal implementation with scope limited to
-// supporting the ../../example/movingtriangle command.
+// supporting the movingtriangle example.
 package ca

 import (
diff --git a/internal/ca/ca.h b/example/movingtriangle/internal/ca/ca.h
similarity index 100%
rename from internal/ca/ca.h
rename to example/movingtriangle/internal/ca/ca.h
diff --git a/internal/ca/ca.m b/example/movingtriangle/internal/ca/ca.m
similarity index 100%
rename from internal/ca/ca.m
rename to example/movingtriangle/internal/ca/ca.m
diff --git a/internal/ns/ns.go b/example/movingtriangle/internal/ns/ns.go
similarity index 87%
rename from internal/ns/ns.go
rename to example/movingtriangle/internal/ns/ns.go
index b81157d..e8d2993 100644
--- a/internal/ns/ns.go
+++ b/example/movingtriangle/internal/ns/ns.go
@@ -1,16 +1,16 @@
 // +build darwin

-// Package ns provides access to Apple's Cocoa API (https://developer.apple.com/documentation/appkit).
+// Package ns provides access to Apple's AppKit API (https://developer.apple.com/documentation/appkit).
 //
 // This package is in very early stages of development.
 // It's a minimal implementation with scope limited to
-// supporting the ../../example/movingtriangle command.
+// supporting the movingtriangle example.
 package ns

 import (
 	"unsafe"

-	"dmitri.shuralyov.com/gpu/mtl/internal/ca"
+	"dmitri.shuralyov.com/gpu/mtl/example/movingtriangle/internal/ca"
 )

 /*
diff --git a/internal/ns/ns.h b/example/movingtriangle/internal/ns/ns.h
similarity index 100%
rename from internal/ns/ns.h
rename to example/movingtriangle/internal/ns/ns.h
diff --git a/internal/ns/ns.m b/example/movingtriangle/internal/ns/ns.m
similarity index 100%
rename from internal/ns/ns.m
rename to example/movingtriangle/internal/ns/ns.m
diff --git a/example/movingtriangle/main.go b/example/movingtriangle/main.go
index b09a63d..cf2aa35 100644
--- a/example/movingtriangle/main.go
+++ b/example/movingtriangle/main.go
@@ -1,6 +1,7 @@
 // +build darwin

 // movingtriangle is an example Metal program that displays a moving triangle in a window.
+// It opens a window and renders a triangle that follows the mouse cursor.
 package main

 import (
@@ -13,8 +14,8 @@ import (
 	"unsafe"

 	"dmitri.shuralyov.com/gpu/mtl"
-	"dmitri.shuralyov.com/gpu/mtl/internal/ca"
-	"dmitri.shuralyov.com/gpu/mtl/internal/ns"
+	"dmitri.shuralyov.com/gpu/mtl/example/movingtriangle/internal/ca"
+	"dmitri.shuralyov.com/gpu/mtl/example/movingtriangle/internal/ns"
 	"github.com/go-gl/glfw/v3.2/glfw"
 	"golang.org/x/image/math/f32"
 )
`
