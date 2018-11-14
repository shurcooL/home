package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/shurcooL/reactions"
	"golang.org/x/net/context/ctxhttp"
)

// Reactions implements reactions.Service remotely over HTTP.
type Reactions struct{}

func (Reactions) List(ctx context.Context, uri string) (map[string][]reactions.Reaction, error) {
	u := url.URL{Path: "/api/react/list", RawQuery: url.Values{"ReactableURL": {uri}}.Encode()}
	resp, err := ctxhttp.Get(ctx, nil, u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("did not get acceptable status code: %v body: %q", resp.Status, body)
	}
	var rm map[string][]reactions.Reaction
	err = json.NewDecoder(resp.Body).Decode(&rm)
	return rm, err
}

func (Reactions) Get(ctx context.Context, uri string, id string) ([]reactions.Reaction, error) {
	u := url.URL{Path: "/api/react", RawQuery: url.Values{"reactableURL": {uri}, "reactableID": {id}}.Encode()}
	resp, err := ctxhttp.Get(ctx, nil, u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("did not get acceptable status code: %v body: %q", resp.Status, body)
	}
	var rs []reactions.Reaction
	err = json.NewDecoder(resp.Body).Decode(&rs)
	return rs, err
}

func (Reactions) Toggle(ctx context.Context, uri string, id string, tr reactions.ToggleRequest) ([]reactions.Reaction, error) {
	resp, err := ctxhttp.PostForm(ctx, nil, "/api/react", url.Values{"reactableURL": {uri}, "reactableID": {id}, "reaction": {string(tr.Reaction)}})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("did not get acceptable status code: %v body: %q", resp.Status, body)
	}
	var rs []reactions.Reaction
	err = json.NewDecoder(resp.Body).Decode(&rs)
	return rs, err
}
