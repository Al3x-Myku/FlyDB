# FlyDB Shell Guide

The FlyDB interactive shell provides a command-line interface for interacting with your database.

## Starting the Shell

```bash
# Use default data directory (./flydb-shell-data)
go run cmd/flydb/shell.go

# Or specify a custom directory
go run cmd/flydb/shell.go ./my-data

# Or use the compiled binary
./flydb ./my-data
```

## Quick Start Example

```
FlyDB Shell v1.0
Type 'help' for available commands

flydb> use users
Switched to collection 'users'

flydb:users> insert {"id":"1","name":"Alice","age":30,"role":"admin"}
Inserted document with ID: 1 (not yet committed)

flydb:users> insert {"id":"2","name":"Bob","age":25,"role":"user"}
Inserted document with ID: 2 (not yet committed)

flydb:users> commit
Committed 2 document(s) to disk

flydb:users> find 1
{
  "age": 30,
  "id": "1",
  "name": "Alice",
  "role": "admin"
}

flydb:users> count
Memtable (uncommitted): 0
Indexed (on disk):      2
Total:                  2

flydb:users> exit
Bye!
```

## Command Reference

### Database Commands

#### `show collections`
List all available collections:
```
flydb> show collections
Collections:
  - users
  - products
  - orders
```

#### `show stats`
Display database statistics:
```
flydb> show stats
Database: ./flydb-shell-data
Collections: 2
Compression: OFF

Collection Details:
  users:
    Memtable:  0 documents
    Indexed:   10 documents
    File:      ./flydb-shell-data/users.toon
```

