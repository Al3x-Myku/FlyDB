package toon

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
)

func Decode(data []byte, targetID string) (Document, error) {
	reader := bytes.NewReader(data)
	scanner := bufio.NewScanner(reader)

	if !scanner.Scan() {
		return nil, ErrEmptyBlock
	}
	header := scanner.Text()
	count, schema, idColumnIndex, err := ParseHeader(header)
	if err != nil {
		return nil, fmt.Errorf("failed to parse TOON header: %w", err)
	}

	for i := 0; i < count; i++ {
		if !scanner.Scan() {
			return nil, ErrMalformedBlock
		}

		line := scanner.Text()
		row := parseTOONRow(line)

		if len(row) != len(schema) {
			return nil, ErrSchemaMismatch
		}

		if row[idColumnIndex] == targetID {

			doc := make(Document)
			for j, key := range schema {
				doc[key] = inferType(row[j])
			}
			return doc, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	return nil, nil
}

func DecodeAll(data []byte) ([]Document, error) {
	reader := bytes.NewReader(data)
	scanner := bufio.NewScanner(reader)

	if !scanner.Scan() {
		return nil, ErrEmptyBlock
	}
	header := scanner.Text()
	count, schema, _, err := ParseHeader(header)
	if err != nil {
		return nil, fmt.Errorf("failed to parse TOON header: %w", err)
	}

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

		doc := make(Document)
		for j, key := range schema {
			doc[key] = inferType(row[j])
		}
		docs = append(docs, doc)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	return docs, nil
}

func ExtractIDs(data []byte) ([]string, error) {
	reader := bytes.NewReader(data)
	scanner := bufio.NewScanner(reader)

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

	ids := make([]string, 0, count)
	for i := 0; i < count; i++ {
		if !scanner.Scan() {
			return nil, ErrMalformedBlock
		}

		line := scanner.Text()

		if idColumnIndex == 0 {
			parts := strings.SplitN(line, ",", 2)
			if len(parts) > 0 {
				ids = append(ids, parts[0])
			}
		} else {

			row := parseTOONRow(line)
			if len(row) > idColumnIndex {
				ids = append(ids, row[idColumnIndex])
			}
		}
	}

	return ids, scanner.Err()
}
