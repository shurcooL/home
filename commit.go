package main

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/shurcooL/highlight_diff"
	homecomponent "github.com/shurcooL/home/component"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/httperror"
	issuescomponent "github.com/shurcooL/issuesapp/component"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/octiconssvg"
	"github.com/shurcooL/users"
	"github.com/sourcegraph/annotate"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"sourcegraph.com/sourcegraph/go-diff/diff"
)

// commitHandler is a handler for displaying a commit of a git repository.
type commitHandler struct {
	Repo          string // Repo URI. E.g., "example.com/some/package".
	Path          string // Path corresponding to repo root, without domain. E.g., "/some/package".
	Name          string // Package name. E.g., "package".
	RepoDir       string // Path to repository directory on disk.
	notifications notifications.Service
	users         users.Service
	gitUsers      map[string]users.User // Key is lower git author email.
}

var commitHTML = template.Must(template.New("").Parse(`<html>
	<head>
		<title>Package {{.Name}} - Commit</title>
		<link href="/icon.png" rel="icon" type="image/png">
		<meta name="viewport" content="width=device-width">
		<link href="/assets/fonts/fonts.css" rel="stylesheet" type="text/css">
		<link href="/assets/commit/style.css" rel="stylesheet" type="text/css">
		{{if .Production}}` + googleAnalytics + `{{end}}
	</head>
	<body>

{{define "CommitMessage"}}
<div class="list-entry list-entry-border commit-message">
	<div class="list-entry-header">
		<div style="display: flex;">
			<pre style="flex-grow: 1;"><strong>{{.Subject}}</strong>{{with .Body}}

{{.}}{{end}}</pre>
			{{.ViewCode}}
		</div>
	</div>
	<div class="list-entry-body">
		<span style="display: inline-block; vertical-align: bottom; margin-right: 5px;">{{.Avatar}}</span>{{/*
		*/}}<span style="display: inline-block;">{{.User}} committed {{.Time}}</span>
		<span style="float: right;">
			<span>commit <code>{{.CommitHash}}</code></span>
		</span>
	</div>
</div>
{{end}}

{{define "FileDiff"}}
<div class="list-entry list-entry-border">
	<div class="list-entry-header">{{.Title}}</div>
	<div class="list-entry-body">
		<pre class="highlight">{{.Diff}}</pre>
	</div>
</div>
{{end}}
`))

func (h *commitHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		return httperror.Method{Allowed: []string{"GET"}}
	}

	authenticatedUser, err := h.users.GetAuthenticated(req.Context())
	if err != nil {
		log.Println(err)
		authenticatedUser = users.User{} // THINK: Should it be a fatal error or not? What about on frontend vs backend?
	}
	var nc uint64
	if authenticatedUser.ID != 0 {
		nc, err = h.notifications.Count(req.Context(), nil)
		if err != nil {
			return err
		}
	}

	commitHash, err := verifyCommitHash(req.URL.Path[1:])
	if err != nil {
		return os.ErrNotExist
	}
	c, err := diffTree(req.Context(), h.RepoDir, commitHash, h.gitUsers)
	if err != nil {
		return err
	}
	if commitHash != c.CommitHash {
		return os.ErrNotExist
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = commitHTML.Execute(w, struct {
		Production bool
		Name       string
	}{
		Production: *productionFlag,
		Name:       h.Name,
	})
	if err != nil {
		return err
	}

	_, err = io.WriteString(w, `<div style="max-width: 800px; margin: 0 auto 100px auto;">`)
	if err != nil {
		return err
	}

	// Render the header.
	header := homecomponent.Header{
		CurrentUser:       authenticatedUser,
		NotificationCount: nc,
		ReturnURL:         req.RequestURI,
	}
	err = htmlg.RenderComponents(w, header)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, `<h2>Package %s</h2>`, h.Name)
	if err != nil {
		return err
	}

	// Render the tabnav.
	err = htmlg.RenderComponents(w, tabnav{
		Tabs: []tab{
			{
				Content: iconText{Icon: octiconssvg.Book, Text: "Overview"},
				URL:     h.Path,
			},
			{
				Content:  iconText{Icon: octiconssvg.History, Text: "History"},
				URL:      h.Path + "/commits",
				Selected: true,
			},
			{
				Content: iconText{Icon: octiconssvg.IssueOpened, Text: "Issues"},
				URL:     h.Path + "/issues",
			},
		},
	})
	if err != nil {
		return err
	}

	err = commitHTML.ExecuteTemplate(w, "CommitMessage", commitMessage{
		RepoURL:    h.Repo,
		CommitHash: c.CommitHash,
		Subject:    c.Subject,
		Body:       c.Body,
		Author:     c.Author,
		AuthorTime: c.AuthorTime,
	})
	if err != nil {
		return err
	}

	fileDiffs, err := diff.ParseMultiFileDiff(c.Patch)
	if err != nil {
		return err
	}
	for _, f := range fileDiffs {
		err = commitHTML.ExecuteTemplate(w, "FileDiff", fileDiff{FileDiff: f})
		if err != nil {
			return err
		}
	}

	_, err = io.WriteString(w, `</div>
	</body>
</html>`)
	return err
}

