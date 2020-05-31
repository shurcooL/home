// +build js,wasm

// frontend renders notifications app pages on the frontend.
// It is a Go command meant to be compiled with GOOS=js GOARCH=wasm
// and executed in a browser, where the DOM is available.
package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"syscall/js"

	homecomponent "github.com/shurcooL/home/component"
	homehttp "github.com/shurcooL/home/http"
	notifcomponent "github.com/shurcooL/home/internal/exp/app/notifications/component"
	"github.com/shurcooL/home/internal/exp/service/notification"
	"github.com/shurcooL/home/internal/exp/service/notification/httpclient"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/users"
	"golang.org/x/net/html"
	"golang.org/x/oauth2"
)

func main() {
	homecomponent.RedLogo = js.Global().Get("RedLogo").Bool()

	notificationService := httpclient.NewNotification(httpClient(), "", "", "/api/notificationv2")
	usersService := homehttp.Users{}
	authenticatedUser, err := usersService.GetAuthenticated(context.Background())
	if err != nil {
		log.Println(err)
		authenticatedUser = users.User{} // THINK: Should it be a fatal error or not? What about on frontend vs backend?
	}

	// TODO: strip "/notificationsv2" prefix somehow?
	switch reqURL := requestURL(); reqURL.Path {
	case "/notificationsv2":
		gopherbot, _ := strconv.ParseBool(reqURL.Query().Get("gopherbot"))

		// Initial page render.
		var buf bytes.Buffer
		err := renderStreamBodyInnerHTML(context.Background(), &buf, reqURL, gopherbot, notificationService, authenticatedUser)
		if err != nil {
			log.Println(err)
			js.Global().Get("document").Get("body").Set("innerHTML", "<pre>"+html.EscapeString(err.Error())+"</pre>")
			return
		}
		js.Global().Get("document").Get("body").Set("innerHTML", buf.String())

		f := frontend{ns: notificationService}
		js.Global().Set("MarkRead", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
			namespace, threadType, threadID := args[0].String(), args[1].String(), uint64(args[2].Int())
			fmt.Printf("frontend: MarkStreamRead: %q, %q, %v\n", namespace, threadType, threadID)
			f.MarkStreamRead(namespace, threadType, threadID)
			return nil
		}))

		// Stream further notifications.
		err = stream(notificationService)
		if err != nil {
			log.Println(err)
			js.Global().Get("document").Get("body").Set("innerHTML", "<pre>"+html.EscapeString(err.Error())+"</pre>")
			return
		}
	case "/notificationsv2/threads":
		all, _ := strconv.ParseBool(reqURL.Query().Get("all"))

		// Initial page render.
		var buf bytes.Buffer
		err := renderThreadBodyInnerHTML(context.Background(), &buf, reqURL, all, notificationService, authenticatedUser)
		if err != nil {
			log.Println(err)
			js.Global().Get("document").Get("body").Set("innerHTML", "<pre>"+html.EscapeString(err.Error())+"</pre>")
			return
		}
		js.Global().Get("document").Get("body").Set("innerHTML", buf.String())

		f := frontend{ns: notificationService}
		js.Global().Set("MarkRead", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
			el, namespace, threadType, threadID := args[0], args[1].String(), args[2].String(), uint64(args[3].Int())
			fmt.Printf("frontend: MarkThreadRead: %q, %q, %v\n", namespace, threadType, threadID)
			f.MarkThreadRead(el, namespace, threadType, threadID)
			return nil
		}))
		select {}
	default:
		err := fmt.Errorf("page %q not found", reqURL.Path)
		js.Global().Get("document").Get("body").Set("innerHTML", "<pre>"+html.EscapeString(err.Error())+"</pre>")
		return
	}
}

func stream(notificationService notification.Service) error {
	show := js.Global().Get("document").Call("getElementById", "show")
	ch := make(chan []notification.Notification, 4)
	err := notificationService.StreamNotifications(context.Background(), ch)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	for notifs := range ch {
		for _, n := range notifs {
			if !n.Unread {
				// This notification thread has been marked read.
				// TODO: use another type, not notification.Notification maybe?
				markStreamRead(n.Namespace, n.ThreadType, n.ThreadID)
				continue
			}

			if notifcomponent.GopherBotNotification(n) {
				continue // Skip events posted via gopherbot comments.
			}

			// Show browser notification.
			if show.Get("checked").Bool() {
				title, body, ok := browserNotification(n)
				if !ok {
					continue
				}
				js.Global().Get("Notification").New(title, map[string]interface{}{
					"icon": n.Actor.AvatarURL,
					"body": body,
				})
			}

			// Add notification to HTML page.
			notif, ok := notifcomponent.RenderNotification(n)
			if !ok {
				continue
			}
			buf.Reset()
			err := htmlg.RenderComponents(&buf, notif)
			if err != nil {
				return err
			}
			today := js.Global().Get("document").Call("querySelector", "div.notificationStream div.heading")
			today.Call("insertAdjacentHTML", "afterend", buf.String())
		}
	}

	return nil
}

