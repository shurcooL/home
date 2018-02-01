// Package route specifies some route paths used by home.
package route

import "strings"

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

// HasImportPathSeparator reports whether path contains the import path separator.
func HasImportPathSeparator(path string) bool {
	return strings.IndexByte(path, importPathSeparator) != -1
}

func PkgIndex(pkgPath string) string     { return pkgPath }
func RepoIndex(repoPath string) string   { return repoPath + "/..." }
func RepoHistory(repoPath string) string { return repoPath + "/...$history" }
func RepoCommit(repoPath string) string  { return repoPath + "/...$commit" }
func RepoIssues(repoPath string) string  { return repoPath + "/...$issues" }
