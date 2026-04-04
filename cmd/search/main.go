package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ./search \"your search query\"")
		fmt.Println("Note: Use 'go build -tags sqlite_fts5 -o search cmd/search/main.go' first for faster searches.")
		return
	}

	searchTerm := strings.Join(os.Args[1:], " ")
	fmt.Printf("Searching index for given query : %s\n", searchTerm)
	fmt.Println(strings.Repeat("-", 60))

	db, err := sql.Open("sqlite3", "./search.db")
	if err != nil {
		log.Fatal("Failed to open db :", err)
	}

	defer db.Close()

	query := `
	select url,snippet(search_index, 3, '[MATCH]', '[/MATCH]', '...',25) as preview
	from search_index
	where content match ?
	order by rank
	limit 5
	`

	rows, err := db.Query(query, searchTerm)
	if err != nil {
		log.Fatal("Search Failed!!!", err)
	}

	defer rows.Close()

	ans := 0
	for rows.Next() {
		var url, preview string
		err := rows.Scan(&url, &preview)
		if err != nil {
			log.Println("Error reading record :", err)
			continue
		}

		ans++

		preview = strings.ReplaceAll(preview, "[MATCH]", "\x1b[1;31m")
		preview = strings.ReplaceAll(preview, "[/MATCH]", "\x1b[0m")

		fmt.Printf("\x1b[36m%s\x1b[0m\n", url)
		fmt.Printf("%s \n\n", preview)

	}

	if ans == 0 {
		fmt.Println("sed news ....... :(")
	} else {
		fmt.Println(strings.Repeat("-", 60))
		fmt.Printf("Top %d results displayed,\n", ans)
	}
}
