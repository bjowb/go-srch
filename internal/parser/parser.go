package parser

import (
	"net/url"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/net/html"
)

//-------------------------PARSING FUNCTIONS---------------------------

func ExtractText(node *html.Node) string {
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
		ans.WriteString(ExtractText(c))
	}

	return ans.String()
}

func ResolveURL(href string, baseURL *url.URL) string {
	parsedHref, err := url.Parse(href)
	if err != nil {
		return ""
	}
	absoluteURL := baseURL.ResolveReference(parsedHref)
	return absoluteURL.String()
}

func ExtractLinks(node *html.Node, base *url.URL) []string {
	var links []string
	var dfs func(*html.Node)
	dfs = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					s := ResolveURL(attr.Val, base)
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

func IsGarbageURL(link string) bool {
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
