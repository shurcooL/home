home
====

[![Build Status](https://travis-ci.org/shurcooL/home.svg?branch=master)](https://travis-ci.org/shurcooL/home) [![GoDoc](https://godoc.org/github.com/shurcooL/home?status.svg)](https://godoc.org/github.com/shurcooL/home)

home is Dmitri Shuralyov's personal website.

Installation
------------

```bash
go get -u github.com/shurcooL/home
```

Directories
-----------

| Path                                                                                              | Synopsis                                                                             |
|---------------------------------------------------------------------------------------------------|--------------------------------------------------------------------------------------|
| [assets](https://godoc.org/github.com/shurcooL/home/assets)                                       | Package assets contains assets for home.                                             |
| [component](https://godoc.org/github.com/shurcooL/home/component)                                 | Package component contains individual components that can render themselves as HTML. |
| [http](https://godoc.org/github.com/shurcooL/home/http)                                           | Package http contains service implementations over HTTP.                             |
| [httphandler](https://godoc.org/github.com/shurcooL/home/httphandler)                             | Package httphandler contains API handlers used by home.                              |
| [httputil](https://godoc.org/github.com/shurcooL/home/httputil)                                   | Package httputil is a custom HTTP framework created specifically for home.           |
| [internal/code](https://godoc.org/github.com/shurcooL/home/internal/code)                         | Package code implements discovery of Go code within a repository store.              |
| [internal/page/blog](https://godoc.org/github.com/shurcooL/home/internal/page/blog)               | Package blog contains functionality for rendering /blog page.                        |
| [internal/page/idiomaticgo](https://godoc.org/github.com/shurcooL/home/internal/page/idiomaticgo) | Package idiomaticgo contains functionality for rendering /idiomatic-go page.         |
| [internal/page/resume](https://godoc.org/github.com/shurcooL/home/internal/page/resume)           | Package resume contains functionality for rendering /resume page.                    |
| [internal/route](https://godoc.org/github.com/shurcooL/home/internal/route)                       | Package route specifies some route paths used by home.                               |
| [presentdata](https://godoc.org/github.com/shurcooL/home/presentdata)                             | Package presentdata contains static data for present format.                         |

License
-------

-	[MIT License](https://opensource.org/licenses/mit-license.php)
