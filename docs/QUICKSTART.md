# Quick Start Guide

## Installation

```bash
git clone https://github.com/Al3x-Myku/FlyDB.git
cd FlyDB
```

## Running the Examples

### Main Demo

```bash
go run cmd/example/main.go
```

This demonstrates:
- Database initialization
- Document insertion
- Commit operations
- Querying from memtable and disk
- TOON escaping
- Database restart and persistence

### Simple Todo Example

```bash
go run examples/simple/main.go
```

A minimal example showing basic CRUD operations.

### Batch Insert Benchmark

```bash
go run examples/batch/main.go
```

Demonstrates batch insertion of 1,000 documents.

## Running Tests

```bash
# Run all tests
go test ./pkg/...

# Run tests with verbose output
go test -v ./pkg/...

# Run specific package tests
go test ./pkg/toon
go test ./pkg/db
```

## Basic Usage

```go
package main

import (
    "fmt"
    "log"
    "github.com/Al3x-Myku/FlyDB/pkg/db"
)

func main() {
    // Create database
    database, err := db.NewDB("./my-data")
    if err != nil {
        log.Fatal(err)
    }
    defer database.Close()
    
    // Get collection
    users, _ := database.GetCollection("users")
    
    // Insert document
    doc := db.Document{
        "id":   "1",
        "name": "Alice",
        "age":  30,
    }
    users.Insert(doc)
    
    // Commit to disk
    users.Commit()
    
    // Query
    found, _ := users.FindByID("1")
    fmt.Println(found)
}
```

## Project Structure

```
FlyDB/
├── pkg/
│   ├── db/              # Database engine
│   └── toon/            # TOON serialization
├── cmd/
│   └── example/         # Main demo application
├── examples/
│   ├── simple/          # Simple usage
│   └── batch/           # Batch operations
├── docs/
│   ├── ARCHITECTURE.md  # Architecture deep dive
│   └── TOON_SPEC.md     # TOON format specification
├── README.md
├── LICENSE
└── go.mod
```

## Key Concepts

### Memtable

The in-memory write buffer. All `Insert()` calls add documents here.

```go
users.Insert(doc) // Added to memtable (volatile)
```

### Commit

Flushes the memtable to disk as a TOON block.

```go
users.Commit() // Writes memtable to disk, updates index
```

### Index

In-memory map of document IDs to disk block locations.

```
ID "1" → BlockInfo{offset: 0, length: 123}
ID "2" → BlockInfo{offset: 0, length: 123}  // Same block
ID "3" → BlockInfo{offset: 123, length: 87} // Different block
```

### TOON Block

On-disk representation of documents:

```
users[2]{id,name,age}:
1,Alice,30
2,Bob,25
```

## Performance Tips

1. **Batch your commits**: Insert many documents, then commit once
   ```go
   for i := 0; i < 1000; i++ {
       users.Insert(doc)
   }
   users.Commit() // One commit for all 1,000
   ```

2. **Optimal batch size**: 100-1,000 documents per commit

3. **Read performance**: Recently inserted documents (in memtable) are fastest to query

## Common Patterns

### Insert and Query

```go
users.Insert(db.Document{"id": "1", "name": "Alice"})
users.Commit()

found, err := users.FindByID("1")
if err == db.ErrNotFound {
    fmt.Println("Not found")
}
```

### Update (Insert New Version)

```go
// Original
users.Insert(db.Document{"id": "1", "name": "Alice", "version": 1})
users.Commit()

// Update (same ID)
users.Insert(db.Document{"id": "1", "name": "Alice", "version": 2})
users.Commit()

// Latest version is returned
found, _ := users.FindByID("1")
// found["version"] == 2
```

### Multiple Collections

```go
users, _ := db.GetCollection("users")
products, _ := db.GetCollection("products")

users.Insert(userDoc)
products.Insert(productDoc)

users.Commit()
products.Commit()
```

### Persistence

```go
// Session 1
db1, _ := db.NewDB("./data")
users1, _ := db1.GetCollection("users")
users1.Insert(doc)
users1.Commit() // Write to disk
db1.Close()

// Session 2 (after restart)
db2, _ := db.NewDB("./data")
users2, _ := db2.GetCollection("users") // Index auto-loaded
found, _ := users2.FindByID("1")        // Data is there!
```

## Troubleshooting

### "document not found" after restart

Make sure you called `Commit()` before closing the database. Uncommitted documents in the memtable are lost.

```go
users.Insert(doc)
users.Commit() // ← Don't forget this!
db.Close()
```

### Memory usage growing

The in-memory index grows with the number of committed documents. Consider:
- Committing in larger batches (reduces index size)
- Implementing compaction (future feature)

### Slow reads

- Check if you're reading from large blocks (many documents in one commit)
- Consider committing more frequently to create smaller blocks

## Next Steps

- Read [ARCHITECTURE.md](ARCHITECTURE.md) for design deep dive
- Read [TOON_SPEC.md](TOON_SPEC.md) for format details
- Check out the [examples](../examples/) directory
- Run the tests to see usage patterns

## Getting Help

- Open an issue on GitHub
- Read the source code (it's well-documented!)
- Check the examples directory

---

**Happy coding with FlyDB!** 🚀
