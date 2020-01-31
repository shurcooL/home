// Package gcpfetch provides a Google Cloud Platform-powered
// implementation of auth.FetchService.
package gcpfetch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/shurcooL/home/internal/exp/service/auth"
	"github.com/shurcooL/users"
	"golang.org/x/oauth2/google"
)

// remoteFetchTimeout is the timeout for remote function call.
//
// It's computed by taking the fetch timeout, and adding some
// buffer on top of that to account for remote communication.
const remoteFetchTimeout = auth.FetchTimeout + 10*time.Second

type service struct {
	funcURL string
	cl      *http.Client // HTTP client for making calls to the cloud function.
}

// NewService creates an auth.FetchService that delegates the fetch work
// to a remote Google Cloud Function that provides the fetch service.
//
// funcURL is the URL of a Google Cloud Function that provides the fetch service.
// keyFile optionally specifies a key file to use.
func NewService(funcURL, keyFile string) (auth.FetchService, error) {
	cl := http.DefaultClient
	if keyFile != "" {
		b, err := ioutil.ReadFile(keyFile)
		if err != nil {
			return nil, err
		}
		conf, err := google.JWTConfigFromJSON(b)
		if err != nil {
			return nil, err
		}
		conf.PrivateClaims = map[string]interface{}{"target_audience": funcURL}
		conf.UseIDToken = true
		cl = conf.Client(context.Background())
	}
	return service{
		funcURL: funcURL,
		cl:      cl,
	}, nil
}

// FetchUserProfile implements auth.FetchService.
func (s service) FetchUserProfile(ctx context.Context, me *url.URL) (auth.UserProfile, error) {
	// Do a fetch using the remote Cloud Function at funcURL.
	// Set the timeout.
	ctx, cancel := context.WithTimeout(ctx, remoteFetchTimeout)
	defer cancel()
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(struct{ Me *url.URL }{me}) // TODO: Automate the conversion if possible...
	if err != nil {
		return auth.UserProfile{}, internalError{err}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.funcURL+"?method=FetchUserProfile", &buf)
	if err != nil {
		return auth.UserProfile{}, internalError{err}
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.cl.Do(req)
	if err != nil {
		return auth.UserProfile{}, internalError{err}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return auth.UserProfile{}, internalError{fmt.Errorf("non-200 OK status code: %v", resp.Status)}
	} else if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		return auth.UserProfile{}, internalError{fmt.Errorf("got Content-Type %q, want %q", ct, "application/json")}
	}
	var v struct {
		auth.UserProfile
		Error *string
	}
	err = json.NewDecoder(resp.Body).Decode(&v)
	if err != nil {
		return auth.UserProfile{}, internalError{err}
	}
	if e := v.Error; e != nil {
		return auth.UserProfile{}, errors.New(*e)
	}
	return v.UserProfile, nil
}

// FetchGitHubUser implements auth.FetchService.
func (s service) FetchGitHubUser(ctx context.Context, login string) (_ users.User, websiteURL string, _ error) {
	// Do a fetch using the remote Cloud Function at funcURL.
	// Set the timeout.
	ctx, cancel := context.WithTimeout(ctx, remoteFetchTimeout)
	defer cancel()
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(struct{ Login string }{login}) // TODO: Automate the conversion if possible...
	if err != nil {
		return users.User{}, "", internalError{err}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.funcURL+"?method=FetchGitHubUser", &buf)
	if err != nil {
		return users.User{}, "", internalError{err}
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.cl.Do(req)
	if err != nil {
		return users.User{}, "", internalError{err}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return users.User{}, "", internalError{fmt.Errorf("non-200 OK status code: %v", resp.Status)}
	} else if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		return users.User{}, "", internalError{fmt.Errorf("got Content-Type %q, want %q", ct, "application/json")}
	}
	var v struct {
		User       users.User
		WebsiteURL string
		Error      *string
	}
	err = json.NewDecoder(resp.Body).Decode(&v)
	if err != nil {
		return users.User{}, "", internalError{err}
	}
	if e := v.Error; e != nil {
		return users.User{}, "", errors.New(*e)
	}
	return v.User, v.WebsiteURL, nil
}

// internalError represents an internal error,
// whose details are not meant to be shown to end users.
type internalError struct {
	Err error // Non-nil.
}

func (internalError) Error() string { return "internal error" }
