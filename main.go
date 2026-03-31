package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

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

// go concurrent function
func worker(id int, jobs <-chan CrawlNode, results chan<- []CrawlNode, allowed map[string]bool) {
	for node := range jobs {
		fmt.Println("Worker ", id, " Fetching :", node.URL)

		//Parse Base URL
		baseURL, err := url.Parse(node.URL)
		if err != nil {
			log.Println("Invalid URL :", node.URL)
			results <- []CrawlNode{}
			continue
		}

		//Request html code
		resp, err := http.Get(node.URL)
		if err != nil {
			log.Println("Failed to fetch : ", node.URL)
			results <- []CrawlNode{}
			continue
		}

		//Parse html into DOM tree
		doc, err := html.Parse(resp.Body)
		if err != nil {
			log.Println("Failed to parse HTML code for :", node.URL)
			resp.Body.Close()
			results <- []CrawlNode{}
			continue
		}

		resp.Body.Close()

		pageLinks := extractLinks(doc, baseURL)
		var nextNodes []CrawlNode
		for _, link := range pageLinks {
			nextLink, err := url.Parse(link)
			if err != nil {
				log.Println("Error while parsing link :", link)
				continue
			}

			_, ok := allowed[nextLink.Hostname()]
			if !ok {
				continue
			}

			if isGarbageURL(link) {
				continue
			}
			nextNodes = append(nextNodes, CrawlNode{URL: link, Depth: node.Depth + 1})
		}
		results <- nextNodes
	}
}
func isGarbageURL(link string) bool {
	// A list of URL substrings that indicate useless Wiki pages
	badPaths := []string{
		"action=edit",
		"action=history",
		"action=info",
		"oldid=",
		"diff=",
		"Special:",
		"User:",
		"User_talk:",
		"Talk:",
		"Category:",
		"Help:",
		// Optional: Block non-English wiki translations (common ArchWiki pattern)
		"(Espa", "(Fran", "(Portugu", "(Magyar)", "(Hrvatski)", "(Italiano)",
		"(Svenska)", "(Suomi)", "(Polski)", "(%D", // %D blocks most Cyrillic/Arabic URL encodings
	}

	for _, bad := range badPaths {
		if strings.Contains(link, bad) {
			return true // It contains garbage, throw it away
		}
	}
	return false // It looks like a clean, useful article
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

	//replaced queues with channels to enable concurrency
	jobs := make(chan CrawlNode)
	results := make(chan []CrawlNode)

	//make 5 go-routines
	for i := 1; i < 6; i++ {
		go worker(i, jobs, results, allowedDomains)
	}

	//counter for number of active jobs to handle concurrency
	activeJobs := 0

	//make a visisted map
	vis := make(map[string]bool)

	//initial seeds
	for _, seeds := range initialSeeds {
		vis[seeds] = true
		activeJobs++

		go func(s string) {
			jobs <- CrawlNode{URL: s, Depth: 0}
		}(seeds)
	}

	//Traverse the queue
	for activeJobs > 0 {

		//block till  there is filling in results
		nextNodes := <-results
		activeJobs--
		for _, nodes := range nextNodes {
			if !vis[nodes.URL] {
				vis[nodes.URL] = true

				if nodes.Depth < 3 {
					activeJobs++

					go func(n CrawlNode) {
						jobs <- n
					}(nodes)
				}
			}
		}
	}
	fmt.Println("WEB CRAWL COMPLETEEE !!!!!!!!!!!")
}
