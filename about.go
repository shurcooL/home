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
	"github.com/shurcooL/home/internal/exp/service/notification"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/octicon"
	"github.com/shurcooL/users"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var aboutHTML = template.Must(template.New("").Parse(`<html>
	<head>
{{.AnalyticsHTML}}		<title>Dmitri Shuralyov - About</title>
		<link href="/icon.svg" rel="icon" type="image/svg+xml">
		<meta name="viewport" content="width=device-width">
		<link href="/assets/fonts/fonts.css" rel="stylesheet" type="text/css">
		<link href="//maxcdn.bootstrapcdn.com/font-awesome/4.7.0/css/font-awesome.min.css" rel="stylesheet">
		<link href="/assets/about/style.css" rel="stylesheet" type="text/css">
	</head>
	<body>
		<div style="max-width: 800px; margin: 0 auto 100px auto;">`))

func initAbout(notification notification.Service, users users.Service) {
	aboutHandler := cookieAuth{httputil.ErrorHandler(users, func(w http.ResponseWriter, req *http.Request) error {
		if req.Method != "GET" {
			return httperror.Method{Allowed: []string{"GET"}}
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := struct{ AnalyticsHTML template.HTML }{analyticsHTML}
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
			nc, err = notification.CountNotifications(req.Context())
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
		err = htmlg.RenderComponents(w, component.TabNav{
			Tabs: []component.Tab{
				{
					Content:  iconText{Icon: octicon.Person, Text: "Overview"},
					URL:      "/about",
					Selected: "/about" == req.URL.Path,
				},
				{
					Content:  iconText{Icon: octicon.DeviceDesktop, Text: "Setup"},
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
			dmitshur, err := users.Get(req.Context(), dmitshur)
			if err != nil {
				return err
			}

			err = vec.RenderHTML(w,
				elem.Span(
					attr.Style("display: table; margin-left: auto; margin-right: auto;"),
					elem.Img(
						attr.Style("width: 240px; height: 240px; border-radius: 8px; margin-bottom: 8px;"),
						attr.Src(dmitshur.AvatarURL),
					),
					elem.Div(
						attr.Style("font-size: 26px; font-weight: 600;"),
						dmitshur.Name,
					),
					elem.Div(
						attr.Style("font-size: 20px; font-weight: 300; color: #666;"),
						dmitshur.Login,
					),
					elem.Ul(
						attr.Style("margin-top: 16px; padding-top: 16px; border-top: 1px solid #f0f0f0;"),
						elem.Li(iconLink{
							Text: dmitshur.Email,
							URL:  "mailto:" + dmitshur.Email,
							Icon: faIcon("envelope-o"),
						}),
						elem.Li(iconLink{
							Text: "github.com/dmitshur",
							URL:  "https://github.com/dmitshur",
							Icon: faIcon("github"),
						}),
						elem.Li(iconLink{
							Text: "twitter.com/dmitshur",
							URL:  "https://twitter.com/dmitshur",
							Icon: faIcon("twitter"),
						}),
					),
				),
				elem.Div(
					elem.P(
						dmitshur.Name+" is a software engineer and an avid ",
						elem.A(attr.Title("Someone who uses Go."), attr.Href("https://golang.org"),
							"gopher",
						),
						". He strives to make software more delightful.",
					),
					elem.P(
						"Coming from a game development and graphics/UI background where C++ was used predominantly, ",
						"he discovered and made a full switch to Go ",
						elem.Abbr(attr.Title("2013."), "eight years ago"),
						", which lead to increased developer happiness.",
					),
					elem.P(
						"In his spare time, he's mostly working on software development tools and ",
						"exploring experimental ideas. He enjoys contributing to open source, ",
						"fixing issues in existing tools and the ",
						elem.A(attr.Href("https://github.com/golang/go/commits/master?author=dmitshur"),
							"Go project",
						),
						" itself.",
					),
				),
			)
			if err != nil {
				return err
			}

		case "/about/setup":
			err = vec.RenderHTML(w,
				elem.Div(
					elem.H3("Home"),
					elem.Ul(
						elem.Li(iconText{
							Icon: faIcon("laptop"),
							Text: "Apple MacBook Air (M1, 2020)",
						}),
						elem.Li(iconText{
							Icon: faIcon("tv"),
							Text: "Apple Pro Display XDR, Apple VESA Mount Adapter",
						}),
						elem.Li(iconText{
							Icon: faIcon("keyboard-o"),
							Text: "Apple Magic Keyboard",
						}),
						elem.Li(iconText{
							Icon: faIcon("mouse-pointer"),
							Text: "Logitech G Pro Gaming Mouse",
						}),
						elem.Li(iconText{
							Icon: faIcon("square-o"),
							Text: "Apple Magic Trackpad 2",
						}),
						elem.Li(iconText{
							Icon: faIcon("headphones"),
							Text: "Apple AirPods Max",
						}),
						elem.Li(iconText{
							Icon: faIcon("square"),
							Text: "Rain Design mStand",
						}),
						elem.Li(iconText{
							Icon: faIcon("user"),
							Text: "IKEA LINNMON (150 cm x 75 cm), Fully Jarvis Frame",
						}),
						elem.Li(iconText{
							Icon: faIcon("user"),
							Text: "Herman Miller Aeron (2016)",
						}),
					),
					elem.H3("Mobile"),
					elem.Ul(
						elem.Li(iconText{
							Icon: faIcon("mobile"),
							Text: "Apple iPhone 12 Pro Max",
						}),
						elem.Li(iconText{
							Icon: faIcon("headphones"),
							Text: "Apple AirPods",
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
