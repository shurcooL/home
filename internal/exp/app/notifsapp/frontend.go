// +build js,wasm,go1.14

package notifsapp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"syscall/js"

	notifcomponent "github.com/shurcooL/home/internal/exp/app/notifsapp/component"
	"github.com/shurcooL/home/internal/exp/service/notification"
	"github.com/shurcooL/htmlg"
)

func (a *app) SetupPage(ctx context.Context, state interface{}) {
	st := state.(State)

	route := st.ReqURL.Path[len("/notifications"):]
	if route == "" {
		route = "/"
	}

	switch route {
	case "/":
		// TODO: Set MarkRead for stream and thread pages? Per-route setup?
		js.Global().Set("MarkRead", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
			el, namespace, threadType, threadID := args[0], args[1].String(), args[2].String(), uint64(args[3].Int())
			fmt.Printf("app.MarkStreamRead: %q, %q, %v\n", namespace, threadType, threadID)
			a.MarkStreamRead(el, namespace, threadType, threadID)
			return nil
		}))

		go func() {
			// Stream further notifications.
			err := stream(ctx, a.ns)
			if errors.Is(err, context.Canceled) {
				log.Println("stopped streaming cuz context canceled")
			} else if err != nil {
				log.Println("stream:", err)
				//js.Global().Get("document").Get("body").Set("innerHTML", "<pre>"+html.EscapeString(err.Error())+"</pre>")
				//return
			}
		}()

	case "/threads":
		// TODO: This is the MarkThreadRead-variant of MarkRead... make it work.
		js.Global().Set("MarkRead", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
			el, namespace, threadType, threadID := args[0], args[1].String(), args[2].String(), uint64(args[3].Int())
			fmt.Printf("app.MarkThreadRead: %q, %q, %v\n", namespace, threadType, threadID)
			a.MarkThreadRead(el, namespace, threadType, threadID)
			return nil
		}))
	}
}

func stream(ctx context.Context, notificationService notification.Service) error {
	show := js.Global().Get("document").Call("getElementById", "show")
	ch := make(chan []notification.Notification, 4)
	err := notificationService.StreamNotifications(ctx, ch)
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

func (a *app) MarkStreamRead(el js.Value, namespace string, threadType string, threadID uint64) {
	el.Set("disabled", true)
	go func() {
		err := a.ns.MarkThreadRead(context.Background(), namespace, threadType, threadID)
		if err != nil {
			log.Println("MarkThreadRead:", err)
			return
		}
		markStreamRead(namespace, threadType, threadID)
	}()
}

// markStreamRead updates the UI, marking all notifications from this thread as read.
func markStreamRead(namespace, threadType string, threadID uint64) {
	stream := js.Global().Get("document").Call("querySelector", "div.notificationStream")
	if stream.IsNull() {
		// TODO: This is needed because markStreamRead may get called by stream
		// after navigated away (due to a stray notification read coming in from
		// StreamNotifications). See if this can be improved.
		return
	}
	allNotifs := stream.Call("getElementsByClassName", "notification")
	for i := 0; i < allNotifs.Length(); i++ {
		n := allNotifs.Index(i)
		if n.Get("dataset").Get("namespace").String() != namespace ||
			n.Get("dataset").Get("threadtype").String() != threadType ||
			n.Get("dataset").Get("threadid").String() != strconv.FormatUint(threadID, 10) {
			continue
		}
		n.Get("style").Set("box-shadow", "none")                              // Hide blue edge marker.
		n.Call("querySelector", "button").Get("style").Set("display", "none") // Hide mark-read button.
	}
}

func (a *app) MarkThreadRead(el js.Value, namespace string, threadType string, threadID uint64) {
	if namespace == "" && threadType == "" && threadID == 0 {
		// When user clicks on the notification link, don't perform mark read operation
		// ourselves, it's expected to be done externally by the service that displays
		// the notification to the user views. Just make it appear as read, and return.
		markThreadRead(el)
		return
	}

	go func() {
		err := a.ns.MarkThreadRead(context.Background(), namespace, threadType, threadID)
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
