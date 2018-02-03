package main

import (
	"context"
	"io"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

func parseHTML(ctx context.Context, url *url.URL, body io.Reader) {
	htmlTokens := html.NewTokenizer(body)

	for {
		tt := htmlTokens.Next()
		switch {
		case tt == html.ErrorToken:
			// End of the document, we're done
			return
		case tt == html.StartTagToken:
			t := htmlTokens.Token()
			// fmt.Printf("tag: %v\n", t)
			// fmt.Printf("type: %v\n", t.Type)
			// fmt.Printf("Data: %v\n", t.Data)
			// fmt.Println("---")
			switch t.Data {
			case "link":
				//fmt.Println("Found an link")
				for _, value := range t.Attr {
					if value.Key == "href" {
						triggerVisit(ctx, url, value.Val)
					}
				}
			case "script":
				for _, value := range t.Attr {
					if value.Key == "src" {
						//fmt.Printf("Found an script! %+v\n", value)
						triggerVisit(ctx, url, value.Val)
					}
				}

			}

		case tt == html.SelfClosingTagToken:
			t := htmlTokens.Token()

			switch t.Data {
			case "img":
				// fmt.Println("Found an image")
				// fmt.Printf("self closing tag: %+v\n", t)
				// fmt.Println("---")

				for _, value := range t.Attr {
					//fmt.Printf("key:%v, value: %v\n", key, value)
					if value.Key == "src" {
						//fmt.Printf("Found an src! %+v\n", value.Val)
						triggerVisit(ctx, url, value.Val)
					}
				}
			}
		}
	}
}

func triggerVisit(ctx context.Context, previousURL *url.URL, newURL string) {
	if strings.HasPrefix(newURL, "http") {
		visit(ctx, parseURL(newURL))
	} else {
		visit(ctx, parseURL(previousURL.Host+"/"+newURL))
	}
}
