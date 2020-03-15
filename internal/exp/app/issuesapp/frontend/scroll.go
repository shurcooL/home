// +build js,wasm,go1.14

package main

import (
	"fmt"
	"net/url"
	"strings"
	"syscall/js"
	"time"

	"github.com/shurcooL/go/gopherjs_http/jsutil/v2"
	"honnef.co/go/js/dom/v2"
)

func setupScroll() {
	js.Global().Set("AnchorScroll", jsutil.Wrap(AnchorScroll))

	// Start watching for hashchange events.
	dom.GetWindow().AddEventListener("hashchange", false, func(event dom.Event) {
		processHash()

		event.PreventDefault()
	})

	document.Body().AddEventListener("keydown", false, func(event dom.Event) {
		if event.DefaultPrevented() {
			return
		}

		switch ke := event.(*dom.KeyboardEvent); {
		// Escape.
		case ke.KeyCode() == 27 && !ke.Repeat() && !ke.CtrlKey() && !ke.AltKey() && !ke.MetaKey() && !ke.ShiftKey():
			if strings.TrimPrefix(dom.GetWindow().Location().Hash(), "#") == "" {
				return
			}

			setFragment("")

			highlight(nil)

			ke.PreventDefault()
		}
	})

	// Jump to desired hash slightly after page loads (override browser's default hash jumping).
	go func() {
		// This needs to be delayed, or else it "happens too early".
		time.Sleep(time.Millisecond)
		processHash()
	}()
}

func processHash() {
	// Scroll to hash target.
	targetID := strings.TrimPrefix(dom.GetWindow().Location().Hash(), "#")
	target, ok := document.GetElementByID(targetID).(dom.HTMLElement)
	if ok {
		centerWindowOn(target)
	}

	highlight(target)
}

// AnchorScroll scrolls window to target that is pointed by fragment of href of given anchor element.
// It must point to a valid target.
func AnchorScroll(anchor dom.HTMLElement, e dom.Event) {
	url, err := url.Parse(anchor.(*dom.HTMLAnchorElement).Href())
	if err != nil {
		// Should never happen if AnchorScroll is used correctly.
		panic(fmt.Errorf("AnchorScroll: url.Parse: %v", err))
	}
	targetID := url.Fragment
	target := document.GetElementByID(targetID).(dom.HTMLElement)

	setFragment(targetID)

	// TODO: Decide if it's better to do this or not to.
	centerWindowOn(target)

	highlight(target)

	e.PreventDefault()
}

// highlight highlights the selected element by giving it a "hash-selected" class.
// target can be nil to highlight nothing.
func highlight(target dom.HTMLElement) {
	// Clear all past highlights.
	for _, e := range document.GetElementsByClassName("hash-selected") {
		e.Class().Remove("hash-selected")
	}

	// Highlight target, if any.
	if target == nil {
		return
	}
	target.Class().Add("hash-selected")
}

// centerWindowOn scrolls window so that (the middle of) target is in the middle of window.
func centerWindowOn(target dom.HTMLElement) {
	windowHalfHeight := dom.GetWindow().InnerHeight() / 2
	targetHalfHeight := target.OffsetHeight() / 2
	if targetHalfHeight > float64(windowHalfHeight)*0.8 { // Prevent top of target from being offscreen.
		targetHalfHeight = float64(windowHalfHeight) * 0.8
	}
	dom.GetWindow().ScrollTo(dom.GetWindow().ScrollX(), int(offsetTopRoot(target)+targetHalfHeight)-windowHalfHeight)
}

// offsetTopRoot returns the offset top of element e relative to root element.
func offsetTopRoot(e dom.HTMLElement) float64 {
	var offsetTopRoot float64
	for ; e != nil; e = e.OffsetParent() {
		offsetTopRoot += e.OffsetTop()
	}
	return offsetTopRoot
}

// setFragment sets current page URL fragment to hash. The leading '#' shouldn't be included.
func setFragment(hash string) {
	url := windowLocation
	url.Fragment = hash
	// TODO: dom.GetWindow().History().ReplaceState(...), blocked on https://github.com/dominikh/go-js-dom/issues/41.
	js.Global().Get("window").Get("history").Call("replaceState", nil, nil, url.String())
}

var windowLocation = func() url.URL {
	url, err := url.Parse(dom.GetWindow().Location().Href())
	if err != nil {
		// We don't expect this can ever happen, so treat it as an internal error if it does.
		panic(fmt.Errorf("internal error: parsing window.location.href as URL failed: %v", err))
	}
	return *url
}()
