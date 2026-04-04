package db

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type SearchDocument struct {
	URL       string
	Domain    string
	Title     string
	Content   string
	Depth     int
	Timestamp int64
}

//-----------------------DATABASE FUNCTIONS-----------------------------

func InitDB(filepath string) (*sql.DB, error) {
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

func SaveDocument(db *sql.DB, doc SearchDocument) error {
	query := `INSERT INTO search_index (url,domain,title,content,depth,timestamp) VALUES (?,?,?,?,?,?)`
	_, err := db.Exec(query, doc.URL, doc.Domain, doc.Title, doc.Content, doc.Depth, doc.Timestamp)
	return err
}
