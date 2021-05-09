home
====

[![Go Reference](https://pkg.go.dev/badge/github.com/shurcooL/home.svg)](https://pkg.go.dev/github.com/shurcooL/home)

home is Dmitri Shuralyov's personal website.

Installation
------------

```sh
go install github.com/shurcooL/home@latest
```

Directories
-----------

| Path                                                                                               | Synopsis                                                                                                             |
|----------------------------------------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------|
| [assets](https://pkg.go.dev/github.com/shurcooL/home/assets)                                       | Package assets contains assets for home.                                                                             |
| [cmd/githook/pre-receive](https://pkg.go.dev/github.com/shurcooL/home/cmd/githook/pre-receive)     | pre-receive is a pre-receive git hook for use with home's git server.                                                |
| [component](https://pkg.go.dev/github.com/shurcooL/home/component)                                 | Package component contains individual components that can render themselves as HTML.                                 |
| [http](https://pkg.go.dev/github.com/shurcooL/home/http)                                           | Package http contains service implementations over HTTP.                                                             |
| [httphandler](https://pkg.go.dev/github.com/shurcooL/home/httphandler)                             | Package httphandler contains API handlers used by home.                                                              |
| [httputil](https://pkg.go.dev/github.com/shurcooL/home/httputil)                                   | Package httputil is a custom HTTP framework created specifically for home.                                           |
| [indieauth](https://pkg.go.dev/github.com/shurcooL/home/indieauth)                                 | Package indieauth implements building blocks for the IndieAuth specification (https://indieauth.spec.indieweb.org/). |
| [internal/code](https://pkg.go.dev/github.com/shurcooL/home/internal/code)                         | Package code implements a Go code service backed by a repository store.                                              |
| [internal/code/httpclient](https://pkg.go.dev/github.com/shurcooL/home/internal/code/httpclient)   | Package httpclient contains issues.Service implementation over HTTP.                                                 |
| [internal/code/httphandler](https://pkg.go.dev/github.com/shurcooL/home/internal/code/httphandler) | Package httphandler contains an API handler for issues.Service.                                                      |
| [internal/code/httproute](https://pkg.go.dev/github.com/shurcooL/home/internal/code/httproute)     | Package httproute contains route paths for httpclient, httphandler.                                                  |
| [internal/mod](https://pkg.go.dev/github.com/shurcooL/home/internal/mod)                           | Package mod exposes select functionality related to module mechanics.                                                |
| [internal/page/blog](https://pkg.go.dev/github.com/shurcooL/home/internal/page/blog)               | Package blog contains functionality for rendering /blog page.                                                        |
| [internal/page/idiomaticgo](https://pkg.go.dev/github.com/shurcooL/home/internal/page/idiomaticgo) | Package idiomaticgo contains functionality for rendering /idiomatic-go page.                                         |
| [internal/page/resume](https://pkg.go.dev/github.com/shurcooL/home/internal/page/resume)           | Package resume contains functionality for rendering /resume page.                                                    |
| [internal/route](https://pkg.go.dev/github.com/shurcooL/home/internal/route)                       | Package route specifies some route paths used by home.                                                               |
| [presentdata](https://pkg.go.dev/github.com/shurcooL/home/presentdata)                             | Package presentdata contains static data for present format.                                                         |

License
-------

-	[MIT License](LICENSE)
