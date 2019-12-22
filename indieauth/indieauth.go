// Package indieauth implements building blocks for the IndieAuth
// specification (https://indieauth.spec.indieweb.org/).
//
// The functionality and API of this package is v0,
// meaning it is in early development and may change.
// There are no compatibility guarantees made.
package indieauth

import (
	"flag"
	"fmt"
	"net"
	"net/url"
	"path"
	"strings"
)

// ParseProfileURL parses a profile URL that a user is identified by.
//
// It verifies the restrictions as described in the IndieAuth specification
// at https://indieauth.spec.indieweb.org/#user-profile-url:
//
// 	Profile URLs
// 		MUST have either an https or http scheme,
// 		MUST contain a path component (/ is a valid path),
// 		MUST NOT contain single-dot or double-dot path segments,
// 		MAY contain a query string component,
// 		MUST NOT contain a fragment component,
// 		MUST NOT contain a username or password component, and
// 		MUST NOT contain a port.
// 	Additionally, hostnames
// 		MUST be domain names and
// 		MUST NOT be ipv4 or ipv6 addresses.
//
// It applies a few additional restrictions for now.
//
func ParseProfileURL(me string) (*url.URL, error) {
	if len(me) > 50 {
		return nil, fmt.Errorf("URL should not be longer than 50 bytes (for now)")
	}
	u, err := url.Parse(me)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "https" {
		// Require "https" scheme. This is stricter than IndieAuth spec requires.
		return nil, fmt.Errorf("URL scheme must be https")
	}
	if u.Path == "" {
		// Canonicalize empty path to "/" to meet the requirement of special URLs.
		//
		// See https://indieauth.spec.indieweb.org/#url-canonicalization
		// and https://url.spec.whatwg.org/#special-scheme.
		u.Path = "/"
	}
	if path.Clean("/"+u.Path) != u.Path || u.RawPath != "" {
		return nil, fmt.Errorf("URL path must be clean")
	}
	if u.Fragment != "" {
		return nil, fmt.Errorf("URL must not have a fragment")
	}
	if u.User != nil {
		return nil, fmt.Errorf("URL must not have a username or password")
	}
	if u.Port() != "" {
		return nil, fmt.Errorf("URL must not contain a port")
	}
	if !strings.Contains(u.Host, ".") {
		return nil, fmt.Errorf("URL must be a domain name (contain a dot)")
	}
	if net.ParseIP(u.Host) != nil {
		return nil, fmt.Errorf("URL must not be an IP")
	}
	u.Host = strings.ToLower(u.Host)
	return u, nil
}

// ParseClientID parses a client ID URL that a client is identified by.
//
// It verifies the restrictions as described in the IndieAuth specification
// at https://indieauth.spec.indieweb.org/#client-identifier:
//
// 	Client identifier URLs
// 		MUST have either an https or http scheme,
// 		MUST contain a path component,
// 		MUST NOT contain single-dot or double-dot path segments,
// 		MAY contain a query string component,
// 		MUST NOT contain a fragment component,
// 		MUST NOT contain a username or password component,
// 		and MAY contain a port.
// 	Additionally, hostnames
// 		MUST be domain names or a loopback interface and
// 		MUST NOT be IPv4 or IPv6 addresses except for IPv4 127.0.0.1 or IPv6 [::1].
//
// It applies a few additional restrictions for now.
//
func ParseClientID(clientID string) (*url.URL, error) {
	if len(clientID) > 50 {
		return nil, fmt.Errorf("URL should not be longer than 50 bytes (for now)")
	}
	u, err := url.Parse(clientID)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "https" {
		// Require "https" scheme. This is stricter than IndieAuth spec requires.
		return nil, fmt.Errorf("URL scheme must be https")
	}
	if u.Path == "" {
		return nil, fmt.Errorf("URL path must not be empty")
	}
	if path.Clean("/"+u.Path) != u.Path || u.RawPath != "" {
		return nil, fmt.Errorf("URL path must be clean")
	}
	if u.Fragment != "" {
		return nil, fmt.Errorf("URL must not have a fragment")
	}
	if u.User != nil {
		return nil, fmt.Errorf("URL must not have a username or password")
	}
	hostname := u.Hostname()
	if ip := net.ParseIP(hostname); ip != nil && !ip.IsLoopback() {
		return nil, fmt.Errorf("if the URL hostname is an IP, it must be loopback")
	} else if !strings.Contains(hostname, ".") && hostname != "localhost" {
		return nil, fmt.Errorf("URL must be a domain name (contain a dot) or a loopback interface")
	}
	u.Host = strings.ToLower(u.Host)
	return u, nil
}

// MeFlag defines a user profile URL flag with specified name, default value, and usage string.
// The return value is the address of a meFlag variable that stores the value of the flag.
// The flag accepts the canonical form of a value acceptable to ParseProfileURL,
// or the empty string. MeFlag panics if the provided default value is not acceptable.
func MeFlag(name string, value string, usage string) *meFlag {
	f := new(meFlag)
	err := f.Set(value)
	if err != nil {
		panic(fmt.Errorf("MeFlag: default value %q was rejected by Set: %v", value, err))
	}
	flag.CommandLine.Var(f, name, usage)
	return f
}

// meFlag implements flag.Value for a user profile URL flag.
type meFlag struct {
	// Me is a valid user profile URL, or nil.
	Me *url.URL
}

func (f *meFlag) Set(s string) error {
	if s == "" {
		f.Me = nil
		return nil
	}
	me, err := ParseProfileURL(s)
	if err != nil {
		return err
	} else if canonicalMe := me.String(); s != canonicalMe {
		return fmt.Errorf("value %q is not the canonical form of a user profile URL, should be %q (see https://indieauth.spec.indieweb.org/#user-profile-url)", s, canonicalMe)
	}
	f.Me = me
	return nil
}

func (f *meFlag) String() string {
	if f.Me == nil {
		return "<nil>"
	}
	return f.Me.String()
}