#### `use <collection>`
Switch to a collection (creates it if it doesn't exist):
```
flydb> use products
Switched to collection 'products'
flydb:products>
```

### Collection Commands

#### `insert <json>`
Insert a document into the current collection:
```
flydb:users> insert {"id":"3","name":"Charlie","age":35}
Inserted document with ID: 3 (not yet committed)
```

**Note:** Documents must include an `"id"` field.

#### `find <id>`
Retrieve a document by its ID:
```
flydb:users> find 3
{
  "age": 35,
  "id": "3",
  "name": "Charlie"
}
```

#### `query <expression>`
Query documents using comparison operators:
```
flydb:users> query age > 30
Searching for documents where age > 30...
```

**Supported Operators:**
- `=` - Equal to
- `!=` - Not equal to
- `>` - Greater than
- `<` - Less than
- `>=` - Greater than or equal
- `<=` - Less than or equal

**Examples:**
```
query name = Alice
query age >= 25
query role != admin
query price < 100
```

**Note:** Current implementation is a demo. Full query support requires document iteration.

#### `commit`
Write pending documents from memtable to disk:
```
flydb:users> commit
Committed 5 document(s) to disk
```

#### `count`
Show document counts:
```
flydb:users> count
Memtable (uncommitted): 3
Indexed (on disk):      10
Total:                  13
```

#### `export <filename>`
Export collection to a JSON file:
```
flydb:users> export users-backup.json
Exporting collection to users-backup.json...
✓ Created demo file: users-backup.json
```

With compression enabled:
```
flydb:users> export users-backup.json
Exporting collection to users-backup.json...
Will compress output to users-backup.json.gz
✓ Created compressed demo file: users-backup.json.gz
```

### Advanced Commands

#### `compress on|off`
Enable or disable gzip compression:
```
flydb> compress on
✓ Compression enabled (gzip)
Note: Compression applies to future operations

flydb> compress off
✓ Compression disabled

flydb> compress
Compression is currently: ON
Usage: compress on|off
```

**Benefits:**
- Reduces export file sizes
- Useful for data transfer and backups
- Standard gzip format compatible with all tools

### General Commands

#### `help`
Display all available commands with descriptions.

#### `exit` or `quit`
Exit the shell gracefully.

## Tips and Best practices

### 1. Always Commit Your Changes
Documents are stored in memory until you call `commit`:
```
flydb:users> insert {"id":"1","name":"Alice"}
flydb:users> insert {"id":"2","name":"Bob"}
flydb:users> commit  # Save to disk!
```

### 2. Check Counts Before Committing
Use `count` to see how many uncommitted documents you have:
```
flydb:users> count
Memtable (uncommitted): 100
Indexed (on disk):      0
Total:                  100

flydb:users> commit
Committed 100 document(s) to disk
```

### 3. Use Stats to Monitor Database
```
flydb> show stats
```
This shows all loaded collections and their sizes.

### 4. Enable Compression for Large Exports
```
flydb> compress on
flydb:users> export backup.json
```
This creates a compressed `.json.gz` file.

### 5. Query Before You Insert
Check if similar documents exist:
```
flydb:users> query name = Alice
flydb:users> insert {"id":"new","name":"Alice","age":30}
```

## Workflow Examples

### Example 1: Building a User Database

```
flydb> use users
flydb:users> insert {"id":"1","name":"Alice","email":"alice@example.com","role":"admin"}
flydb:users> insert {"id":"2","name":"Bob","email":"bob@example.com","role":"user"}
flydb:users> insert {"id":"3","name":"Charlie","email":"charlie@example.com","role":"user"}
flydb:users> commit
flydb:users> find 1
flydb:users> count
flydb:users> show stats
```

### Example 2: Product Catalog

```
flydb> use products
flydb:products> insert {"id":"prod-1","name":"Laptop","price":999.99,"stock":50}
flydb:products> insert {"id":"prod-2","name":"Mouse","price":29.99,"stock":200}
flydb:products> insert {"id":"prod-3","name":"Keyboard","price":79.99,"stock":100}
flydb:products> commit
flydb:products> query price < 100
flydb:products> export products-catalog.json
```

### Example 3: Data Migration

```
flydb> use old_data
flydb:old_data> show stats
flydb:old_data> compress on
flydb:old_data> export backup.json
flydb:old_data> use new_data
flydb:new_data> # Import functionality would go here
```

## Limitations

### Current Query Implementation
The query command currently parses expressions but doesn't perform full document scanning. To implement full queries:

1. Add document iteration support to collections
2. Scan memtable and all indexed documents
3. Apply filter predicate to each document
4. Return matching results

### No Document Updates
To update a document:
1. Insert a new version with the same ID
2. Commit to disk
3. The latest version will be returned by `find`

### No Document Deletion
Currently, FlyDB is append-only. Deletion would require:
- Tombstone records
- Compaction to remove old versions

## Advanced: Query Language Extension

The query parser in the shell can be extended to support:

### Logical Operators
```
query age > 30 AND role = admin
query price < 100 OR stock > 1000
```

### Pattern Matching
```
query name LIKE %alice%
query email ENDS_WITH @example.com
```

### Multiple Fields
```
query age BETWEEN 25 AND 35
query tags CONTAINS golang
```

### Sorting and Limiting
```
query age > 25 ORDER BY age DESC LIMIT 10
```

These features require implementing a full query engine with an AST parser and execution planner.

## Troubleshooting

### "No collection selected" Error
```
flydb> insert {"id":"1"}
Error: No collection selected. Use 'use <collection>' first

flydb> use mycollection
flydb:mycollection> insert {"id":"1"}
✓
```

### "Invalid JSON" Error
```
flydb:users> insert {id:1}
Error: Invalid JSON: invalid character 'i' looking for beginning of object key string

flydb:users> insert {"id":"1"}
✓
```
JSON requires double quotes around keys and string values.

### "Document missing 'id' field"
```
flydb:users> insert {"name":"Alice"}
Error: document missing 'id' field

flydb:users> insert {"id":"1","name":"Alice"}
✓
```

## Next Steps

- See [QUICKSTART.md](QUICKSTART.md) for programmatic API usage
- See [ARCHITECTURE.md](ARCHITECTURE.md) to understand how FlyDB works
- See [TOON_SPEC.md](TOON_SPEC.md) to learn about the data format
- See [CONTRIBUTING.md](../CONTRIBUTING.md) to contribute features

---

Happy querying! 🚀
