package db

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/Al3x-Myku/FlyDB/pkg/toon"
)

type Collection struct {
	name        string
	filePath    string
	file        *os.File
	mutex       sync.RWMutex
	memtable    []Document
	index       map[string]BlockInfo
	compression bool
}

func newCollection(name, filePath string, file *os.File, compression bool) *Collection {
	return &Collection{
		name:        name,
		filePath:    filePath,
		file:        file,
		memtable:    make([]Document, 0),
		index:       make(map[string]BlockInfo),
		compression: compression,
	}
}

func (c *Collection) Insert(doc Document) (string, error) {
	idVal, ok := doc["id"]
	if !ok {
		return "", ErrMissingID
	}
	id, ok := idVal.(string)
	if !ok {

		id = fmt.Sprint(idVal)
		doc["id"] = id
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.file == nil {
		return "", ErrCollectionClosed
	}

	c.memtable = append(c.memtable, doc)
	return id, nil
}

// Delete removes a document from the memtable and index
// Note: This is a logical delete that removes from memory and creates a tombstone
func (c *Collection) Delete(id string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.file == nil {
		return ErrCollectionClosed
	}

	// Remove from memtable
	found := false
	for i := len(c.memtable) - 1; i >= 0; i-- {
		if fmt.Sprint(c.memtable[i]["id"]) == id {
			c.memtable = append(c.memtable[:i], c.memtable[i+1:]...)
			found = true
			break
		}
	}

	// Remove from index (will be gone after commit)
	if _, ok := c.index[id]; ok {
		delete(c.index, id)
		found = true
	}

	if !found {
		return ErrNotFound
	}

	return nil
}

// Update modifies an existing document
func (c *Collection) Update(id string, doc Document) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.file == nil {
		return ErrCollectionClosed
	}

	doc["id"] = id

	inMemtable := false
	for i := len(c.memtable) - 1; i >= 0; i-- {
		if fmt.Sprint(c.memtable[i]["id"]) == id {
			c.memtable[i] = doc
			inMemtable = true
			break
		}
	}

	if inMemtable {
		return nil
	}

	if _, ok := c.index[id]; ok {
		c.memtable = append(c.memtable, doc)
		return nil
	}

	return ErrNotFound
}

func (c *Collection) isInMemtable(id string) bool {
	for i := len(c.memtable) - 1; i >= 0; i-- {
		if fmt.Sprint(c.memtable[i]["id"]) == id {
			return true
		}
	}
	return false
}

func (c *Collection) Commit() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.file == nil {
		return ErrCollectionClosed
	}

	if len(c.memtable) == 0 {
		return nil
	}

	toonBlock, err := toon.Encode(c.name, c.memtable)
	if err != nil {
		return fmt.Errorf("could not encode TOON block: %w", err)
	}

	dataToWrite := toonBlock
	if c.compression {
		var buf bytes.Buffer
		gzipWriter := gzip.NewWriter(&buf)
		if _, err := gzipWriter.Write(toonBlock); err != nil {
			return fmt.Errorf("could not compress TOON block: %w", err)
		}
		if err := gzipWriter.Close(); err != nil {
			return fmt.Errorf("could not close gzip writer: %w", err)
		}
		dataToWrite = buf.Bytes()
	}

	offset, err := c.file.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("could not seek to end of file: %w", err)
	}

	n, err := c.file.Write(dataToWrite)
	if err != nil {
		return fmt.Errorf("could not write TOON block to file: %w", err)
	}

	// Flush to ensure data is written to disk
	if err := c.file.Sync(); err != nil {
		return fmt.Errorf("could not sync file: %w", err)
	}

	info := BlockInfo{
		Offset: offset,
		Length: int64(n),
	}

	for _, doc := range c.memtable {
		id := fmt.Sprint(doc["id"])
		c.index[id] = info
	}

	c.memtable = make([]Document, 0)

	return nil
}

