package toon

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
)

// Decode scans a raw TOON block and finds a *single* document by its ID.
// This is optimized for single-document lookup rather than full block parsing.
//
// Returns the document if found, or an error if not found or the block is malformed.
func Decode(data []byte, targetID string) (Document, error) {
	reader := bytes.NewReader(data)
	scanner := bufio.NewScanner(reader)

	// 1. Parse Header
	if !scanner.Scan() {
		return nil, ErrEmptyBlock
	}
	header := scanner.Text()
	count, schema, idColumnIndex, err := ParseHeader(header)
	if err != nil {
		return nil, fmt.Errorf("failed to parse TOON header: %w", err)
	}

	// 2. Scan Data Lines
	for i := 0; i < count; i++ {
		if !scanner.Scan() {
			return nil, ErrMalformedBlock
		}

		line := scanner.Text()
		row := parseTOONRow(line)

		if len(row) != len(schema) {
			return nil, ErrSchemaMismatch
		}

		// 3. Check for Target ID
		if row[idColumnIndex] == targetID {
			// Found it. Reconstruct the document.
			doc := make(Document)
			for j, key := range schema {
				doc[key] = inferType(row[j])
			}
			return doc, nil
		}
	}

	// 4. Check for scanner errors
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	// Scanned the whole block, target ID not found
	return nil, nil
}

// DecodeAll parses an entire TOON block and returns all documents.
// This is useful for batch operations or full block scans.
func DecodeAll(data []byte) ([]Document, error) {
	reader := bytes.NewReader(data)
	scanner := bufio.NewScanner(reader)

	// 1. Parse Header
	if !scanner.Scan() {
		return nil, ErrEmptyBlock
	}
	header := scanner.Text()
	count, schema, _, err := ParseHeader(header)
	if err != nil {
		return nil, fmt.Errorf("failed to parse TOON header: %w", err)
	}

	// 2. Scan All Data Lines
	docs := make([]Document, 0, count)
	for i := 0; i < count; i++ {
		if !scanner.Scan() {
			return nil, ErrMalformedBlock
		}

		line := scanner.Text()
		row := parseTOONRow(line)

		if len(row) != len(schema) {
			return nil, ErrSchemaMismatch
		}

		// Reconstruct document
		doc := make(Document)
		for j, key := range schema {
			doc[key] = inferType(row[j])
		}
		docs = append(docs, doc)
	}

	// 3. Check for scanner errors
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	return docs, nil
}

// ExtractIDs efficiently extracts all document IDs from a TOON block
// without fully parsing each document. Used for index building.
func ExtractIDs(data []byte) ([]string, error) {
	reader := bytes.NewReader(data)
	scanner := bufio.NewScanner(reader)

	// 1. Parse Header
	if !scanner.Scan() {
		if scanner.Err() == io.EOF {
			return nil, nil
		}
		return nil, ErrEmptyBlock
	}
	header := scanner.Text()
	count, _, idColumnIndex, err := ParseHeader(header)
	if err != nil {
		return nil, fmt.Errorf("failed to parse TOON header: %w", err)
	}

	// 2. Extract IDs efficiently
	ids := make([]string, 0, count)
	for i := 0; i < count; i++ {
		if !scanner.Scan() {
			return nil, ErrMalformedBlock
		}

		line := scanner.Text()

		// Optimize for id as first column (most common case)
		if idColumnIndex == 0 {
			parts := strings.SplitN(line, ",", 2)
			if len(parts) > 0 {
				ids = append(ids, parts[0])
			}
		} else {
			// Parse full row if id is not first
			row := parseTOONRow(line)
			if len(row) > idColumnIndex {
				ids = append(ids, row[idColumnIndex])
			}
		}
	}

	return ids, scanner.Err()
}
