package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"

	"golang.org/x/net/html"
)

type CrawlNode struct {
	URL   string
	Depth int
}

func resolveURL(href string, baseURL *url.URL) string {
	parsedHref, err := url.Parse(href)
	if err != nil {
		return ""
	}
	absoluteURL := baseURL.ResolveReference(parsedHref)
	return absoluteURL.String()
}

func extractLinks(node *html.Node, base *url.URL) []string {
	var links []string
	var dfs func(*html.Node)
	dfs = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					s := resolveURL(attr.Val, base)
					if s != "" {
						links = append(links, s)
					}
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			dfs(c)
		}
	}
	dfs(node)
	return links
}

func main() {

	var initialSeeds []string
	initialSeeds = append(initialSeeds, "https://cppreference.com/")
	initialSeeds = append(initialSeeds, "https://wiki.archlinux.org/")

	// The Whitelist
	allowedDomains := map[string]bool{
		"cppreference.com":    true,
		"en.cppreference.com": true, // Handle subdomains explicitly for now
		"wiki.archlinux.org":  true,
	}

	//make a queue of string
	var queue []CrawlNode

	//make a visisted map
	vis := make(map[string]bool)

	//initial seeds
	for _, seeds := range initialSeeds {
		vis[seeds] = true
		queue = append(queue, CrawlNode{seeds, 0})
	}

	//Traverse the queue
	for len(queue) > 0 {

		//Deque elements
		s := queue[0].URL
		deep := queue[0].Depth
		fmt.Println("Extracted Element from queue : ", s)
		queue = queue[1:]

		// if depth >= 3 then skip the link
		if deep == 3 {
			continue
		}

		//Parse Base URL
		baseURL, err := url.Parse(s)
		if err != nil {
			log.Println("Invalid URL :", s)
			continue
		}
		fmt.Println(s, " : is the given seed by user")

		//Request html code
		resp, err := http.Get(s)
		if err != nil {
			log.Println("Failed to fetch : ", s)
			continue
		}

		//Parse html into DOM tree
		doc, err := html.Parse(resp.Body)
		if err != nil {
			log.Println("Failed to parse HTML code for :", s)
			resp.Body.Close()
			continue
		}

		resp.Body.Close()

		pageLinks := extractLinks(doc, baseURL)

		for _, link := range pageLinks {
			nextLink, err := url.Parse(link)
			if err != nil {
				log.Println("Error while parsing link :", link)
				continue
			}

			_, ok := allowedDomains[nextLink.Hostname()]
			if !ok {
				continue
			}
			if !vis[link] {
				vis[link] = true
				queue = append(queue, CrawlNode{link, deep + 1})
			}
		}
	}
}