func (c *Collection) FindByID(id string) (Document, error) {
	c.mutex.RLock()

	if c.file == nil {
		c.mutex.RUnlock()
		return nil, ErrCollectionClosed
	}

	for i := len(c.memtable) - 1; i >= 0; i-- {
		doc := c.memtable[i]
		if fmt.Sprint(doc["id"]) == id {
			c.mutex.RUnlock()
			return doc, nil
		}
	}

	info, ok := c.index[id]

	c.mutex.RUnlock()

	if !ok {
		return nil, ErrNotFound
	}

	buf := make([]byte, info.Length)

	_, err := c.file.ReadAt(buf, info.Offset)
	if err != nil {
		return nil, fmt.Errorf("could not read block from disk: %w", err)
	}

	blockData := buf
	isCompressed := len(buf) >= 2 && buf[0] == 0x1f && buf[1] == 0x8b
	if isCompressed {
		gzipReader, err := gzip.NewReader(bytes.NewReader(buf))
		if err != nil {
			return nil, fmt.Errorf("could not create gzip reader: %w", err)
		}
		defer func() {
			_ = gzipReader.Close()
		}()

		decompressed, err := io.ReadAll(gzipReader)
		if err != nil {
			return nil, fmt.Errorf("could not decompress block: %w", err)
		}
		blockData = decompressed
	}

	doc, err := toon.Decode(blockData, id)
	if err != nil {
		return nil, fmt.Errorf("could not decode TOON block: %w", err)
	}
	if doc == nil {
		return nil, ErrNotFound
	}

	return doc, nil
}

func (c *Collection) loadIndex() error {

	fileInfo, err := c.file.Stat()
	if err != nil {
		return fmt.Errorf("could not stat file: %w", err)
	}

	if fileInfo.Size() == 0 {

		return nil
	}

	if _, err := c.file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("could not seek to file start: %w", err)
	}

	data, err := io.ReadAll(c.file)
	if err != nil {
		return fmt.Errorf("could not read file: %w", err)
	}

	currentOffset := int64(0)

	for currentOffset < int64(len(data)) {
		blockStart := currentOffset

		isCompressed := false
		if currentOffset+2 < int64(len(data)) && data[currentOffset] == 0x1f && data[currentOffset+1] == 0x8b {
			isCompressed = true
		}

		if isCompressed {
			// Create a reader starting at currentOffset
			reader := bytes.NewReader(data[currentOffset:])
			gzipReader, err := gzip.NewReader(reader)
			if err != nil {
				log.Printf("Warning: Could not create gzip reader at offset %d: %v", blockStart, err)
				currentOffset++
				continue
			}
			gzipReader.Multistream(false)

			decompressed, err := io.ReadAll(gzipReader)
			gzipCloseErr := gzipReader.Close()
			if err != nil {
				log.Printf("Warning: Could not decompress block at offset %d: %v", blockStart, err)
				currentOffset++
				continue
			}
			if gzipCloseErr != nil {
				log.Printf("Warning: Error closing gzip reader at offset %d: %v", blockStart, gzipCloseErr)
			}

			// Calculate how many bytes were consumed from the source
			// by checking the position of the underlying reader
			bytesRemaining := reader.Len()
			bytesConsumed := int64(len(data[currentOffset:])) - int64(bytesRemaining)
			blockLen := bytesConsumed

			ids, err := toon.ExtractIDs(decompressed)
			if err != nil {
				log.Printf("Warning: Could not extract IDs from compressed block at offset %d: %v", blockStart, err)
				currentOffset += blockLen
				continue
			}

			info := BlockInfo{
				Offset: blockStart,
				Length: blockLen,
			}
			for _, id := range ids {
				c.index[id] = info
			}

			currentOffset += blockLen
		} else {
			scanner := bufio.NewScanner(bytes.NewReader(data[currentOffset:]))

			if !scanner.Scan() {
				break
			}
			headerLine := scanner.Text() + "\n"
			headerLen := len(headerLine)

			count, _, _, err := toon.ParseHeader(headerLine)
			if err != nil {

				log.Printf("Warning: Skipping malformed block at offset %d: %v", blockStart, err)
				currentOffset += int64(headerLen)
				continue
			}

			blockData := headerLine
			for i := 0; i < count; i++ {
				if !scanner.Scan() {
					break
				}
				blockData += scanner.Text() + "\n"
			}

			blockLen := int64(len(blockData))

			ids, err := toon.ExtractIDs([]byte(blockData))
			if err != nil {
				log.Printf("Warning: Could not extract IDs from block at offset %d: %v", blockStart, err)
				currentOffset += blockLen
				continue
			}

			info := BlockInfo{
				Offset: blockStart,
				Length: blockLen,
			}
			for _, id := range ids {
				c.index[id] = info
			}

			currentOffset += blockLen
		}
	}

	if _, err := c.file.Seek(0, io.SeekEnd); err != nil {
		return fmt.Errorf("could not seek to file end after index load: %w", err)
	}

	return nil
}

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

