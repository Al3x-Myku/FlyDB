package db

import (
	"errors"

	"github.com/Al3x-Myku/FlyDB/pkg/toon"
)

// Document is re-exported from the toon package for convenience.
type Document = toon.Document

// BlockInfo stores the on-disk location of a TOON block.
// A single block may contain many documents.
type BlockInfo struct {
	Offset int64 // Byte offset of the block's header in the file
	Length int64 // Total byte length of the block (header + data lines)
}

var (
	// ErrNotFound indicates that a requested document was not found.
	ErrNotFound = errors.New("document not found")

	// ErrMissingID indicates that a document is missing the required 'id' field.
	ErrMissingID = toon.ErrMissingID

	// ErrCollectionClosed indicates an operation on a closed collection.
	ErrCollectionClosed = errors.New("collection is closed")
)
