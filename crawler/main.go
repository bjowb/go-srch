package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/net/html"
)

type SearchDocument struct {
	URL       string
	Domain    string
	Title     string
	Content   string
	Depth     int
	Timestamp int64
}

type CrawlNode struct {
	URL   string
	Depth int
}

//-----------------------DATABASE FUNCTIONS-----------------------------

func initDB(filepath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", filepath)
	if err != nil {
		return nil, err
	}

	createTableSQL := `
	CREATE VIRTUAL TABLE IF NOT EXISTS search_index USING fts5(
		url UNINDEXED,
		domain,
		title,
		content,
		depth UNINDEXED,
		timestamp UNINDEXED
	);
	`

	_, err1 := db.Exec(createTableSQL)
	return db, err1
}

func saveDocument(db *sql.DB, doc SearchDocument) error {
	query := `INSERT INTO search_index (url,domain,title,content,depth,timestamp) VALUES (?,?,?,?,?,?)`
	_, err := db.Exec(query, doc.URL, doc.Domain, doc.Title, doc.Content, doc.Depth, doc.Timestamp)
	return err
}

//-------------------------PARSING FUNCTIONS---------------------------

func extractText(node *html.Node) string {
	if node.Type == html.TextNode {
		cleanText := strings.TrimSpace(node.Data)
		if cleanText != "" {
			return cleanText + " "
		}
		return ""
	}

	if node.Type == html.ElementNode {
		if node.Data == "script" || node.Data == "style" || node.Data == "noscript" {
			return ""
		}
	}

	var ans strings.Builder
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		ans.WriteString(extractText(c))
	}

	return ans.String()
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
		// Standard Wiki/Forum Noise
		"action=edit", "action=history", "action=info", "oldid=", "diff=",
		"Special:", "User:", "User_talk:", "Talk:", "Category:", "Help:",

		// Translations to avoid indexing duplicates
		"(Espa", "(Fran", "(Portugu", "(Magyar)", "(Hrvatski)", "(Italiano)",
		"(Svenska)", "(Suomi)", "(Polski)", "(%D", "(%E",

		// --- NEW: CODEFORCES TRAPS ---
		"/profile/", "/status/", "/submission/", "/standings/",
		"/contest/", "/gym/", "enter?back=", "?locale=",

		// --- NEW: GEEKSFORGEEKS TRAPS ---
		"/courses/", "/jobs/", "/events/", "/premium/",
		"login", "register", "/payment/",

		// --- NEW: USACO & LEETCODE TRAPS ---
		"sign-in", "/discuss/", "/submissions/",
	}

	for _, bad := range badPaths {
		if strings.Contains(link, bad) {
			return true // It contains garbage, throw it away
		}
	}

	if strings.Contains(link, "codeforces.com") && !strings.Contains(link, "/blog/entry") {
		return true
	}
	return false // It looks like a clean, useful article
}

// -----------------------------------------ENGINE-----------------------------------
// go concurrent function
func worker(id int, jobs <-chan CrawlNode, results chan<- []CrawlNode, allowed map[string]bool, db *sql.DB) {
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

		pageText := extractText(doc)
		if len(pageText) > 100 {
			docRecord := SearchDocument{
				URL:       node.URL,
				Domain:    baseURL.Hostname(),
				Title:     "Unknown",
				Content:   pageText,
				Depth:     node.Depth,
				Timestamp: time.Now().Unix(),
			}

			err = saveDocument(db, docRecord)
			if err != nil {
				log.Println("Database error : ", err)
			} else {
				fmt.Printf("[Worker %d] SAved %d chars from %s\n", id, len(pageText), node.URL)
			}
		}

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

func main() {

	//-------------INITIALIZE DATABASE----------
	db, err := initDB("./search.db")
	if err != nil {
		log.Fatal("FAiled to initialize db :", err)
	}

	defer db.Close()

	var initialSeeds []string
	//initialSeeds = append(initialSeeds, "https://cp-algorithms.com/")
	//initialSeeds = append(initialSeeds, "https://usaco.guide/")
	//initialSeeds = append(initialSeeds, "https://www.geeksforgeeks.org/fundamentals-of-algorithms/")
	initialSeeds = append(initialSeeds, "https://codeforces.com/blog/entry/91363") // Famous tutorial hub
	//initialSeeds = append(initialSeeds, "https://walkccc.me/LeetCode/")
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
		go worker(i, jobs, results, allowedDomains, db)
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
