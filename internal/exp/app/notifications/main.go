// Package notifications app is a web frontend for a notification service.
package notifications

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/home/internal/exp/app/notifications/assets"
	"github.com/shurcooL/home/internal/exp/service/notification"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/httpgzip"
	"github.com/shurcooL/users"
)

// New returns a notifications app http.Handler using given services and options.
// It uses users service, if not nil, when displaying errors (admins see full details).
//
// An HTTP notificationv2 API must be available at /api/notificationv2:
//
// 	// Register HTTP API endpoints.
// 	apiHandler := httphandler.Notification{Notification: service}
// 	mux.Handle(path.Join("/api/notificationv2", httproute.ListNotifications), errorHandler(apiHandler.ListNotifications))
// 	mux.Handle(path.Join("/api/notificationv2", httproute.StreamNotifications), errorHandler(apiHandler.StreamNotifications))
// 	mux.Handle(path.Join("/api/notificationv2", httproute.CountNotifications), errorHandler(apiHandler.CountNotifications))
// 	mux.Handle(path.Join("/api/notificationv2", httproute.MarkThreadRead), errorHandler(apiHandler.MarkThreadRead))
//
func New(
	service notification.Service,
	githubActivity, gerritActivity interface{ Status() string },
	users users.Service,
	opt Options,
) interface {
	ServeHTTP(w http.ResponseWriter, req *http.Request) error
} {
	return &handler{
		ns:               service,
		githubActivity:   githubActivity,
		gerritActivity:   gerritActivity,
		users:            users,
		assetsFileServer: httpgzip.FileServer(assets.Assets, httpgzip.FileServerOptions{ServeError: httpgzip.Detailed}),
		opt:              opt,
	}
}

// handler handles all requests to notifications app. It acts
// like a request multiplexer, choosing from various endpoints.
type handler struct {
	ns             notification.Service
	githubActivity interface{ Status() string }
	gerritActivity interface{ Status() string }
	users          users.Service

	assetsFileServer http.Handler

	opt Options
}

func (h *handler) ServeHTTP(w http.ResponseWriter, req *http.Request) error {
	// TODO: Caller still does a lot of work outside to calculate req.URL.Path by
	//       subtracting BaseURL from full original req.URL.Path. We should be able
	//       to compute it here internally by using req.RequestURI and BaseURL.

	switch {
	// Handle "/assets/...".
	case strings.HasPrefix(req.URL.Path, "/assets/"):
		req := stripPrefix(req, len("/assets"))
		h.assetsFileServer.ServeHTTP(w, req)
		return nil

	// Handle "/" and "/threads".
	case req.URL.Path == "/",
		req.URL.Path == "/threads":
		return h.frontendHandler(w, req)

	// Handle "/status".
	case req.URL.Path == "/status":
		return h.statusHandler(w, req)

	default:
		return httperror.HTTP{Code: http.StatusNotFound, Err: errors.New("no route")}
	}
}

// Options for configuring notifications app.
type Options struct {
	BaseURL string // Must have no trailing slash. Can be empty string.
	RedLogo bool
	HeadPre template.HTML
}

var frontendHTML = template.Must(template.New("").Parse(`<!DOCTYPE html>
<html lang="en">
	<head>
		{{.HeadPre}}
		<link href="{{.BaseURL}}/assets/stream.css" rel="stylesheet" type="text/css">
		<link href="{{.BaseURL}}/assets/thread.css" rel="stylesheet" type="text/css">
		<script>var RedLogo = {{.RedLogo}};</script>
		<script src="{{.BaseURL}}/assets/wasm_exec.js"></script>
		<script>
			if (!WebAssembly.instantiateStreaming) { // polyfill for Safari :/
				WebAssembly.instantiateStreaming = async (resp, importObject) => {
					const source = await (await resp).arrayBuffer();
					return await WebAssembly.instantiate(source, importObject);
				};
			}
			const go = new Go();
			const resp = fetch("{{.BaseURL}}/assets/frontend.wasm").then((resp) => {
				if (!resp.ok) {
					resp.text().then((body) => {
						document.body.innerHTML = "<pre>" + body + "</pre>";
					});
					throw new Error("did not get acceptable status code: " + resp.status);
				}
				return resp;
			});
			WebAssembly.instantiateStreaming(resp, go.importObject).then((result) => {
				go.run(result.instance);
			}).catch((error) => { document.body.textContent = error; });
			window.addEventListener('keydown', (event) => {
				if (event.key !== '®') {
					return;
				}
				WebAssembly.instantiateStreaming(fetch("{{.BaseURL}}/assets/frontend.wasm"), go.importObject).then((result) => {
					go.run(result.instance);
				}).catch((error) => { document.body.textContent = error; });
				event.preventDefault();
			});
		</script>
	</head>
	<body></body>
</html>`))

func (h *handler) frontendHandler(w http.ResponseWriter, req *http.Request) error {
	if err := httputil.AllowMethods(req, http.MethodGet, http.MethodHead); err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if req.Method == http.MethodHead {
		return nil
	}
	err := frontendHTML.Execute(w, struct {
		BaseURL string
		RedLogo bool
		HeadPre template.HTML
	}{h.opt.BaseURL, h.opt.RedLogo, h.opt.HeadPre})
	return err
}

func (h *handler) statusHandler(w http.ResponseWriter, req *http.Request) error {
	if err := httputil.AllowMethods(req, http.MethodGet, http.MethodHead); err != nil {
		return err
	}
	if user, err := h.users.GetAuthenticated(req.Context()); err != nil {
		return err
	} else if !user.SiteAdmin {
		return os.ErrPermission
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if req.Method == http.MethodHead {
		return nil
	}
	fmt.Fprintln(w, "GitHub Activity Service:", h.githubActivity.Status())
	fmt.Fprintln(w, "Gerrit Activity Service:", h.gerritActivity.Status())
	return nil
}

// stripPrefix returns request r with prefix of length prefixLen stripped from r.URL.Path.
// prefixLen must not be longer than len(r.URL.Path), otherwise stripPrefix panics.
// If r.URL.Path is empty after the prefix is stripped, the path is changed to "/".
func stripPrefix(r *http.Request, prefixLen int) *http.Request {
	r2 := new(http.Request)
	*r2 = *r
	r2.URL = new(url.URL)
	*r2.URL = *r.URL
	r2.URL.Path = r.URL.Path[prefixLen:]
	if r2.URL.Path == "" {
		r2.URL.Path = "/"
	}
	return r2
}
