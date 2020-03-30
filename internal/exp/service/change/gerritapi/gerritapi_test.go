package gerritapi

import (
	"testing"
)

func TestParseMessage(t *testing.T) {
	tests := []struct {
		in         string
		wantLabels string
		wantBody   string
	}{
		{
			in:         "Patch Set 2: Code-Review+2",
			wantLabels: "Code-Review+2",
			wantBody:   "",
		},
		{
			in:         "Patch Set 3: Run-TryBot+1 Code-Review+2",
			wantLabels: "Run-TryBot+1 Code-Review+2",
			wantBody:   "",
		},
		{
			in:         "Patch Set 2: Code-Review+2\n\nThanks.",
			wantLabels: "Code-Review+2",
			wantBody:   "Thanks.",
		},
		{
			in:         "Patch Set 1:\n\nFirst contribution — trying to get my feet wet. Please review.",
			wantLabels: "",
			wantBody:   "First contribution — trying to get my feet wet. Please review.",
		},
	}
	for i, tc := range tests {
		gotLabels, gotBody, ok := parseMessage(tc.in)
		if !ok {
			t.Fatalf("%d: not ok", i)
		}
		if gotLabels != tc.wantLabels {
			t.Errorf("%d: got labels: %q, want: %q", i, gotLabels, tc.wantLabels)
		}
		if gotBody != tc.wantBody {
			t.Errorf("%d: got body: %q, want: %q", i, gotBody, tc.wantBody)
		}
	}
}

func TestParsePSMessage(t *testing.T) {
	tests := []struct {
		inMessage        string
		inRevisionNumber int
		wantBody         string
		wantError        bool
	}{
		{
			inMessage:        "Uploaded patch set 1.",
			inRevisionNumber: 1,
			wantBody:         "",
		},
		{
			inMessage:        "Uploaded patch set 2.\n\n(3 comments)",
			inRevisionNumber: 2,
			wantBody:         "(3 comments)",
		},
		{
			inMessage:        "Patch Set 3: Commit message was updated.",
			inRevisionNumber: 3,
			wantBody:         "",
		},
		{
			inMessage:        "Uploaded patch set 4: Run-TryBot+1.\n\n(1 comment)",
			inRevisionNumber: 4,
			wantBody:         "(1 comment)",
		},
		{
			inMessage:        "Uploaded patch set 5: Run-TryBot+1.",
			inRevisionNumber: 5,
			wantBody:         "",
		},
		{
			inMessage:        "Uploaded patch set 6.\nThis Gerrit CL corresponds to GitHub PR golang/tools#123.\n\nAuthor: Foo Bar \u003cfoo@bar.com\u003e",
			inRevisionNumber: 6,
			wantBody:         "This Gerrit CL corresponds to GitHub PR golang/tools#123.\n\nAuthor: Foo Bar \u003cfoo@bar.com\u003e",
		},
		{
			inMessage:        "something unexpected",
			inRevisionNumber: 3,
			wantError:        true,
		},
	}
	for i, tc := range tests {
		body, err := parsePSMessage(tc.inMessage, tc.inRevisionNumber)
		if got, want := err != nil, tc.wantError; got != want {
			t.Errorf("%d: got error: %v, want: %v", i, got, want)
			continue
		}
		if tc.wantError {
			continue
		}
		if got, want := body, tc.wantBody; got != want {
			t.Errorf("%d: got body: %q, want: %q", i, got, want)
		}
	}
}

func TestCommitMessageBody(t *testing.T) {
	for i, tc := range []struct {
		in   string
		want string
	}{
		{
			in: `cmd/gopherbot: assign reviewers based on commit message prefixes

Previously, we assigned reviewers based on the file paths involved.

In practice, many changes focused in one directory have trivial
repercussions elsewhere, so that heuristic tends to involve too many
reviewers.

I added a path expansion function in CL 170863, so let's use that here
too: a human selected the paths for the commit message, so use that
human's choice to guide reviewer selection.

Fixes golang/go#30695

Change-Id: If28d6cb2511f4e3f3c651bf736dda394e098c17d
`,
			want: `Previously, we assigned reviewers based on the file paths involved.

In practice, many changes focused in one directory have trivial
repercussions elsewhere, so that heuristic tends to involve too many
reviewers.

I added a path expansion function in CL 170863, so let's use that here
too: a human selected the paths for the commit message, so use that
human's choice to guide reviewer selection.

Fixes golang/go#30695`,
		},
		{
			in: `transform, unicode/cldr: spell "Deprecated: Use etc" consistently

Change-Id: I5194e58a7679e33555856a413f6081fea26d8e34
`,
			want: "",
		},
	} {
		got := commitMessageBody(tc.in)
		if got != tc.want {
			t.Errorf("%d: got: %q, want: %q", i, got, tc.want)
		}
	}
}
