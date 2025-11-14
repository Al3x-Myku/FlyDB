package db

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/Al3x-Myku/FlyDB/pkg/toon"
)

// Collection manages a single data file (e.g., "users.toon"),
// its memtable, and its index.
//
// Architecture: "Memtable-on-TOON" LSM-tree variant
//   - Memtable: In-memory buffer for new writes
//   - Index: In-memory map of document IDs to on-disk block locations
//   - File: Append-only TOON-formatted data blocks
type Collection struct {
	name     string
	filePath string
	file     *os.File             // Persistent file handle for O_RDWR
	mutex    sync.RWMutex         // Protects memtable and index
	memtable []Document           // In-memory buffer for new writes
	index    map[string]BlockInfo // In-memory index: docID -> blockInfo
}

// newCollection creates a new collection instance.
// The file must already be opened.
func newCollection(name, filePath string, file *os.File) *Collection {
	return &Collection{
		name:     name,
		filePath: filePath,
		file:     file,
		memtable: make([]Document, 0),
		index:    make(map[string]BlockInfo),
	}
}

// Insert adds a new document to the collection's memtable.
// It does *not* write to disk. Call Commit() to persist.
//
// Returns the document ID and any error.
func (c *Collection) Insert(doc Document) (string, error) {
	idVal, ok := doc["id"]
	if !ok {
		return "", ErrMissingID
	}
	id, ok := idVal.(string)
	if !ok {
		// Try to convert to string
		id = fmt.Sprint(idVal)
		doc["id"] = id // Normalize in-memory
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.file == nil {
		return "", ErrCollectionClosed
	}

	c.memtable = append(c.memtable, doc)
	return id, nil
}

// Commit flushes the memtable to a new TOON block on disk and updates the index.
// This is an atomic operation that either succeeds completely or fails without
// partial writes.
func (c *Collection) Commit() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.file == nil {
		return ErrCollectionClosed
	}

	// 1. Check if there's work to do
	if len(c.memtable) == 0 {
		return nil
	}

	// 2. Serialize memtable to TOON block
	toonBlock, err := toon.Encode(c.name, c.memtable)
	if err != nil {
		return fmt.Errorf("could not encode TOON block: %w", err)
	}

	// 3. Get write offset (current end of file)
	offset, err := c.file.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("could not seek to end of file: %w", err)
	}

	// 4. Write block to disk
	n, err := c.file.Write(toonBlock)
	if err != nil {
		return fmt.Errorf("could not write TOON block to file: %w", err)
	}

	// 5. Create block info entry
	info := BlockInfo{
		Offset: offset,
		Length: int64(n),
	}

	// 6. Update index for all docs in the flushed block
	for _, doc := range c.memtable {
		id := fmt.Sprint(doc["id"])
		c.index[id] = info
	}

	// 7. Clear memtable
	c.memtable = make([]Document, 0)

	return nil
}

// FindByID efficiently finds a document by its ID.
// It searches the memtable first (newest data), then uses the on-disk index.
//
// The read lock is released before disk I/O to maximize concurrency.
func (c *Collection) FindByID(id string) (Document, error) {
	c.mutex.RLock()

	if c.file == nil {
		c.mutex.RUnlock()
		return nil, ErrCollectionClosed
	}

	// 1. Check memtable (newest data)
	// Iterate in reverse to find the most recent version
	for i := len(c.memtable) - 1; i >= 0; i-- {
		doc := c.memtable[i]
		if fmt.Sprint(doc["id"]) == id {
			c.mutex.RUnlock()
			return doc, nil
		}
	}

	// 2. Check on-disk index
	info, ok := c.index[id]

	// 3. *** Release read lock before disk I/O ***
	c.mutex.RUnlock()

	if !ok {
		return nil, ErrNotFound
	}

	// 4. Read the block from disk
	buf := make([]byte, info.Length)

	// os.File.ReadAt is concurrent-safe and does not affect
	// the file's main cursor, making it perfect for this use case.
	_, err := c.file.ReadAt(buf, info.Offset)
	if err != nil {
		return nil, fmt.Errorf("could not read block from disk: %w", err)
	}

	// 5. Parse the block to find the specific document
	doc, err := toon.Decode(buf, id)
	if err != nil {
		return nil, fmt.Errorf("could not decode TOON block: %w", err)
	}
	if doc == nil {
		return nil, ErrNotFound
	}

	return doc, nil
}

