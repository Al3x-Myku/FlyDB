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
	if c.compression {
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
			gzipReader, err := gzip.NewReader(bytes.NewReader(data[currentOffset:]))
			if err != nil {
				log.Printf("Warning: Could not create gzip reader at offset %d: %v", blockStart, err)
				currentOffset++
				continue
			}

			decompressed, err := io.ReadAll(gzipReader)
			if err != nil {
				_ = gzipReader.Close()
				log.Printf("Warning: Could not decompress block at offset %d: %v", blockStart, err)
				currentOffset++
				continue
			}
			_ = gzipReader.Close()

			var buf bytes.Buffer
			gzipWriter := gzip.NewWriter(&buf)
			if _, err := gzipWriter.Write(decompressed); err == nil {
				_ = gzipWriter.Close()
				blockLen := int64(buf.Len())

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
				currentOffset++
			}
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
