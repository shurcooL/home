// +build js,wasm,go1.14

package changesapp

import (
	"context"
	"syscall/js"

	"github.com/shurcooL/frontend/reactionsmenu/v2"
	"github.com/shurcooL/go/gopherjs_http/jsutil/v2"
	"honnef.co/go/js/dom/v2"
)

var document = dom.GetWindow().Document().(dom.HTMLDocument)

func (a *app) SetupPage(ctx context.Context, state interface{}) {
	st := state.(State)

	js.Global().Set("ToggleDetails", jsutil.Wrap(ToggleDetails))

	a.setupScroll(ctx, st)

	// TODO: Make this work better across page navigation.
	reactionsService := ChangeReactions{Change: a.cs}
	reactionsmenu.Setup(st.RepoSpec, reactionsService, st.CurrentUser)
}

func ToggleDetails(el dom.HTMLElement) {
	container := getAncestorByClassName(el, "commit-container").(dom.HTMLElement)
	details := container.QuerySelector("pre.commit-details").(dom.HTMLElement)

	switch details.Style().GetPropertyValue("display") {
	default:
		details.Style().SetProperty("display", "none", "")
	case "none":
		details.Style().SetProperty("display", "block", "")
	}
}

func getAncestorByClassName(el dom.Element, class string) dom.Element {
	for ; el != nil && !el.Class().Contains(class); el = el.ParentElement() {
	}
	return el
}
