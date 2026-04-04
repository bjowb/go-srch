# go-srch 🚀

A high-performance, concurrent web crawler and search engine specialized for competitive programming resources. This project indexes content from sites like Codeforces, CP-Algorithms, and USACO Guide, providing a fast local search interface using SQLite's FTS5 (Full-Text Search).

## ✨ Features

- **Concurrent Crawler:** Uses Go routines and worker pools for fast, parallel web crawling with domain whitelisting.
- **Codeforces API Sync:** Dedicated tool to sync the latest blog entries directly via the Codeforces API.
- **Fast Local Search:** Powered by SQLite FTS5 for sub-millisecond full-text search across indexed content.
- **Smart Filtering:** Built-in "garbage" filters to skip login pages, submission status, and non-English wiki translations.
- **Terminal UI:** Color-coded search results with matched snippets directly in your CLI.

## 🛠️ Prerequisites

- **Go:** 1.26.1 or higher.
- **C Compiler:** (e.g., `gcc` or `clang`) required for CGO (SQLite driver).

## 🚀 Getting Started

### 1. Installation

Clone the repository and download dependencies:

```bash
git clone https://github.com/bjowb/go-srch.git
cd go-srch
go mod download
```

### 2. Populating the Index

You need to index some content before you can search. Choose one or both methods:

#### A. Run the General Crawler
Crawls hardcoded seeds (Codeforces tutorials, CP-Algorithms, etc.) to a depth of 3:
```bash
go run cmd/crawler/main.go
```

#### B. Sync Recent Codeforces Blogs
Fetches the 50 most recent blog posts via API:
```bash
go run cmd/cfsync/main.go
```

### 3. Searching

Search your local index for algorithms, problems, or tutorials:

```bash
# Direct run
go run cmd/search/main.go "segment tree"

# Recommended: Build for performance
go build -tags sqlite_fts5 -o search cmd/search/main.go
./search "dynamic programming"
```

## 📁 Project Structure

- `cmd/`
    - `crawler/`: The concurrent web crawler engine.
    - `cfsync/`: Codeforces API synchronization tool.
    - `search/`: CLI search interface.
- `internal/`
    - `db/`: SQLite initialization and FTS5 schema management.
    - `parser/`: HTML parsing, text extraction, and URL filtering logic.

## ⚙️ Technical Details

- **Concurrency:** The crawler uses a worker pool pattern with shared channels to balance load across multiple domains.
- **Storage:** Data is stored in a local `search.db` file.
- **FTS5:** Utilizes SQLite's `snippet()` and `rank` functions to provide relevant search results with context.