func verifyCommitHash(commitHash string) (string, error) {
	if len(commitHash) != 40 {
		return "", fmt.Errorf("length is %v instead of 40", len(commitHash))
	}
	for _, b := range []byte(commitHash) {
		ok := ('0' <= b && b <= '9') || ('a' <= b && b <= 'f')
		if !ok {
			return "", fmt.Errorf("commit hash contains unexpected character %+q", b)
		}
	}
	return commitHash, nil
}

func diffTree(ctx context.Context, repoDir, treeish string, gitUsers map[string]users.User) (diffTreeResponse, error) {
	cmd := exec.CommandContext(ctx, "git", "diff-tree",
		"--unified=5",
		"--format=tformat:%H%x00%s%x00%b%x00%an%x00%ae%x00%aI",
		"-z",
		"--no-prefix",
		"--always",
		"--root",
		"--find-renames",
		//"--break-rewrites",
		treeish)
	cmd.Dir = repoDir
	var buf bytes.Buffer
	cmd.Stdout = &buf
	err := cmd.Start()
	if os.IsNotExist(err) {
		return diffTreeResponse{}, err
	} else if err != nil {
		return diffTreeResponse{}, fmt.Errorf("could not start command: %v", err)
	}
	err = cmd.Wait()
	if ee, _ := err.(*exec.ExitError); ee != nil && ee.Sys().(syscall.WaitStatus).ExitStatus() == 128 {
		return diffTreeResponse{}, os.ErrNotExist // Commit doesn't exist.
	} else if err != nil {
		return diffTreeResponse{}, err
	}
	b := buf.Bytes()
	var (
		// Calls to readLine match exactly what is specified in --format.
		commitHash  = readLine(&b)
		subject     = readLine(&b)
		body        = readLine(&b)
		authorName  = readLine(&b)
		authorEmail = readLine(&b)
		authorDate  = readLine(&b)
		patch       = b[1:]
	)

	c := diffTreeResponse{
		CommitHash: commitHash,
		Subject:    subject,
		Body:       body,
	}

	var ok bool
	c.Author, ok = gitUsers[strings.ToLower(authorEmail)]
	if !ok {
		c.Author = users.User{
			Name:      authorName,
			Email:     authorEmail,
			AvatarURL: "https://secure.gravatar.com/avatar?d=mm&f=y&s=96", // TODO: Use email.
		}
	}

	c.AuthorTime, err = time.Parse(time.RFC3339, authorDate)
	if err != nil {
		return diffTreeResponse{}, err
	}

	c.Patch = patch
	return c, nil
}

type diffTreeResponse struct {
	CommitHash string
	Subject    string
	Body       string
	Author     users.User
	AuthorTime time.Time
	Patch      []byte
}

// readLine reads a line until zero byte, then updates b to the byte that immediately follows.
// A zero byte must exist in b, otherwise readLine panics.
func readLine(b *[]byte) string {
	i := bytes.IndexByte(*b, 0)
	s := string((*b)[:i])
	*b = (*b)[i+1:]
	return s
}

type commitMessage struct {
	RepoURL    string // TODO: This is more of import path rather than repo; it should change for subpackages.
	CommitHash string
	Subject    string
	Body       string
	Author     users.User
	AuthorTime time.Time
}

func (c commitMessage) ViewCode() template.HTML {
	return template.HTML(htmlg.Render(&html.Node{
		Type: html.ElementNode, Data: atom.A.String(),
		Attr: []html.Attribute{
			{Key: atom.Class.String(), Val: "lightgray"},
			{Key: atom.Style.String(), Val: "height: 16px;"},
			{Key: atom.Href.String(), Val: "https://gotools.org/" + c.RepoURL + "?rev=" + c.CommitHash},
			{Key: atom.Title.String(), Val: "View code at this revision."},
		},
		FirstChild: octiconssvg.Code(),
	}))
}

func (c commitMessage) Avatar() template.HTML {
	return template.HTML(htmlg.RenderComponentsString(issuescomponent.Avatar{User: c.Author, Size: 24}))
}

