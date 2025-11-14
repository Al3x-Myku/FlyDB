# FlyDB

> A lightweight, document-oriented NoSQL database built in Go with a novel TOON (Text Object Notation) serialization format.

[![Go Version](https://img.shields.io/badge/Go-1.22.2-blue.svg)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

## 🚀 Features

- **Document-Oriented**: Store and query JSON-like documents with ease
- **TOON Format**: Compact, schema-batched serialization that eliminates field name redundancy
- **LSM-Tree Architecture**: Memtable-on-TOON design for efficient writes and reads
- **Type Inference**: Automatic detection of integers, floats, booleans, and strings
- **Thread-Safe**: Concurrent reads and writes with fine-grained locking
- **Interactive Shell**: Built-in shell with query language and compression support
- **Zero Core Dependencies**: Pure Go implementation for database core
- **Human-Readable**: Data files are plain text and can be inspected/edited manually
- **Compression**: Optional gzip compression for exports and data transfer

## 📦 Installation

```bash
git clone https://github.com/Al3x-Myku/FlyDB.git
cd FlyDB
go mod download
```

## 🏃 Quick Start

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/Al3x-Myku/FlyDB/pkg/db"
)

func main() {
    // Create database
    database, err := db.NewDB("./data")
    if err != nil {
        log.Fatal(err)
    }
    defer database.Close()
    
    // Get collection
    users, _ := database.GetCollection("users")
    
    // Insert document
    user := db.Document{
        "id":    "1",
        "name":  "Alice",
        "age":   30,
        "email": "alice@example.com",
    }
    users.Insert(user)
    
    // Commit to disk
    users.Commit()
    
    // Find by ID
    found, _ := users.FindByID("1")
    fmt.Printf("Found: %v\n", found)
}
```

## 📚 What is TOON?

**TOON (Text Object Notation)** is a compact serialization format that stores collections of documents with shared schemas.

### JSON vs TOON

**JSON** (84 bytes):
```json
[
  {"id": "1", "name": "Alice", "age": 30},
  {"id": "2", "name": "Bob", "age": 25}
]
```

**TOON** (46 bytes - 45% smaller):
```
users[2]{id,name,age}:
1,Alice,30
2,Bob,25
```

TOON eliminates field name duplication, making it ideal for storing many similar documents.

## 🏗️ Architecture

FlyDB implements a simplified LSM-tree (Log-Structured Merge-Tree) architecture:

```
┌─────────────┐
│  Insert()   │
└──────┬──────┘
       ▼
┌─────────────────┐
│    Memtable     │  ← In-memory buffer
│   (Documents)   │
└──────┬──────────┘
       │ Commit()
       ▼
┌─────────────────┐
│   TOON Block    │  ← Serialized format
└──────┬──────────┘
       ▼
┌─────────────────┐
│   Disk File     │  ← Append-only .toon file
│   users.toon    │
└─────────────────┘
       │
       ▼
┌─────────────────┐
│  In-Memory Index│  ← BlockInfo map for fast lookups
│  id → location  │
└─────────────────┘
```

### Key Components

- **Memtable**: In-memory write buffer for new documents
- **TOON Blocks**: Compressed on-disk representation
- **Index**: Maps document IDs to block locations for O(1) lookups
- **Collection**: Manages a single `.toon` file with its memtable and index

## 📖 Documentation

- **[Quick Start Guide](docs/QUICKSTART.md)** - Get up and running in 5 minutes
- **[Shell Guide](docs/SHELL_GUIDE.md)** - Interactive shell with query language and compression
- **[Architecture Deep Dive](docs/ARCHITECTURE.md)** - Detailed system design
- **[TOON Specification](docs/TOON_SPEC.md)** - Format specification and examples
- **[Contributing Guide](CONTRIBUTING.md)** - How to contribute to FlyDB

## 🧪 Examples

### Simple Todo App

```bash
go run examples/simple/main.go
```

A minimal example demonstrating basic CRUD operations with a todo list.

### Batch Insert Benchmark

```bash
go run examples/batch/main.go
```

Demonstrates batch insertion of 1,000 documents and query performance.

### Full Demo

```bash
go run cmd/example/main.go
```

Comprehensive demonstration of all FlyDB features including:
- Document insertion and querying
- Memtable vs disk reads
- TOON escaping (commas, newlines, backslashes)
- Database restart and persistence

## 🧰 API Reference

### FlyDB Shell

Run the interactive shell:

```bash
go run cmd/flydb/shell.go
# or after building:
./flydb
```

**Shell Commands:**

```
Database Commands:
  show collections       - List all collections
  show stats            - Show database statistics
  use <collection>      - Switch to a collection

Collection Commands:
  insert <json>         - Insert a document
  find <id>            - Find a document by ID
  query <expr>         - Query documents (e.g., query age > 30)
  commit               - Commit pending changes to disk
  count                - Show document counts
  export <file>        - Export collection to JSON

Advanced:
  compress on|off      - Enable/disable gzip compression

Query Language:
  field = value        - Exact match
  field > value        - Greater than
  field < value        - Less than
  field >= value       - Greater or equal
  field <= value       - Less or equal
  field != value       - Not equal
```

### Database Operations

```go
// Create database
db, err := db.NewDB("./data")

// Get or create collection
collection, err := db.GetCollection("users")

// List all collections
collections, err := db.ListCollections()

// Get statistics
stats := db.GetStats()

// Close database
db.Close()
```

### Collection Operations

```go
// Insert document (to memtable)
id, err := collection.Insert(db.Document{
    "id": "1",
    "name": "Alice",
})

// Commit memtable to disk
err := collection.Commit()

// Find document by ID
doc, err := collection.FindByID("1")

// Get collection stats
size := collection.Size()        // Memtable size
indexSize := collection.IndexSize() // Indexed documents
```

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./pkg/...

# Run specific package tests
go test ./pkg/db
go test ./pkg/toon
```

All tests pass ✅ (12/12)

## 🎯 Use Cases

FlyDB is ideal for:

- **Embedded databases** in Go applications
- **Local data storage** for desktop/CLI tools
- **Prototyping** and learning database internals
- **Testing** without external database dependencies
- **Log aggregation** with append-only writes
- **Time-series data** with simple key-value lookups

## ⚡ Performance Characteristics

- **Writes**: O(1) - Append to memtable
- **Commits**: O(n) - Serialize and append block to disk
- **Reads**: O(1) - Index lookup + single disk read
- **Memory**: O(m + i) - Memtable size + index size
- **Disk**: Append-only, no fragmentation

## 🛣️ Roadmap

- [x] **Query language** for complex queries (basic implementation in shell)
- [x] **Compression** (gzip support in shell)
- [ ] Secondary indexes for non-ID fields
- [ ] Compaction to reclaim space from old versions
- [ ] Background memtable flush
- [ ] Write-ahead log (WAL) for crash recovery
- [ ] HTTP API server
- [ ] Replication and clustering

## 🤝 Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Inspired by LSM-tree databases like RocksDB and LevelDB
- TOON format influenced by CSV and Protocol Buffers
- Built as a learning project to understand database internals

## 📧 Contact

**Alex Myku** - [@Al3x-Myku](https://github.com/Al3x-Myku)

Project Link: [https://github.com/Al3x-Myku/FlyDB](https://github.com/Al3x-Myku/FlyDB)

---

<p align="center">
  Made with ❤️ and Go
</p>
