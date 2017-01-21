package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	pathpkg "path"
	"sort"
	"strings"

	"github.com/shurcooL/home/component"
	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/home/presentdata"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/httpfs/html/vfstemplate"
	"github.com/shurcooL/httpfs/vfsutil"
	"github.com/shurcooL/httpgzip"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/users"
	"golang.org/x/net/html"
	"golang.org/x/tools/present"
)

// initTalks registers a talks handler with root as talks content source.
func initTalks(root http.FileSystem, notifications notifications.Service, users users.Service) {
	// Host static files that slides need.
	http.Handle("/static/", userMiddleware{httpgzip.FileServer(presentdata.Assets, httpgzip.FileServerOptions{ServeError: detailedForAdmin{Users: users}.ServeError})})

	// Create a template for slides.
	tmpl := present.Template()
	tmpl = tmpl.Funcs(template.FuncMap{"playable": func(present.Code) bool { return false }})
	tmpl = template.Must(vfstemplate.ParseFiles(presentdata.Assets, tmpl, "/templates/action.tmpl", "/templates/slides.tmpl"))

	talksHandler := http.StripPrefix("/talks", userMiddleware{httputil.ErrorHandler(users, (&talksHandler{
		base:   "/talks",
		fs:     root,
		slides: tmpl,

		notifications: notifications,
		users:         users,
	}).ServeHTTP)})
	http.Handle("/talks", talksHandler)
	http.Handle("/talks/", talksHandler)
}

type talksHandler struct {
	base   string // Base URL to prepend to links.
	fs     http.FileSystem
	slides *template.Template

	notifications notifications.Service
	users         users.Service
}

func (h *talksHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		return httputil.MethodError{Allowed: []string{"GET"}}
	}
	if canonicalURL := pathpkg.Clean(req.RequestURI); canonicalURL != req.RequestURI {
		if req.URL.RawQuery != "" {
			canonicalURL += "?" + req.URL.RawQuery
		}
		return httputil.Redirect{URL: canonicalURL}
	}

	path := pathpkg.Clean("/" + req.URL.Path)

	f, err := h.fs.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return err
	}

	switch dir, ext := fi.IsDir(), pathpkg.Ext(fi.Name()); {
	// Serve a .slide presentation.
	case !dir && ext == ".slide":
		pctx := present.Context{
			ReadFile: func(path string) ([]byte, error) { return vfsutil.ReadFile(h.fs, path) },
		}
		doc, err := pctx.Parse(f, path, 0)
		if err != nil {
			return err
		}
		return doc.Render(w, h.slides)

	// Serve regular files (assets).
	case !dir && ext != ".slide":
		httpgzip.ServeContent(w, req, path, fi.ModTime(), f)
		return nil

	// Serve a directory listing.
	case dir:
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := struct{ Production bool }{*productionFlag}
		err := talksDirHTML.Execute(w, data)
		if err != nil {
			return err
		}

		io.WriteString(w, `<div style="max-width: 800px; margin: 0 auto 100px auto;">`)

		authenticatedUser, err := h.users.GetAuthenticated(req.Context())
		if err != nil {
			log.Println(err)
			authenticatedUser = users.User{} // THINK: Should it be a fatal error or not? What about on frontend vs backend?
		}
		returnURL := req.RequestURI

		// Render the header.
		header := component.Header{
			CurrentUser:   authenticatedUser,
			ReturnURL:     returnURL,
			Notifications: h.notifications,
		}
		err = htmlg.RenderComponentsContext(req.Context(), w, header)
		if err != nil {
			return err
		}

		// Render the directory listing.
		h.renderDir(w, h.fs, path, f)

		io.WriteString(w, `</div>`)

		_, err = io.WriteString(w, `</body></html>`)
		return err

	default:
		panic("unreachable")
	}
}

// renderDir renders the directory listing of d to w.
func (h *talksHandler) renderDir(w io.Writer, fs http.FileSystem, path string, d dirReader) error {
	fis, err := d.Readdir(0)
	if err != nil {
		return err
	}

	dl := &dirList{Base: h.base, Path: path}
	for _, fi := range fis {
		switch dir, ext := fi.IsDir(), pathpkg.Ext(fi.Name()); {
		// Add .slide files to Slides.
		case !dir && ext == ".slide":
			title, err := parseTitle(fs, pathpkg.Join(path, fi.Name()))
			if err != nil {
				log.Println(err)
				title = ""
			}
			dl.Slides = append(dl.Slides, dirEntry{
				Name:  fi.Name(),
				Path:  pathpkg.Join(path, fi.Name()),
				Title: title,
			})

		// Add .pdf files to Files.
		case !dir && ext == ".pdf":
			dl.Files = append(dl.Files, dirEntry{
				Name: fi.Name(),
				Path: pathpkg.Join(path, fi.Name()),
			})

		// Add directories to Dirs.
		case dir && !strings.HasPrefix(fi.Name(), "."):
			dl.Dirs = append(dl.Dirs, dirEntry{
				Name: fi.Name(),
				Path: pathpkg.Join(path, fi.Name()),
			})
		}
	}
	sort.Sort(dl.Slides)
	sort.Sort(dl.Dirs)

	_, err = io.WriteString(w, string(htmlg.Render(dl.Render()...)))
	return err
}

