package toon

import "errors"

type Document map[string]interface{}

var (
	ErrMissingID      = errors.New("document missing 'id' field")
	ErrInvalidHeader  = errors.New("invalid TOON header")
	ErrEmptyBlock     = errors.New("empty TOON block")
	ErrMalformedBlock = errors.New("TOON block malformed")
	ErrSchemaMismatch = errors.New("schema/row length mismatch")
)
