package main

import (
	"html/template"
	"io"
	"net/http"

	"github.com/shurcooL/home/component"
	"github.com/shurcooL/home/exp/vec"
	"github.com/shurcooL/home/exp/vec/attr"
	"github.com/shurcooL/home/exp/vec/elem"
	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/octiconssvg"
	"github.com/shurcooL/users"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var aboutHTML = template.Must(template.New("").Parse(`<html>
	<head>
		<title>Dmitri Shuralyov - About</title>
		<link href="/icon.png" rel="icon" type="image/png">
		<meta name="viewport" content="width=device-width">
		<link href="/assets/fonts/fonts.css" rel="stylesheet" type="text/css">
		<link href="//maxcdn.bootstrapcdn.com/font-awesome/4.7.0/css/font-awesome.min.css" rel="stylesheet">
		<link href="/assets/about/style.css" rel="stylesheet" type="text/css">
		{{if .Production}}` + googleAnalytics + `{{end}}
	</head>
	<body>
		<div style="max-width: 800px; margin: 0 auto 100px auto;">`))

func initAbout(notifications notifications.Service, users users.Service) {
	aboutHandler := cookieAuth{httputil.ErrorHandler(users, func(w http.ResponseWriter, req *http.Request) error {
		if req.Method != "GET" {
			return httperror.Method{Allowed: []string{"GET"}}
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := struct{ Production bool }{*productionFlag}
		err := aboutHTML.Execute(w, data)
		if err != nil {
			return err
		}

		authenticatedUser, err := users.GetAuthenticated(req.Context())
		if err != nil {
			return err
		}
		var nc uint64
		if authenticatedUser.ID != 0 {
			nc, err = notifications.Count(req.Context(), nil)
			if err != nil {
				return err
			}
		}
		returnURL := req.RequestURI

		// Render the header.
		header := component.Header{
			CurrentUser:       authenticatedUser,
			NotificationCount: nc,
			ReturnURL:         returnURL,
		}
		err = htmlg.RenderComponents(w, header)
		if err != nil {
			return err
		}

		// Render the tabnav.
		err = htmlg.RenderComponents(w, tabnav{
			Tabs: []tab{
				{
					Content:  iconText{Icon: octiconssvg.Person, Text: "Overview"},
					URL:      "/about",
					Selected: "/about" == req.URL.Path,
				},
				{
					Content:  iconText{Icon: octiconssvg.DeviceDesktop, Text: "Setup"},
					URL:      "/about/setup",
					Selected: "/about/setup" == req.URL.Path,
				},
			},
		})
		if err != nil {
			return err
		}

		// Render content.
		switch req.URL.Path {
		case "/about":
			shurcool, err := users.Get(req.Context(), shurcool)
			if err != nil {
				return err
			}

			err = vec.RenderHTML(w,
				elem.Span(
					attr.Style("display: table; margin-left: auto; margin-right: auto;"),
					elem.Img(
						attr.Style("width: 240px; height: 240px; border-radius: 8px; margin-bottom: 8px;"),
						attr.Src(shurcool.AvatarURL),
					),
					elem.Div(
						attr.Style("font-size: 26px; font-weight: 600;"),
						shurcool.Name,
					),
					elem.Div(
						attr.Style("font-size: 20px; font-weight: 300; color: #666;"),
						shurcool.Login,
					),
				),
				elem.Div(
					attr.Style("margin-top: 24px;"),
					elem.P(
						"Dmitri Shuralyov is a software engineer and an avid ",
						elem.A(attr.Title("Someone who uses Go."), attr.Href("https://golang.org"),
							"gopher",
						),
						". He strives to make software more delightful.",
					),
					elem.P(
						"Coming from a game development and graphics/UI background where C++ was used predominantly, ",
						"he discovered and made a full switch to Go ",
						elem.Abbr(attr.Title("2013."), attr.Href("https://golang.org"),
							"four years ago",
						),
						", which lead to increased developer happiness.",
					),
					elem.P(
						"In his spare time, he's mostly working on software development tools and ",
						"exploring experimental ideas. He enjoys contributing to open source, ",
						"fixing issues in existing tools and the ",
						elem.A(attr.Href("https://github.com/golang/go/commits/master?author=shurcooL"),
							"Go project",
						),
						" itself.",
					),
				),
				elem.Div(
					attr.Style("border-top: 1px solid #f0f0f0;"),
					elem.Ul(
						elem.Li(iconLink{
							Text: "github.com/shurcooL",
							URL:  "https://github.com/shurcooL",
							Icon: faIcon("github"),
						}),
						elem.Li(iconLink{
							Text: "twitter.com/shurcooL",
							URL:  "https://twitter.com/shurcooL",
							Icon: faIcon("twitter"),
						}),
					),
				),
			)
			if err != nil {
				return err
			}

		case "/about/setup":
			err = vec.RenderHTML(w,
				elem.Div(
					attr.Style("margin-top: 24px;"),
					elem.Ul(
						elem.Li(iconText{
							Icon: faIcon("laptop"),
							Text: "Apple MacBook Pro (15-inch, Late 2011)",
						}),
						elem.Li(iconText{
							Icon: faIcon("tv"),
							Text: "Dell 3008WFP",
						}),
						elem.Li(iconText{
							Icon: faIcon("keyboard-o"),
							Text: "Apple Magic Keyboard",
						}),
						elem.Li(iconText{
							Icon: faIcon("mouse-pointer"),
							Text: "Logitech G502",
						}),
						elem.Li(iconText{
							Icon: faIcon("square"),
							Text: "SteelSeries QcK",
						}),
						elem.Li(iconText{
							Icon: faIcon("square-o"),
							Text: "Apple Magic Trackpad 2",
						}),
						elem.Li(iconText{
							Icon: faIcon("volume-off"),
							Text: "Bose Companion 2 Series II",
						}),
						elem.Li(iconText{
							Icon: faIcon("headphones"),
							Text: "Bose QuietComfort 25",
						}),
						elem.Li(iconText{
							Icon: faIcon("user"),
							Text: "Fully Jarvis Frame + IKEA LINNMON (150 cm x 75 cm)",
						}),
						elem.Li(iconText{
							Icon: faIcon("user"),
							Text: "Herman Miller Aeron (2016)",
						}),
					),
				),
			)
			if err != nil {
				return err
			}
		}

		_, err = io.WriteString(w, `</div>`)
		if err != nil {
			return err
		}

		_, err = io.WriteString(w, `</body></html>`)
		return err
	})}
	http.Handle("/about", aboutHandler)
	http.Handle("/about/setup", aboutHandler)
}

// faIcon returns a func that creates a Font Awesome icon.
func faIcon(icon string) func() *html.Node {
	return func() *html.Node {
		return &html.Node{
			Type: html.ElementNode, Data: atom.Span.String(),
			Attr: []html.Attribute{{Key: atom.Style.String(), Val: "display: inline-block; width: 20px; color: #666;"}},
			FirstChild: &html.Node{
				Type: html.ElementNode, Data: atom.I.String(),
				Attr: []html.Attribute{{Key: atom.Class.String(), Val: "fa fa-" + icon}},
			},
		}
	}
}
