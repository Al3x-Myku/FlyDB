package db

import (
	"errors"

	"github.com/Al3x-Myku/FlyDB/pkg/toon"
)

type Document = toon.Document

type BlockInfo struct {
	Offset int64
	Length int64
}

var (
	ErrNotFound = errors.New("document not found")

	ErrMissingID = toon.ErrMissingID

	ErrCollectionClosed = errors.New("collection is closed")
)
