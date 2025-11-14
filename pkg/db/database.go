package db

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// DB is the main database instance. It manages collections and global state.
//
// A database consists of:
//   - A data directory containing .toon files
//   - A map of loaded collections
//   - Thread-safe access to collections
type DB struct {
	dataDir     string
	collections map[string]*Collection
	dbMutex     sync.Mutex // Protects the 'collections' map
}

// NewDB initializes a new database at the given data directory.
// It scans the directory for existing collection files and loads them on-demand.
//
// The data directory will be created if it doesn't exist.
func NewDB(dataDir string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("could not create data dir: %w", err)
	}

	db := &DB{
		dataDir:     dataDir,
		collections: make(map[string]*Collection),
	}

	return db, nil
}

// GetCollection retrieves or creates a collection.
// If the collection doesn't exist, it will be created.
// If it exists on disk, the index will be loaded into memory.
func (db *DB) GetCollection(name string) (*Collection, error) {
	db.dbMutex.Lock()
	defer db.dbMutex.Unlock()

	// 1. Check if already loaded
	if c, ok := db.collections[name]; ok {
		return c, nil
	}

	// 2. Create new collection
	filePath := filepath.Join(db.dataDir, name+".toon")

	// Open file with O_RDWR (read-write), O_CREATE (create if not exist)
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("could not open collection file: %w", err)
	}

	c := newCollection(name, filePath, file)

	// 3. Load the on-disk index into memory
	if err := c.loadIndex(); err != nil {
		file.Close()
		return nil, fmt.Errorf("could not load index for %s: %w", name, err)
	}

	db.collections[name] = c
	return c, nil
}

// ListCollections returns the names of all collections (loaded and on-disk).
func (db *DB) ListCollections() ([]string, error) {
	// Scan for .toon files in the data directory
	files, err := filepath.Glob(filepath.Join(db.dataDir, "*.toon"))
	if err != nil {
		return nil, fmt.Errorf("could not scan data dir: %w", err)
	}

	names := make([]string, 0, len(files))
	for _, fPath := range files {
		baseName := filepath.Base(fPath)
		name := strings.TrimSuffix(baseName, ".toon")
		names = append(names, name)
	}

	return names, nil
}

// LoadAllCollections pre-loads all collection files into memory.
// This is optional but can improve performance if you know you'll need all collections.
func (db *DB) LoadAllCollections() error {
	names, err := db.ListCollections()
	if err != nil {
		return err
	}

	for _, name := range names {
		if _, err := db.GetCollection(name); err != nil {
			log.Printf("Warning: Failed to load collection %s: %v", name, err)
		}
	}

	return nil
}

// Close gracefully closes the file handles for all collections.
// This should be called before the application exits.
func (db *DB) Close() error {
	db.dbMutex.Lock()
	defer db.dbMutex.Unlock()

	var firstErr error
	for name, c := range db.collections {
		if err := c.Close(); err != nil {
			log.Printf("Error closing collection %s: %v", name, err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// Stats returns database statistics.
type Stats struct {
	DataDir          string
	CollectionsCount int
	Collections      map[string]CollectionStats
}

// CollectionStats contains statistics for a single collection.
type CollectionStats struct {
	Name         string
	MemtableSize int
	IndexSize    int
	FilePath     string
}

// GetStats returns current database statistics.
func (db *DB) GetStats() Stats {
	db.dbMutex.Lock()
	defer db.dbMutex.Unlock()

	stats := Stats{
		DataDir:          db.dataDir,
		CollectionsCount: len(db.collections),
		Collections:      make(map[string]CollectionStats),
	}

	for name, c := range db.collections {
		stats.Collections[name] = CollectionStats{
			Name:         name,
			MemtableSize: c.Size(),
			IndexSize:    c.IndexSize(),
			FilePath:     c.filePath,
		}
	}

	return stats
}
