# Compression Guide

FlyDB supports **gzip compression** for TOON blocks, reducing storage requirements by 60-80% while maintaining fast read/write performance.

## Default Behavior

**Compression is ENABLED by default** for all new databases created with `NewDB()` or `NewDBWithConfig()`.

```go
// Compression enabled by default
db, err := db.NewDB("./data")

// Equivalent to:
config := db.Config{Compression: true}
db, err := db.NewDBWithConfig("./data", config)
```

## Configuration

### At Database Creation

```go
// Enable compression (default)
config := db.Config{Compression: true}
db, err := db.NewDBWithConfig("./data", config)

// Disable compression
config := db.Config{Compression: false}
db, err := db.NewDBWithConfig("./data", config)
```

### Runtime Toggle

```go
// Enable compression for new commits
db.SetCompression(true)

// Disable compression for new commits
db.SetCompression(false)

// Check current state
if db.IsCompressionEnabled() {
    fmt.Println("Compression is ON")
}
```

## How It Works

### Write Path (Commit)
1. Documents in memtable are encoded to TOON format
2. **If compression enabled**: TOON block is compressed with gzip
3. Compressed (or uncompressed) block is appended to `.toon` file
4. Index updated with block location and length

### Read Path (FindByID)
1. Document ID looked up in index → block location
2. Block read from disk at offset
3. **Auto-detect compression**: Check if block starts with gzip magic bytes (`0x1f 0x8b`)
4. If compressed: decompress with gzip
5. TOON block decoded to retrieve document

**Important**: Reads use **auto-detection**, not the compression flag. This ensures backward compatibility with existing uncompressed files.

## Compression Detection

FlyDB **automatically detects** compressed vs uncompressed blocks:

- **Compressed blocks**: Start with gzip magic bytes `0x1f 0x8b`
- **Uncompressed blocks**: Start with TOON header `[count]<collection>{schema}`

This allows **mixed-format files** where some blocks are compressed and others are not.

## Performance

### Storage Savings
- **TOON alone**: ~45% smaller than JSON (schema deduplication)
- **TOON + gzip**: ~75% smaller than JSON (total reduction)

### Speed Impact
- **Writes**: ~10-15% slower (gzip compression overhead)
- **Reads**: ~5-10% slower (gzip decompression overhead)
- **Index lookups**: No impact (index is always in memory)

## Shell Usage

The FlyDB shell uses compression by default:

```bash
flydb> compress on    # Enable compression
flydb> compress off   # Disable compression
flydb> show stats     # Shows compression status

Database: ./flydb-data
Collections: 1
Compression: ON       # ← Current state
```

### Export with Compression

```bash
flydb> export backup.toon      # Creates compressed backup.toon.gz (if compression ON)
flydb> export backup.toon      # Creates uncompressed backup.toon (if compression OFF)
```

## Backward Compatibility

### Reading Old Files
- FlyDB automatically detects uncompressed blocks in existing `.toon` files
- No migration needed - old files work as-is

### Writing to Old Files
- When compression is enabled, new blocks are compressed
- Existing uncompressed blocks remain uncompressed
- Result: **Mixed-format file** (both compressed and uncompressed blocks)

### Migrating Files

To compress an existing uncompressed database:

```bash
# Method 1: Use shell
flydb> use mycollection
flydb> compress on
flydb> commit

# Method 2: Programmatic
db, _ := db.NewDB("./data")
db.SetCompression(true)
coll, _ := db.GetCollection("mycollection")
// Insert new documents or re-commit
coll.Commit()  // New blocks will be compressed
```

**Note**: Existing blocks are NOT recompressed. To fully compress, export and re-import:

```go
// 1. Export all documents
// 2. Delete old file
// 3. Create new database with compression
// 4. Import documents
// 5. Commit (compressed)
```

## Advanced: Compression Levels

FlyDB uses Go's standard `gzip.DefaultCompression` (level 6).

To customize compression level, modify `pkg/db/collection.go`:

```go
// In Commit() function
gzipWriter, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)  // Level 9 (slower, smaller)
gzipWriter, err := gzip.NewWriterLevel(&buf, gzip.BestSpeed)        // Level 1 (faster, larger)
```

## Best Practices

1. **Use compression for production** - 60-80% storage savings outweigh small performance cost
2. **Disable for development** - Easier debugging with plain text files
3. **Monitor with `show stats`** - Track compression status in shell
4. **Benchmark your workload** - Test with your data to measure impact
5. **Keep compression consistent** - Avoid toggling mid-session to prevent mixed blocks

## Troubleshooting

### "Could not decompress block" error
- **Cause**: Corrupted gzip data or partial write
- **Fix**: Check disk space, verify file integrity, restore from backup

### Mixed compressed/uncompressed blocks
- **Not an error**: FlyDB supports this by design
- **To fix**: Export and reimport to create uniform file

### Compression not working
```go
// Check configuration
if !db.IsCompressionEnabled() {
    db.SetCompression(true)
}

// Verify after commit
// Compressed blocks start with 0x1f8b hex bytes
```

## Examples

### Full Compression Example

```go
package main

import (
    "fmt"
    "os"
    "github.com/Al3x-Myku/FlyDB/pkg/db"
)

func main() {
    // Create database with compression
    config := db.Config{Compression: true}
    database, _ := db.NewDBWithConfig("./data", config)
    defer database.Close()

    // Create collection
    users, _ := database.GetCollection("users")

    // Insert documents
    for i := 1; i <= 100; i++ {
        users.Insert(db.Document{
            "id":   fmt.Sprintf("%d", i),
            "name": fmt.Sprintf("User%d", i),
            "age":  20 + i,
        })
    }

    // Commit - blocks will be compressed
    users.Commit()
    fmt.Println("✓ 100 documents committed (compressed)")

    // Toggle compression off
    database.SetCompression(false)

    // Insert more documents
    for i := 101; i <= 200; i++ {
        users.Insert(db.Document{
            "id":   fmt.Sprintf("%d", i),
            "name": fmt.Sprintf("User%d", i),
            "age":  20 + i,
        })
    }

    // Commit - blocks will NOT be compressed
    users.Commit()
    fmt.Println("✓ 100 documents committed (uncompressed)")

    // Result: users.toon contains both compressed and uncompressed blocks
}
```

## Summary

| Feature | Status |
|---------|--------|
| Default Compression | ✓ Enabled |
| Runtime Toggle | ✓ Supported |
| Mixed Format Files | ✓ Supported |
| Auto-Detection | ✓ Automatic |
| Backward Compatible | ✓ Yes |
| Storage Savings | ~60-80% |
| Performance Impact | ~5-15% |

**Recommendation**: Keep compression enabled for production use. Disable only for debugging or performance-critical scenarios.
