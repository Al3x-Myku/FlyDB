package db

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type DB struct {
	dataDir     string
	collections map[string]*Collection
	dbMutex     sync.Mutex
}

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

func (db *DB) GetCollection(name string) (*Collection, error) {
	db.dbMutex.Lock()
	defer db.dbMutex.Unlock()

	if c, ok := db.collections[name]; ok {
		return c, nil
	}

	filePath := filepath.Join(db.dataDir, name+".toon")

	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("could not open collection file: %w", err)
	}

	c := newCollection(name, filePath, file)

	if err := c.loadIndex(); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("could not load index for %s: %w", name, err)
	}

	db.collections[name] = c
	return c, nil
}

func (db *DB) ListCollections() ([]string, error) {

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

type Stats struct {
	DataDir          string
	CollectionsCount int
	Collections      map[string]CollectionStats
}

type CollectionStats struct {
	Name         string
	MemtableSize int
	IndexSize    int
	FilePath     string
}

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
