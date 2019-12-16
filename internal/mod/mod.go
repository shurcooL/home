// Package mod exposes select functionality related to module mechanics.
// Its code is mostly copied from cmd/go/internal/... packages.
package mod

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
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

// ParseV000PseudoVersion returns the time stamp and the revision identifier
// of the v0.0.0 pseudo-version v.
// It returns an error if v is not a valid v0.0.0 pseudo-version
// of the form "v0.0.0-yyyymmddhhmmss-abcdef123456".
func ParseV000PseudoVersion(v string) (_ time.Time, rev string, err error) {
	if len(v) != len("v0.0.0-yyyymmddhhmmss-abcdef123456") ||
		!strings.HasPrefix(v, "v0.0.0-") ||
		v[len("v0.0.0-yyyymmddhhmmss")] != '-' {
		return time.Time{}, "", fmt.Errorf("not a v0.0.0 pseudo-version %q", v)
	}
	timestamp, rev := v[len("v0.0.0-"):len("v0.0.0-yyyymmddhhmmss")], v[len("v0.0.0-yyyymmddhhmmss-"):]
	if !allDec(timestamp) ||
		!allHex(rev) {
		return time.Time{}, "", fmt.Errorf("not a v0.0.0 pseudo-version %q", v)
	}
	t, err := time.Parse("20060102150405", timestamp)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("pseudo-version with malformed time %s: %q", timestamp, v)
	}
	return t, rev, nil
}

// allDec reports whether timestamp is entirely decimal digits.
func allDec(timestamp string) bool {
	for _, b := range []byte(timestamp) {
		ok := '0' <= b && b <= '9'
		if !ok {
			return false
		}
	}
	return true
}

// allHex reports whether the revision rev is entirely lower-case hexadecimal digits.
func allHex(rev string) bool {
	for _, b := range []byte(rev) {
		ok := '0' <= b && b <= '9' || 'a' <= b && b <= 'f'
		if !ok {
			return false
		}
	}
	return true
}

// HashZip is like dirhash.HashZip, but the .zip file
// it takes can be in memory rather than on disk.
func HashZip(b []byte, hash dirhash.Hash) (string, error) {
	z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
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
