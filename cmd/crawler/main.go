package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/bjowb/go-srch/internal/db"
	"github.com/bjowb/go-srch/internal/parser"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/net/html"
)

type CrawlNode struct {
	URL   string
	Depth int
}

// -----------------------------------------ENGINE-----------------------------------
// go concurrent function
func worker(id int, jobs <-chan CrawlNode, results chan<- []CrawlNode, allowed map[string]bool, dbconn *sql.DB) {
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
		//resp, err := http.Get(node.URL)
		//if err != nil {
		//	log.Println("Failed to fetch : ", node.URL)
		//	results <- []CrawlNode{}
		//	continue
		//}

		//----------------CHROME DISGUISE FOR CODEFORCES----------------
		req, err := http.NewRequest("GET", node.URL, nil)
		if err != nil {
			results <- []CrawlNode{}
			continue
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
		req.Header.Set("Accept-Language", "en-US,en;q=0.5")

		client := &http.Client{
			Timeout: 10 * time.Second,
		}

		resp, err := client.Do(req)
		if err != nil {
			log.Println("Failed to fetch : ", node.URL)
			results <- []CrawlNode{}
			continue
		}

		//Rate Limiter
		time.Sleep(500 * time.Millisecond)
		//Parse html into DOM tree
		doc, err := html.Parse(resp.Body)
		if err != nil {
			log.Println("Failed to parse HTML code for :", node.URL)
			resp.Body.Close()
			results <- []CrawlNode{}
			continue
		}

		resp.Body.Close()

		pageText := parser.ExtractText(doc)
		if len(pageText) > 100 {
			docRecord := db.SearchDocument{
				URL:       node.URL,
				Domain:    baseURL.Hostname(),
				Title:     "Unknown",
				Content:   pageText,
				Depth:     node.Depth,
				Timestamp: time.Now().Unix(),
			}

			err = db.SaveDocument(dbconn, docRecord)
			if err != nil {
				log.Println("Database error : ", err)
			} else {
				fmt.Printf("[Worker %d] SAved %d chars from %s\n", id, len(pageText), node.URL)
			}
		}

		pageLinks := parser.ExtractLinks(doc, baseURL)
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

			if parser.IsGarbageURL(link) {
				continue
			}
			nextNodes = append(nextNodes, CrawlNode{URL: link, Depth: node.Depth + 1})
		}
		results <- nextNodes
	}
}

func main() {

	//-------------INITIALIZE DATABASE----------
	db1, err := db.InitDB("./search.db")
	if err != nil {
		log.Fatal("FAiled to initialize db :", err)
	}

	defer db1.Close()

	var initialSeeds []string
	initialSeeds = append(initialSeeds, "https://cp-algorithms.com/")
	initialSeeds = append(initialSeeds, "https://usaco.guide/")
	initialSeeds = append(initialSeeds, "https://www.geeksforgeeks.org/fundamentals-of-algorithms/")
	//initialSeeds = append(initialSeeds, "https://codeforces.com/blog/entry/91363") // Famous tutorial hub
	initialSeeds = append(initialSeeds, "https://walkccc.me/LeetCode/")
	// The Whitelist
	allowedDomains := map[string]bool{
		"cppreference.com":           true,
		"en.cppreference.com":        true, // Handle subdomains explicitly for now
		"cp-algorithms.com":          true,
		"usaco.guide":                true,
		"www.geeksforgeeks.org":      true,
		"practice.geeksforgeeks.org": true,
		"codeforces.com":             true,
		"walkccc.me":                 true,
	}

	//replaced queues with channels to enable concurrency
	jobs := make(chan CrawlNode)
	results := make(chan []CrawlNode)

	//make 5 go-routines
	for i := 1; i < 6; i++ {
		go worker(i, jobs, results, allowedDomains, db1)
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