func (c commitMessage) User() template.HTML {
	return template.HTML(htmlg.RenderComponentsString(issuescomponent.User{User: c.Author}))
}

func (c commitMessage) Time() template.HTML {
	return template.HTML(htmlg.RenderComponentsString(issuescomponent.Time{Time: c.AuthorTime}))
}

type fileDiff struct {
	*diff.FileDiff
}

func (f fileDiff) Title() (template.HTML, error) {
	switch new, old := f.NewName, f.OrigName; {
	case old != "/dev/null" && new != "/dev/null" && old == new: // Modified.
		return template.HTML(html.EscapeString(new)), nil
	case old != "/dev/null" && new != "/dev/null" && old != new: // Renamed.
		return template.HTML(html.EscapeString(old + " -> " + new)), nil
	case old == "/dev/null" && new != "/dev/null": // Added.
		return template.HTML(html.EscapeString(new)), nil
	case old != "/dev/null" && new == "/dev/null": // Removed.
		return template.HTML("<strikethrough>" + html.EscapeString(old) + "</strikethrough>"), nil
	default:
		return "", fmt.Errorf("unexpected *diff.FileDiff: %+v", f)
	}
}

func (f fileDiff) Diff() (template.HTML, error) {
	hunks, err := diff.PrintHunks(f.Hunks)
	if err != nil {
		return "", err
	}
	diff, err := highlightDiff(hunks)
	if err != nil {
		return "", err
	}
	return template.HTML(diff), nil
}

// highlightDiff highlights the src diff, returning the annotated HTML.
func highlightDiff(src []byte) ([]byte, error) {
	anns, err := highlight_diff.Annotate(src)
	if err != nil {
		return nil, err
	}

	lines := bytes.Split(src, []byte("\n"))
	lineStarts := make([]int, len(lines))
	var offset int
	for lineIndex := 0; lineIndex < len(lines); lineIndex++ {
		lineStarts[lineIndex] = offset
		offset += len(lines[lineIndex]) + 1
	}

	lastDel, lastIns := -1, -1
	for lineIndex := 0; lineIndex < len(lines); lineIndex++ {
		var lineFirstChar byte
		if len(lines[lineIndex]) > 0 {
			lineFirstChar = lines[lineIndex][0]
		}
		switch lineFirstChar {
		case '+':
			if lastIns == -1 {
				lastIns = lineIndex
			}
		case '-':
			if lastDel == -1 {
				lastDel = lineIndex
			}
		default:
			if lastDel != -1 || lastIns != -1 {
				if lastDel == -1 {
					lastDel = lastIns
				} else if lastIns == -1 {
					lastIns = lineIndex
				}

				beginOffsetLeft := lineStarts[lastDel]
				endOffsetLeft := lineStarts[lastIns]
				beginOffsetRight := lineStarts[lastIns]
				endOffsetRight := lineStarts[lineIndex]

				anns = append(anns, &annotate.Annotation{Start: beginOffsetLeft, End: endOffsetLeft, Left: []byte(`<span class="gd input-block">`), Right: []byte(`</span>`), WantInner: 0})
				anns = append(anns, &annotate.Annotation{Start: beginOffsetRight, End: endOffsetRight, Left: []byte(`<span class="gi input-block">`), Right: []byte(`</span>`), WantInner: 0})

				if '@' != lineFirstChar {
					//leftContent := string(src[beginOffsetLeft:endOffsetLeft])
					//rightContent := string(src[beginOffsetRight:endOffsetRight])
					// This is needed to filter out the "-" and "+" at the beginning of each line from being highlighted.
					// TODO: Still not completely filtered out.
					leftContent := ""
					for line := lastDel; line < lastIns; line++ {
						leftContent += "\x00" + string(lines[line][1:]) + "\n"
					}
					rightContent := ""
					for line := lastIns; line < lineIndex; line++ {
						rightContent += "\x00" + string(lines[line][1:]) + "\n"
					}

					var sectionSegments [2][]*annotate.Annotation
					highlight_diff.HighlightedDiffFunc(leftContent, rightContent, &sectionSegments, [2]int{beginOffsetLeft, beginOffsetRight})

					anns = append(anns, sectionSegments[0]...)
					anns = append(anns, sectionSegments[1]...)
				}
			}
			lastDel, lastIns = -1, -1
		}
	}

	sort.Sort(anns)

	out, err := annotate.Annotate(src, anns, template.HTMLEscape)
	if err != nil {
		return nil, err
	}

	return out, nil
}