func (c *Collection) SetCompression(enabled bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.compression = enabled
}

func (c *Collection) Compact() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.file == nil {
		return ErrCollectionClosed
	}

	allDocs, err := c.allInternal()
	if err != nil {
		return fmt.Errorf("could not get all documents: %w", err)
	}

	if err := c.file.Truncate(0); err != nil {
		return fmt.Errorf("could not truncate file: %w", err)
	}

	if _, err := c.file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("could not seek to start: %w", err)
	}

	c.index = make(map[string]BlockInfo)
	c.memtable = allDocs

	if len(c.memtable) == 0 {
		return nil
	}

	return c.commitInternal()
}

func (c *Collection) commitInternal() error {
	if len(c.memtable) == 0 {
		return nil
	}

	toonBlock, err := toon.Encode(c.name, c.memtable)
	if err != nil {
		return fmt.Errorf("could not encode TOON block: %w", err)
	}

	dataToWrite := toonBlock
	if c.compression {
		var buf bytes.Buffer
		gzipWriter := gzip.NewWriter(&buf)
		if _, err := gzipWriter.Write(toonBlock); err != nil {
			return fmt.Errorf("could not compress TOON block: %w", err)
		}
		if err := gzipWriter.Close(); err != nil {
			return fmt.Errorf("could not close gzip writer: %w", err)
		}
		dataToWrite = buf.Bytes()
	}

	offset, err := c.file.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("could not seek to end of file: %w", err)
	}

	n, err := c.file.Write(dataToWrite)
	if err != nil {
		return fmt.Errorf("could not write TOON block to file: %w", err)
	}

	if err := c.file.Sync(); err != nil {
		return fmt.Errorf("could not sync file: %w", err)
	}

	info := BlockInfo{
		Offset: offset,
		Length: int64(n),
	}

	for _, doc := range c.memtable {
		id := fmt.Sprint(doc["id"])
		c.index[id] = info
	}

	c.memtable = make([]Document, 0)

	return nil
}

func (c *Collection) allInternal() ([]Document, error) {
	if c.file == nil {
		return nil, ErrCollectionClosed
	}

	var allDocs []Document
	seenIDs := make(map[string]bool)

	for i := len(c.memtable) - 1; i >= 0; i-- {
		doc := c.memtable[i]
		id := fmt.Sprint(doc["id"])
		if !seenIDs[id] {
			allDocs = append(allDocs, doc)
			seenIDs[id] = true
		}
	}

	processedBlocks := make(map[BlockInfo]bool)

	for id, info := range c.index {
		if seenIDs[id] {
			continue
		}

		if processedBlocks[info] {
			continue
		}
		processedBlocks[info] = true

		buf := make([]byte, info.Length)
		_, err := c.file.ReadAt(buf, info.Offset)
		if err != nil {
			return nil, fmt.Errorf("could not read block from disk: %w", err)
		}

		blockData := buf
		isCompressed := len(buf) >= 2 && buf[0] == 0x1f && buf[1] == 0x8b
		if isCompressed {
			gzipReader, err := gzip.NewReader(bytes.NewReader(buf))
			if err != nil {
				log.Printf("Warning: Could not create gzip reader: %v", err)
				continue
			}

			decompressed, err := io.ReadAll(gzipReader)
			_ = gzipReader.Close()
			if err != nil {
				log.Printf("Warning: Could not decompress block: %v", err)
				continue
			}
			blockData = decompressed
		}

		docs, err := toon.DecodeAll(blockData)
		if err != nil {
			log.Printf("Warning: Could not decode block: %v", err)
			continue
		}

		for _, doc := range docs {
			docID := fmt.Sprint(doc["id"])
			if !seenIDs[docID] {
				allDocs = append(allDocs, doc)
				seenIDs[docID] = true
			}
		}
	}

	return allDocs, nil
}

func (c *Collection) All() ([]Document, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.allInternal()
}
