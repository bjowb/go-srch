package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/bjowb/go-srch/internal/db"
	"github.com/bjowb/go-srch/internal/parser"
	"golang.org/x/net/html"
)

type CFRecentActions struct {
	Result []struct {
		BlogEntry struct {
			ID    int    `json:"id"`
			Title string `json:"title"`
		} `json:"blogEntry"`
	} `json:"result"`
}

type CFBlogView struct {
	Result struct {
		Content string `json:"content"`
	} `json:"result"`
}

func main() {
	fmt.Println("🚀 Starting Codeforces API Sync...")

	// 1. Initialize our shared Database Package
	dbConn, err := db.InitDB("./search.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer dbConn.Close()

	fmt.Println("📡 Fetching recent blogs from Codeforces API...")
	resp, err := http.Get("https://codeforces.com/api/recentActions?maxCount=50")
	if err != nil {
		log.Fatal("API Request failed:", err)
	}
	defer resp.Body.Close()

	var recent CFRecentActions
	if err := json.NewDecoder(resp.Body).Decode(&recent); err != nil {
		log.Fatal("Failed to parse JSON:", err)
	}

	for _, action := range recent.Result {
		if action.BlogEntry.ID == 0 {
			continue // Not a blog action
		}

		blogID := action.BlogEntry.ID
		blogTitle := action.BlogEntry.Title // We actually get real titles from the API!
		blogURL := fmt.Sprintf("https://codeforces.com/blog/entry/%d", blogID)

		fmt.Printf("⬇️ Downloading Blog %d: %s\n", blogID, blogTitle)

		apiURL := fmt.Sprintf("https://codeforces.com/api/blogEntry.view?blogEntryId=%d", blogID)
		blogResp, err := http.Get(apiURL)
		if err != nil {
			log.Println("Failed to fetch blog:", err)
			continue
		}

		var blogData CFBlogView
		if err := json.NewDecoder(blogResp.Body).Decode(&blogData); err != nil {
			blogResp.Body.Close()
			continue
		}
		blogResp.Body.Close()

		// Parse the HTML string provided by the API
		doc, err := html.Parse(strings.NewReader(blogData.Result.Content))
		if err != nil {
			continue
		}

		// 2. Use our shared Parser Package
		pageText := parser.ExtractText(doc)

		if len(pageText) > 100 {
			docRecord := db.SearchDocument{
				URL:       blogURL,
				Domain:    "codeforces.com",
				Title:     blogTitle,
				Content:   pageText,
				Depth:     0,
				Timestamp: time.Now().Unix(),
			}

			// 3. Save using our shared DB Package
			err = db.SaveDocument(dbConn, docRecord)
			if err != nil {
				log.Println("DB Insert Error:", err)
			} else {
				fmt.Printf("✅ Saved %d characters to offline index.\n", len(pageText))
			}
		}

		// Rate limit to be polite to the API
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println("🏁 Codeforces API Sync Complete!")
}