func browserNotification(n notification.Notification) (title, body string, ok bool) {
	switch p := n.Payload.(type) {
	case notification.Issue:
		return n.Actor.Login + " " + p.Action + " an issue", n.ImportPaths[0] + ": " + p.IssueTitle, true
	case notification.Change:
		return n.Actor.Login + " " + p.Action + " a change", n.ImportPaths[0] + ": " + p.ChangeTitle, true
	case notification.IssueComment:
		return n.Actor.Login + " " + "commented", n.ImportPaths[0] + ": " + p.IssueTitle, true
	case notification.ChangeComment:
		var verb string
		switch p.CommentReview {
		case 0:
			verb = "commented"
		default:
			verb = fmt.Sprintf("reviewed %+d", p.CommentReview)
		}
		return n.Actor.Login + " " + verb, n.ImportPaths[0] + ": " + p.ChangeTitle, true
	default:
		log.Printf("browserNotification: unexpected notification type: %T\n", p)
		return "", "", false
	}
}

type frontend struct {
	ns notification.Service
}

func (f frontend) MarkStreamRead(namespace string, threadType string, threadID uint64) {
	go func() {
		err := f.ns.MarkThreadRead(context.Background(), namespace, threadType, threadID)
		if err != nil {
			log.Println("MarkThreadRead:", err)
			return
		}
		markStreamRead(namespace, threadType, threadID)
	}()
}

// markStreamRead updates the UI, marking all notifications from this thread as read.
func markStreamRead(namespace, threadType string, threadID uint64) {
	allNotifs := js.Global().Get("document").Call("querySelector", "div.notificationStream").
		Call("getElementsByClassName", "notification")
	for i := 0; i < allNotifs.Length(); i++ {
		n := allNotifs.Index(i)
		if n.Get("dataset").Get("namespace").String() != namespace ||
			n.Get("dataset").Get("threadtype").String() != threadType ||
			n.Get("dataset").Get("threadid").String() != strconv.FormatUint(threadID, 10) {
			continue
		}
		n.Get("style").Set("box-shadow", "none")                              // Hide blue edge marker.
		n.Call("querySelector", "a.icon").Get("style").Set("display", "none") // Hide mark-read button.
	}
}

func (f frontend) MarkThreadRead(el js.Value, namespace string, threadType string, threadID uint64) {
	if namespace == "" && threadType == "" && threadID == 0 {
		// When user clicks on the notification link, don't perform mark read operation
		// ourselves, it's expected to be done externally by the service that displays
		// the notification to the user views. Just make it appear as read, and return.
		markThreadRead(el)
		return
	}

	go func() {
		err := f.ns.MarkThreadRead(context.Background(), namespace, threadType, threadID)
		if err != nil {
			log.Println("MarkThreadRead:", err)
			return
		}
		markThreadRead(el)
	}()
}

// markThreadRead marks the notification thread containing element el as read.
func markThreadRead(el js.Value) {
	// Mark this particular notification thread as read.
	getAncestorByClassName(el, "mark-as-read").Get("classList").Call("add", "read")

	// If all notifications within the parent RepoNotifications are read by now,
	// then mark entire RepoNotifications group as read.
	repo := getAncestorByClassName(el, "RepoNotifications")
	if repo.Call("querySelectorAll", ".read").Length() ==
		repo.Call("querySelectorAll", ".mark-as-read").Length() {
		repo.Get("classList").Call("add", "read")
	}
}

func getAncestorByClassName(el js.Value, class string) js.Value {
	for ; !el.IsNull() && !el.Get("classList").Call("contains", class).Bool(); el = el.Get("parentElement") {
	}
	return el
}

// requestURL returns the effective HTTP request URL of the current page.
func requestURL() *url.URL {
	u, err := url.Parse(js.Global().Get("location").Get("href").String())
	if err != nil {
		log.Fatalln(err)
	}
	u.Scheme, u.Opaque, u.User, u.Host = "", "", nil, ""
	return u
}

// httpClient returns an *http.Client suitable for making authenticated API requests.
func httpClient() *http.Client {
	cookies := &http.Request{Header: http.Header{"Cookie": {js.Global().Get("document").Get("cookie").String()}}}
	accessToken, err := cookies.Cookie("accessToken")
	if err != nil {
		// Unauthenticated client.
		return http.DefaultClient
	}
	// Authenticated client.
	return oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken.Value},
	))
}
