// Package route specifies some route paths used by home.
package route

import (
	"fmt"
	"strings"
)

const importPathSeparator = '$'

// BeforeImportPathSeparator returns the path before the import path separator.
func BeforeImportPathSeparator(path string) string {
	switch i := strings.IndexByte(path, importPathSeparator); i {
	default:
		return path[:i]
	case -1:
		return path
	}
}

// SplitImportPathSeparator returns the path before the import path separator,
// and the path before the import path separator (including the separator, if any).
func SplitImportPathSeparator(path string) (before, after string) {
	switch i := strings.IndexByte(path, importPathSeparator); i {
	default:
		return path[:i], path[i:]
	case -1:
		return path, ""
	}
}

// HasImportPathSeparator reports whether path contains the import path separator.
func HasImportPathSeparator(path string) bool {
	return strings.IndexByte(path, importPathSeparator) != -1
}

func PkgIndex(pkgPath string) string   { return pkgPath }
func PkgLicense(pkgPath string) string { return pkgPath + "$file/LICENSE" }
func PkgHistory(pkgPath string) string { return pkgPath + "$history" }
func PkgCommit(pkgPath string) string  { return pkgPath + "$commit" }
func PkgIssues(pkgPath string) string  { return pkgPath + "$issuesv2" } // TODO.
func PkgIssue(pkgPath string, issueID int64) string {
	return pkgPath + fmt.Sprintf("$issue/%d", issueID)
}
func PkgChanges(pkgPath string) string { return pkgPath + "$changesv2" } // TODO.
func PkgChange(pkgPath string, changeID int64) string {
	return pkgPath + fmt.Sprintf("$change/%d", changeID)
}

func PatternIndex(patternPath string) string   { return patternPath }
func PatternIssues(patternPath string) string  { return patternPath + "$issuesv2" }  // TODO.
func PatternChanges(patternPath string) string { return patternPath + "$changesv2" } // TODO.

func RepoIndex(repoPath string) string   { return repoPath + "/..." }
func RepoHistory(repoPath string) string { return repoPath + "/...$history" }
func RepoCommit(repoPath string) string  { return repoPath + "/...$commit" }
func RepoIssues(repoPath string) string  { return repoPath + "/...$issuesv2" }  // TODO.
func RepoChanges(repoPath string) string { return repoPath + "/...$changesv2" } // TODO.

func RepoIssuesV1(repoPath string) string  { return repoPath + "/...$issues" }  // TODO: Remove.
func RepoChangesV1(repoPath string) string { return repoPath + "/...$changes" } // TODO: Remove.
