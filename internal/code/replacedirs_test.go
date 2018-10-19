package code

import (
	"reflect"
	"testing"
)

func TestReplaceDirs(t *testing.T) {
	tests := []struct {
		s           []*Directory
		repoRoot    string
		dirs        []*Directory
		want        []*Directory
		wantOldDirs []*Directory
	}{
		// 2 -> 1 (shrink by 1).
		{
			s: []*Directory{
				{RepoRoot: "a", ImportPath: "a"},
				{RepoRoot: "b", ImportPath: "b/1"},
				{RepoRoot: "b", ImportPath: "b/2"},
				{RepoRoot: "c", ImportPath: "c"},
			},
			repoRoot: "b",
			dirs: []*Directory{
				{RepoRoot: "b", ImportPath: "b/3"},
			},
			want: []*Directory{
				{RepoRoot: "a", ImportPath: "a"},
				{RepoRoot: "b", ImportPath: "b/3"},
				{RepoRoot: "c", ImportPath: "c"},
			},
			wantOldDirs: []*Directory{
				{RepoRoot: "b", ImportPath: "b/1"},
				{RepoRoot: "b", ImportPath: "b/2"},
			},
		},
		// 2 -> 2 (stay same size).
		{
			s: []*Directory{
				{RepoRoot: "a", ImportPath: "a"},
				{RepoRoot: "b", ImportPath: "b/1"},
				{RepoRoot: "b", ImportPath: "b/2"},
				{RepoRoot: "c", ImportPath: "c"},
			},
			repoRoot: "b",
			dirs: []*Directory{
				{RepoRoot: "b", ImportPath: "b/3"},
				{RepoRoot: "b", ImportPath: "b/4"},
			},
			want: []*Directory{
				{RepoRoot: "a", ImportPath: "a"},
				{RepoRoot: "b", ImportPath: "b/3"},
				{RepoRoot: "b", ImportPath: "b/4"},
				{RepoRoot: "c", ImportPath: "c"},
			},
			wantOldDirs: []*Directory{
				{RepoRoot: "b", ImportPath: "b/1"},
				{RepoRoot: "b", ImportPath: "b/2"},
			},
		},
		// 2 -> 3 (grow by 1).
		{
			s: []*Directory{
				{RepoRoot: "a", ImportPath: "a"},
				{RepoRoot: "b", ImportPath: "b/1"},
				{RepoRoot: "b", ImportPath: "b/2"},
				{RepoRoot: "c", ImportPath: "c"},
			},
			repoRoot: "b",
			dirs: []*Directory{
				{RepoRoot: "b", ImportPath: "b/3"},
				{RepoRoot: "b", ImportPath: "b/4"},
				{RepoRoot: "b", ImportPath: "b/5"},
			},
			want: []*Directory{
				{RepoRoot: "a", ImportPath: "a"},
				{RepoRoot: "b", ImportPath: "b/3"},
				{RepoRoot: "b", ImportPath: "b/4"},
				{RepoRoot: "b", ImportPath: "b/5"},
				{RepoRoot: "c", ImportPath: "c"},
			},
			wantOldDirs: []*Directory{
				{RepoRoot: "b", ImportPath: "b/1"},
				{RepoRoot: "b", ImportPath: "b/2"},
			},
		},

		// 2 -> 0.
		{
			s: []*Directory{
				{RepoRoot: "a", ImportPath: "a"},
				{RepoRoot: "b", ImportPath: "b/1"},
				{RepoRoot: "b", ImportPath: "b/2"},
				{RepoRoot: "c", ImportPath: "c"},
			},
			repoRoot: "b",
			dirs:     nil,
			want: []*Directory{
				{RepoRoot: "a", ImportPath: "a"},
				{RepoRoot: "c", ImportPath: "c"},
			},
			wantOldDirs: []*Directory{
				{RepoRoot: "b", ImportPath: "b/1"},
				{RepoRoot: "b", ImportPath: "b/2"},
			},
		},
		// 0 -> 2.
		{
			s: []*Directory{
				{RepoRoot: "a", ImportPath: "a"},
				{RepoRoot: "c", ImportPath: "c"},
			},
			repoRoot: "b",
			dirs: []*Directory{
				{RepoRoot: "b", ImportPath: "b/1"},
				{RepoRoot: "b", ImportPath: "b/2"},
			},
			want: []*Directory{
				{RepoRoot: "a", ImportPath: "a"},
				{RepoRoot: "b", ImportPath: "b/1"},
				{RepoRoot: "b", ImportPath: "b/2"},
				{RepoRoot: "c", ImportPath: "c"},
			},
			wantOldDirs: []*Directory{},
		},
		// 0 -> 0.
		{
			s: []*Directory{
				{RepoRoot: "a", ImportPath: "a"},
				{RepoRoot: "c", ImportPath: "c"},
			},
			repoRoot: "b",
			dirs:     nil,
			want: []*Directory{
				{RepoRoot: "a", ImportPath: "a"},
				{RepoRoot: "c", ImportPath: "c"},
			},
			wantOldDirs: []*Directory{},
		},
	}
	for _, tc := range tests {
		got := tc.s
		gotOldDirs := replaceDirs(&got, tc.repoRoot, tc.dirs)
		if !reflect.DeepEqual(got, tc.want) {
			t.Error("not equal")
		}
		if !reflect.DeepEqual(gotOldDirs, tc.wantOldDirs) {
			t.Error("oldDirs not equal")
		}
	}
}
