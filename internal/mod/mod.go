// Package mod exposes select functionality related to module mechanics.
// Its code is mostly copied from cmd/go/internal/... packages.
package mod

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"golang.org/x/mod/semver"
	"golang.org/x/mod/sumdb/dirhash"
)

// A RevInfo describes a single revision in a module repository.
type RevInfo struct {
	Version string    // version string
	Time    time.Time // commit time
}

// PseudoVersion returns a pseudo-version for the given major version ("v1")
// preexisting older tagged version ("" or "v1.2.3" or "v1.2.3-pre"), revision time,
// and revision identifier (usually a 12-byte commit hash prefix).
func PseudoVersion(major, older string, t time.Time, rev string) string {
	if major == "" {
		major = "v0"
	}
	major = strings.TrimSuffix(major, "-unstable") // make gopkg.in/macaroon-bakery.v2-unstable use "v2"
	segment := fmt.Sprintf("%s-%s", t.UTC().Format("20060102150405"), rev)
	build := semver.Build(older)
	older = semver.Canonical(older)
	if older == "" {
		return major + ".0.0-" + segment // form (1)
	}
	if semver.Prerelease(older) != "" {
		return older + ".0." + segment + build // form (4), (5)
	}

	// Form (2), (3).
	// Extract patch from vMAJOR.MINOR.PATCH
	v := older[:]
	i := strings.LastIndex(v, ".") + 1
	v, patch := v[:i], v[i:]

	// Increment PATCH by adding 1 to decimal:
	// scan right to left turning 9s to 0s until you find a digit to increment.
	// (Number might exceed int64, but math/big is overkill.)
	digits := []byte(patch)
	for i = len(digits) - 1; i >= 0 && digits[i] == '9'; i-- {
		digits[i] = '0'
	}
	if i >= 0 {
		digits[i]++
	} else {
		// digits is all zeros
		digits[0] = '1'
		digits = append(digits, '0')
	}
	patch = string(digits)

	// Reassemble.
	return v + patch + "-0." + segment + build
}

// ParsePseudoVersion returns the time stamp and the revision identifier
// of the pseudo-version v.
// It returns an error if v is not a pseudo-version or if the time stamp
// embedded in the pseudo-version is not a valid time.
func ParsePseudoVersion(v string) (_ time.Time, rev string, err error) {
	timestamp, rev, err := parsePseudoVersion(v)
	if err != nil {
		return time.Time{}, "", err
	}
	t, err := time.Parse("20060102150405", timestamp)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("pseudo-version with malformed time %s: %q", timestamp, v)
	}
	return t, rev, nil
}

func parsePseudoVersion(v string) (timestamp, rev string, err error) {
	if !isPseudoVersion(v) {
		return "", "", fmt.Errorf("malformed pseudo-version %q", v)
	}
	v = strings.TrimSuffix(v, "+incompatible")
	j := strings.LastIndex(v, "-")
	v, rev = v[:j], v[j+1:]
	i := strings.LastIndex(v, "-")
	if j := strings.LastIndex(v, "."); j > i {
		timestamp = v[j+1:]
	} else {
		timestamp = v[i+1:]
	}
	return timestamp, rev, nil
}

// isPseudoVersion reports whether v is a pseudo-version.
func isPseudoVersion(v string) bool {
	return strings.Count(v, "-") >= 2 && semver.IsValid(v) && pseudoVersionRE.MatchString(v)
}

var pseudoVersionRE = regexp.MustCompile(`^v[0-9]+\.(0\.0-|\d+\.\d+-([^+]*\.)?0\.)\d{14}-[A-Za-z0-9]+(\+incompatible)?$`)

// AllHex reports whether the revision rev is entirely lower-case hexadecimal digits.
func AllHex(rev string) bool {
	for i := 0; i < len(rev); i++ {
		c := rev[i]
		if '0' <= c && c <= '9' || 'a' <= c && c <= 'f' {
			continue
		}
		return false
	}
	return true
}

// HashZip is like dirhash.HashZip, but the .zip file
// it takes can be in memory rather than on disk.
func HashZip(r *bytes.Reader, hash dirhash.Hash) (string, error) {
	z, err := zip.NewReader(r, int64(r.Len()))
	if err != nil {
		return "", err
	}
	var files []string
	zfiles := make(map[string]*zip.File)
	for _, file := range z.File {
		files = append(files, file.Name)
		zfiles[file.Name] = file
	}
	zipOpen := func(name string) (io.ReadCloser, error) {
		f := zfiles[name]
		if f == nil {
			return nil, fmt.Errorf("file %q not found in zip", name) // should never happen
		}
		return f.Open()
	}
	return hash(files, zipOpen)
}
