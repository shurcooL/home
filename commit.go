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
	"path"
	"sort"
	"strings"
	"syscall"
	"time"

	statepkg "dmitri.shuralyov.com/state"
	"github.com/shurcooL/highlight_diff"
	homecomponent "github.com/shurcooL/home/component"
	"github.com/shurcooL/home/internal/code"
	"github.com/shurcooL/home/internal/exp/service/change"
	issues "github.com/shurcooL/home/internal/exp/service/issue"
	"github.com/shurcooL/home/internal/exp/service/notification"
	"github.com/shurcooL/home/internal/route"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/httperror"
	issuescomponent "github.com/shurcooL/issuesapp/component"
	"github.com/shurcooL/octicon"
	"github.com/shurcooL/users"
	"github.com/sourcegraph/annotate"
	"github.com/sourcegraph/go-diff/diff"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// commitHandler is a handler for displaying a commit of a git repository.
type commitHandler struct {
	Repo repoInfo

	issues       issueCounter
	change       changeCounter
	notification notification.Service
	users        users.Service
	gitUsers     map[string]users.User // Key is lower git author email.
}

var commitHTML = template.Must(template.New("").Parse(`<html>
	<head>
{{.AnalyticsHTML}}		<title>{{.FullName}} - Commit {{.Hash}}</title>
		<link href="/icon.svg" rel="icon" type="image/svg+xml">
		<meta name="viewport" content="width=device-width">
		<link href="/assets/fonts/fonts.css" rel="stylesheet" type="text/css">
		<link href="/assets/commit/style.css" rel="stylesheet" type="text/css">
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
		nc, err = h.notification.CountNotifications(req.Context())
		if err != nil {
			return err
		}
	}

	t0 := time.Now()
	openIssues, err := h.issues.Count(req.Context(), issues.RepoSpec{URI: h.Repo.Spec}, issues.IssueListOptions{State: issues.StateFilter(statepkg.IssueOpen)})
	if err != nil {
		return err
	}
	openChanges, err := h.change.Count(req.Context(), h.Repo.Spec, change.ListOptions{Filter: change.FilterOpen})
	if err != nil {
		return err
	}
	fmt.Println("counting open issues & changes took:", time.Since(t0).Nanoseconds(), "for:", h.Repo.Spec)

	commitHash, err := verifyCommitHash(req.URL.Path[1:])
	if err != nil {
		return os.ErrNotExist
	}
	c, err := diffTree(req.Context(), h.Repo.Dir, commitHash, ":", h.gitUsers)
	if err != nil {
		return err
	}
	if commitHash != c.CommitHash {
		return os.ErrNotExist
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = commitHTML.Execute(w, struct {
		AnalyticsHTML template.HTML
		FullName      string
		Hash          string
	}{
		AnalyticsHTML: analyticsHTML,
		FullName:      "Repository " + path.Base(h.Repo.Spec),
		Hash:          shortSHA(c.CommitHash),
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

	err = html.Render(w, htmlg.H2(htmlg.Text(h.Repo.Spec+"/...")))
	if err != nil {
		return err
	}

	// Render the tabnav.
	err = htmlg.RenderComponents(w, homecomponent.RepositoryTabNav(homecomponent.HistoryTab, h.Repo.Path, h.Repo.Packages, openIssues, openChanges))
	if err != nil {
		return err
	}

	err = commitHTML.ExecuteTemplate(w, "CommitMessage", commitMessage{
		ImportPath: h.Repo.Spec,
		CommitHash: c.CommitHash,
		Subject:    c.Subject,
		Body:       c.Body,
		Author:     c.Author,
		AuthorTime: c.AuthorTime,
	})
	if err != nil {
		return err
	}

	if len(c.Patch) == 0 {
		// Empty commit. Let the user know via a blank slate.
		err := htmlg.RenderComponents(w, homecomponent.BlankSlate{
			Content: htmlg.Nodes{htmlg.Text("There are no affected files.")},
		})
		if err != nil {
			return err
		}
	} else {
		fileDiffs, err := diff.ParseMultiFileDiff(c.Patch)
		if err != nil {
			return err
		}
		for _, f := range fileDiffs {
			err := commitHTML.ExecuteTemplate(w, "FileDiff", fileDiff{FileDiff: f})
			if err != nil {
				return err
			}
		}
	}

	_, err = io.WriteString(w, `</div>
	</body>
</html>`)
	return err
}

// commitHandlerPkg is a handler for displaying a commit of a single package.
type commitHandlerPkg struct {
	Repo    repoInfo
	PkgPath string
	Dir     *code.Directory

	notification notification.Service
	users        users.Service
	gitUsers     map[string]users.User // Key is lower git author email.
}

func (h *commitHandlerPkg) ServeHTTP(w http.ResponseWriter, req *http.Request) error {
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
		nc, err = h.notification.CountNotifications(req.Context())
		if err != nil {
			return err
		}
	}

	commitHash, err := verifyCommitHash(req.URL.Path[1:])
	if err != nil {
		return os.ErrNotExist
	}
	c, err := diffTree(req.Context(), h.Repo.Dir, commitHash, directoryGitPathspec(h.Dir), h.gitUsers)
	if err != nil {
		return err
	}
	if commitHash != c.CommitHash {
		return os.ErrNotExist
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var fullName string
	if h.Dir.Package == nil {
		fullName = "Directory " + path.Base(h.Dir.ImportPath)
	} else if h.Dir.Package.IsCommand() {
		fullName = "Command " + path.Base(h.Dir.ImportPath)
	} else {
		fullName = "Package " + h.Dir.Package.Name
	}
	err = commitHTML.Execute(w, struct {
		AnalyticsHTML template.HTML
		FullName      string
		Hash          string
	}{
		AnalyticsHTML: analyticsHTML,
		FullName:      fullName,
		Hash:          shortSHA(c.CommitHash),
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

	err = html.Render(w, htmlg.H2(htmlg.Text(h.Dir.ImportPath)))
	if err != nil {
		return err
	}

	// Render the tabnav.
	err = htmlg.RenderComponents(w, directoryTabnav(homecomponent.HistoryTab, h.PkgPath))
	if err != nil {
		return err
	}

	err = commitHTML.ExecuteTemplate(w, "CommitMessage", commitMessage{
		ImportPath: h.Dir.ImportPath,
		CommitHash: c.CommitHash,
		Subject:    strings.TrimPrefix(c.Subject, pathWithinRepo(h.Dir)+": "), // THINK: Trim package prefix from subject better?
		Body:       c.Body,
		Author:     c.Author,
		AuthorTime: c.AuthorTime,
	})
	if err != nil {
		return err
	}

	// Show warning we're displaying a part of the commit, link to entire commit.
	err = htmlg.RenderComponents(w, homecomponent.Flash{
		Content: htmlg.Nodes{
			htmlg.Text("Showing partial commit. "),
			htmlg.A("Full Commit", route.RepoCommit(h.Repo.Path)+"/"+c.CommitHash),
		},
	})
	if err != nil {
		return err
	}

	if len(c.Patch) == 0 {
		// Empty commit. Let the user know via a blank slate.
		err := htmlg.RenderComponents(w, homecomponent.BlankSlate{
			Content: htmlg.Nodes{htmlg.Text("There are no affected files.")},
		})
		if err != nil {
			return err
		}
	} else {
		fileDiffs, err := diff.ParseMultiFileDiff(c.Patch)
		if err != nil {
			return err
		}
		for _, f := range fileDiffs {
			// THINK: Trim package prefix from file paths better?
			f.OrigName = strings.TrimPrefix(f.OrigName, pathWithinRepo(h.Dir)+"/")
			f.NewName = strings.TrimPrefix(f.NewName, pathWithinRepo(h.Dir)+"/")
			err := commitHTML.ExecuteTemplate(w, "FileDiff", fileDiff{FileDiff: f})
			if err != nil {
				return err
			}
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

func shortSHA(sha string) string {
	return sha[:8]
}

func diffTree(ctx context.Context, gitDir, treeish, pathspec string, gitUsers map[string]users.User) (diffTreeResponse, error) {
	cmd := exec.CommandContext(ctx, "git", "diff-tree",
		"--unified=5",
		"--format=tformat:%H%x00%s%x00%b%x00%an%x00%ae%x00%aI",
		"-z",
		"--no-prefix",
		"--always",
		"--root",
		"--find-renames",
		//"--break-rewrites",
		treeish, "--", pathspec)
	cmd.Dir = gitDir
	var buf bytes.Buffer
	cmd.Stdout = &buf
	err := cmd.Start()
	if os.IsNotExist(err) {
		// TODO: Document when this happens. Is it when gitDir doesn't exist, or git doesn't exist, or?
		return diffTreeResponse{}, err
	} else if err != nil {
		return diffTreeResponse{}, fmt.Errorf("could not start command: %v", err)
	}
	err = cmd.Wait()
	if ee, _ := err.(*exec.ExitError); ee != nil && ee.Sys().(syscall.WaitStatus).ExitStatus() == 128 {
		return diffTreeResponse{}, os.ErrNotExist // Commit doesn't exist.
	} else if err != nil {
		return diffTreeResponse{}, fmt.Errorf("%v: %v", cmd.Args, err)
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
		patch       = b // There may be a leading '\n', but diff.ParseMultiFileDiff ignores it anyway, so leave it. It's not there when commit is empty.
	)
	author, ok := gitUsers[strings.ToLower(authorEmail)]
	if !ok {
		author = users.User{
			Name:      authorName,
			Email:     authorEmail,
			AvatarURL: "https://secure.gravatar.com/avatar?d=mm&f=y&s=96", // TODO: Use email.
		}
	}
	authorTime, err := time.Parse(time.RFC3339, authorDate)
	if err != nil {
		return diffTreeResponse{}, err
	}
	return diffTreeResponse{
		CommitHash: commitHash,
		Subject:    subject,
		Body:       body,
		Author:     author,
		AuthorTime: authorTime,
		Patch:      patch,
	}, nil
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
	ImportPath string // Import path corresponding to directory used for linking to source code view.
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
			{Key: atom.Href.String(), Val: "https://gotools.org/" + c.ImportPath + "?rev=" + c.CommitHash},
			{Key: atom.Title.String(), Val: "View code at this revision."},
		},
		FirstChild: octicon.Code(),
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
	html, err := highlightDiff(hunks)
	if err != nil {
		log.Println("fileDiff.Diff: highlightDiff:", err)
		var buf bytes.Buffer
		template.HTMLEscape(&buf, hunks)
		html = buf.Bytes()
	}
	return template.HTML(html), nil
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