// dirList is a directory listing of slides and directories.
type dirList struct {
	Base                string // Base URL to prepend to links.
	Path                string
	Slides, Files, Dirs dirEntries
}

// Render renders the directory listing as HTML.
func (dl *dirList) Render() []*html.Node {
	var nodes []*html.Node

	nodes = append(nodes,
		htmlg.H1(htmlg.Text("Talks")),
	)

	nodes = append(nodes,
		htmlg.H2(htmlg.Text(dl.Path)),
	)

	if len(dl.Slides) > 0 {
		nodes = append(nodes,
			htmlg.H4(htmlg.Text("Slide decks:")),
		)
		var ns []*html.Node
		for _, s := range dl.Slides {
			ns = append(ns,
				htmlg.DD(
					htmlg.A(s.Name, template.URL(pathpkg.Join(dl.Base, s.Path))), htmlg.Text(": "+s.Title),
				),
			)
		}
		nodes = append(nodes, htmlg.DL(ns...))
	}

	if len(dl.Files) > 0 {
		nodes = append(nodes,
			htmlg.H4(htmlg.Text("Files:")),
		)
		var ns []*html.Node
		for _, s := range dl.Files {
			ns = append(ns,
				htmlg.DD(
					htmlg.A(s.Name, template.URL(pathpkg.Join(dl.Base, s.Path))),
				),
			)
		}
		nodes = append(nodes, htmlg.DL(ns...))
	}

	if len(dl.Dirs) > 0 && len(dl.Slides) == 0 {
		nodes = append(nodes,
			htmlg.H4(htmlg.Text("Sub-directories:")),
		)
		var ns []*html.Node
		for _, d := range dl.Dirs {
			ns = append(ns,
				htmlg.DD(
					htmlg.A(d.Name, template.URL(pathpkg.Join(dl.Base, d.Path))),
				),
			)
		}
		nodes = append(nodes, htmlg.DL(ns...))
	}

	return nodes
}

// dirEntry is an entry within a directory.
type dirEntry struct {
	Name, Path, Title string
}

type dirEntries []dirEntry

func (s dirEntries) Len() int           { return len(s) }
func (s dirEntries) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s dirEntries) Less(i, j int) bool { return s[i].Name < s[j].Name }

// parseTitle parses the title of .slide presentation at path.
func parseTitle(fs http.FileSystem, path string) (string, error) {
	f, err := fs.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	doc, err := titlesContext.Parse(f, path, present.TitlesOnly)
	if err != nil {
		return "", err
	}
	return doc.Title, nil
}

// titlesContext is used for parsing titles only.
var titlesContext = present.Context{
	// ReadFile should not be needed to parse titles.
	ReadFile: func(path string) ([]byte, error) { return nil, fmt.Errorf("implementation not provided") },
}

var talksDirHTML = template.Must(template.New("").Parse(`<html>
	<head>
		<title>Dmitri Shuralyov - Talks</title>
		<link href="/icon.png" rel="icon" type="image/png">
		<style type="text/css">
			body {
				margin: 20px;
				font-family: Helvetica, Arial, sans-serif;
				font-size: 16px;
			}
			a,
			.exampleHeading .text {
				color: #375EAB;
				text-decoration: none;
			}
			a:hover,
			.exampleHeading .text:hover {
				text-decoration: underline;
			}
			h1,
			h2,
			h3,
			h4,
			.rootHeading {
				margin: 20px 0;
				padding: 0;
				color: rgb(35, 35, 35);
				font-weight: bold;
			}
			h1 {
				font-size: 24px;
			}
			h2 {
				font-size: 20px;
				background: #E0EBF5;
				padding: 2px 5px;
			}
			h3 {
				font-size: 20px;
			}
			h3,
			h4 {
				margin: 20px 5px;
			}
			h4 {
				font-size: 16px;
			}

			dl {
				margin: 20px;
			}
			dd {
				margin: 2px 20px;
			}
			dl,
			dd {
				font-size: 14px;
			}
		</style>
		{{if .Production}}` + googleAnalytics + `{{end}}
	</head>
	<body>`))

type dirReader interface {
	Readdir(count int) ([]os.FileInfo, error)
}
