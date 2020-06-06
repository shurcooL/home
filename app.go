package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"net/http"

	"github.com/shurcooL/home/component"
	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/home/internal/exp/spa"
)

type appHandler struct {
	app spa.App
}

func (h *appHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) error {
	if err := httputil.AllowMethods(req, http.MethodGet, http.MethodHead); err != nil {
		return err
	}
	var buf bytes.Buffer
	_, err := h.app.ServePage(req.Context(), &buf, req.URL)
	if _, ok := spa.IsOutOfScope(err); ok {
		return fmt.Errorf("internal error: app returned OutOfScopeError on backend: %v", err)
	} else if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if req.Method == http.MethodHead {
		return nil
	}
	err = appHTML.Execute(w, struct {
		AnalyticsHTML template.HTML
		RedLogo       bool
	}{analyticsHTML, component.RedLogo})
	if err != nil {
		return err
	}
	_, err = io.Copy(w, &buf)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, `</body></html>`)
	return err
}

var appHTML = template.Must(template.New("").Parse(`<!DOCTYPE html>
<html lang="en">
	<head>
{{.AnalyticsHTML}}		<link href="/icon.png" rel="icon" type="image/png">
		<meta name="viewport" content="width=device-width">
		<link href="/assets/fonts/fonts.css" rel="stylesheet" type="text/css">
		<style type="text/css">
			body {
				margin: 20px;
				font-family: Go;
				font-size: 14px;
				line-height: initial;
				color: rgb(35, 35, 35);
			}
		</style>

		<script>var RedLogo = {{.RedLogo}};</script>
		<script src="/assets/wasm_exec_go114.js"></script>
		<script>
			if (!WebAssembly.instantiateStreaming) { // polyfill for Safari :/
				WebAssembly.instantiateStreaming = async (resp, importObject) => {
					const source = await (await resp).arrayBuffer();
					return await WebAssembly.instantiate(source, importObject);
				};
			}
			const go = new Go();
			const resp = fetch("/assets/spa.wasm").then((resp) => {
				if (!resp.ok) {
					resp.text().then((body) => {
						document.body.innerHTML = "<pre>" + body + "</pre>";
					});
					throw new Error("did not get acceptable status code: " + resp.status);
				}
				return resp;
			});
			WebAssembly.instantiateStreaming(resp, go.importObject).then((result) => {
				go.run(result.instance);
			}).catch((error) => { document.body.textContent = error; });
		</script>
	</head>
	<body>`))
