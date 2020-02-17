package main

import (
	"strings"

	"github.com/shurcooL/home/internal/route"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/octicon"
)

func patternTabnav(selected repositoryTab, pattern string, packages, openIssues, openChanges int) htmlg.Component {
	if !strings.HasPrefix(pattern, "dmitri.shuralyov.com/") {
		h3 := htmlg.H3(htmlg.Text("Packages"))
		return htmlg.NodeComponent(*h3)
	}
	patternPath := pattern[len("dmitri.shuralyov.com"):]
	return tabnav{
		Tabs: []tab{
			{
				Content: contentCounter{
					Content: iconText{Icon: octicon.Package, Text: "Packages"},
					Count:   packages,
				},
				URL:      route.PatternIndex(patternPath),
				Selected: selected == packagesTab,
			},
			/*{
				Content: iconText{Icon: octicon.History, Text: "History"},
				URL: route.PatternHistory(patternPath),
				Selected: selected == historyTab,
			},*/
			{
				Content: contentCounter{
					Content: iconText{Icon: octicon.IssueOpened, Text: "Issues"},
					Count:   openIssues,
				},
				URL:      route.PatternIssues(patternPath),
				Selected: selected == issuesTab,
			},
			/*{
				Content: contentCounter{
					Content: iconText{Icon: octicon.IssueOpened, Text: "Changes"},
					Count:   openChanges,
				},
				URL:      route.PatternChanges(patternPath),
				Selected: selected == changesTab,
			},*/
		},
	}
}
