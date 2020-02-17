package main

import (
	"github.com/shurcooL/home/internal/code"
	"github.com/shurcooL/home/internal/route"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/octicon"
)

// directoryGitPathspec returns a git pathspec that
// constraints git operations to only directory d.
func directoryGitPathspec(d *code.Directory) string {
	if d.ImportPath == d.RepoRoot {
		return ":(glob)*"
	}
	return ":(glob)" + d.ImportPath[len(d.RepoRoot)+len("/"):] + "/*"
}

// pathWithinRepo returns the path of directory d
// relative to repository root, or empty string if at root.
func pathWithinRepo(d *code.Directory) string {
	if d.ImportPath == d.RepoRoot {
		return ""
	}
	return d.ImportPath[len(d.RepoRoot)+len("/"):]
}

func directoryTabnav(selected repositoryTab, pkgPath string, openIssues, openChanges int) htmlg.Component {
	return tabnav{
		Tabs: []tab{
			{
				Content:  iconText{Icon: octicon.Package, Text: "Package"},
				URL:      route.PkgIndex(pkgPath),
				Selected: selected == packagesTab,
			},
			{
				Content:  iconText{Icon: octicon.History, Text: "History"},
				URL:      route.PkgHistory(pkgPath),
				Selected: selected == historyTab,
			},
			{
				Content: contentCounter{
					Content: iconText{Icon: octicon.IssueOpened, Text: "Issues"},
					Count:   openIssues,
				},
				URL:      route.PkgIssues(pkgPath),
				Selected: selected == issuesTab,
			},
			/*{
				Content: contentCounter{
					Content: iconText{Icon: octicon.GitPullRequest, Text: "Changes"},
					Count:   openChanges,
				},
				URL:      route.PkgChanges(pkgPath),
				Selected: selected == changesTab,
			},*/
		},
	}
}