// loadIndex rebuilds the in-memory index by scanning the collection file.
// This is called when opening an existing collection.
func (c *Collection) loadIndex() error {
	// 1. Get file size to check if empty
	fileInfo, err := c.file.Stat()
	if err != nil {
		return fmt.Errorf("could not stat file: %w", err)
	}

	if fileInfo.Size() == 0 {
		// Empty file, nothing to index
		return nil
	}

	// 2. Rewind file to start
	if _, err := c.file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("could not seek to file start: %w", err)
	}

	// 3. Read entire file into memory (simplest approach for now)
	data, err := io.ReadAll(c.file)
	if err != nil {
		return fmt.Errorf("could not read file: %w", err)
	}

	// 4. Parse blocks from the data
	currentOffset := int64(0)
	scanner := bufio.NewScanner(bytes.NewReader(data))

	for currentOffset < int64(len(data)) {
		blockStart := currentOffset

		// Read header
		if !scanner.Scan() {
			break
		}
		headerLine := scanner.Text() + "\n"
		headerLen := len(headerLine)

		// Parse header to get count
		count, _, _, err := toon.ParseHeader(headerLine)
		if err != nil {
			// Skip malformed blocks
			log.Printf("Warning: Skipping malformed block at offset %d: %v", blockStart, err)
			currentOffset += int64(headerLen)
			continue
		}

		// Calculate block end by reading 'count' data lines
		blockData := headerLine
		for i := 0; i < count; i++ {
			if !scanner.Scan() {
				break
			}
			blockData += scanner.Text() + "\n"
		}

		blockLen := int64(len(blockData))

		// Extract IDs from this block
		ids, err := toon.ExtractIDs([]byte(blockData))
		if err != nil {
			log.Printf("Warning: Could not extract IDs from block at offset %d: %v", blockStart, err)
			currentOffset += blockLen
			continue
		}

		// Update index
		info := BlockInfo{
			Offset: blockStart,
			Length: blockLen,
		}
		for _, id := range ids {
			c.index[id] = info
		}

		currentOffset += blockLen
	}

	// 5. After loading, seek to end for future appends
	if _, err := c.file.Seek(0, io.SeekEnd); err != nil {
		return fmt.Errorf("could not seek to file end after index load: %w", err)
	}

	return nil
} // findBlockEnd finds the end of a TOON block by looking for the next header
// or end of data. Returns -1 if not found.
func findBlockEnd(data []byte, currentHeader string) int {
	// Simple heuristic: find next line that looks like a header
	// A header contains '[' and ']' and '{' and '}'
	lines := 0
	pos := len(currentHeader)

	for pos < len(data) {
		lineEnd := pos
		for lineEnd < len(data) && data[lineEnd] != '\n' {
			lineEnd++
		}

		lines++
		line := string(data[pos:lineEnd])

		// Check if this looks like a new header
		if lines > 1 && isHeaderLine(line) {
			return pos
		}

		pos = lineEnd + 1
	}

	return len(data)
}

// isHeaderLine checks if a line looks like a TOON header
func isHeaderLine(line string) bool {
	hasLeftBracket := false
	hasRightBracket := false
	hasLeftBrace := false
	hasRightBrace := false

	for _, r := range line {
		switch r {
		case '[':
			hasLeftBracket = true
		case ']':
			hasRightBracket = true
		case '{':
			hasLeftBrace = true
		case '}':
			hasRightBrace = true
		}
	}

	return hasLeftBracket && hasRightBracket && hasLeftBrace && hasRightBrace
}

// Close closes the collection's file handle.
func (c *Collection) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.file != nil {
		err := c.file.Close()
		c.file = nil
		return err
	}
	return nil
}

// Size returns the current size of the memtable (uncommitted documents).
func (c *Collection) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.memtable)
}

func (c *Collection) IndexSize() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.index)
}

func (c *Collection) Name() string {
	return c.name
}
