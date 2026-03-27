package main

import (
	"fmt"
	"golang.org/x/net/html"
	//"io"
	"log"
	"net/http"
	"net/url"
)

// This is the Node type used in DOM created by golang.org/x/net/html
//type Node struct {
//	Type        html.Attribute
//	Data        string
//	Attr        []html.Attribute
//	FirstChild  *Node
//	NextSibling *Node
//}

func resolveURL(href string, baseURL *url.URL) string {
	parsedHref, err := url.Parse(href)
	if err != nil {
		return ""
	}

	absoluteURL := baseURL.ResolveReference(parsedHref)

	return absoluteURL.String()
}

func extractLinks(node *html.Node, base *url.URL) {
	if node.Type == html.ElementNode && node.Data == "a" {
		for _, attr := range node.Attr {
			if attr.Key == "href" {
				s := resolveURL(attr.Val, base)
				fmt.Println(s)
				break
			}
		}
	}

	for c := node.FirstChild; c != nil; c = c.NextSibling {
		extractLinks(c, base)
	}
}

func main() {
	//initial seed
	const s = "https://cppreference.com/"
	baseUrl, err := url.Parse(s)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println(s, " : is the given seed by user")

	//http get request to get html,css,javascript files
	resp, err := http.Get(s)
	if err != nil {
		log.Fatalln(err)
	}

	//priting to debug
	defer resp.Body.Close()
	//body, err := io.ReadAll(resp.Body)
	//if err != nil {
	//	log.Fatalln(err)
	//}
	//log.Println(string(body))

	// convert html file to DOM
	doc, err := html.Parse(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	extractLinks(doc, baseUrl)

}
