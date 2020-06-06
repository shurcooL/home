// +build go1.14

package notifsapp

import (
	homecomponent "github.com/shurcooL/home/component"
	"github.com/shurcooL/htmlg"
)

type notificationTab uint8

const (
	noTab notificationTab = iota
	streamTab
	threadTab
)

func notificationTabnav(selected notificationTab) htmlg.Component {
	return homecomponent.TabNav{
		Tabs: []homecomponent.Tab{
			{
				Content: htmlg.NodeComponent(*htmlg.Text("Stream")),
				URL:     "/notificationsv2", OnClick: "Open(event, this)",
				Selected: selected == streamTab,
			},
			{
				Content: htmlg.NodeComponent(*htmlg.Text("Threads")),
				URL:     "/notificationsv2/threads", OnClick: "Open(event, this)",
				Selected: selected == threadTab,
			},
		},
	}
}
