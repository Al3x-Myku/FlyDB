package toon

import (
	"fmt"
	"strconv"
	"strings"
)

func parseTOONRow(line string) []string {
	var values []string
	var current strings.Builder
	var escaped bool

	for _, r := range line {
		if escaped {
			switch r {
			case '\\':
				current.WriteRune('\\')
			case ',':
				current.WriteRune(',')
			case 'n':
				current.WriteRune('\n')
			case 'r':
				current.WriteRune('\r')
			default:
				current.WriteRune('\\')
				current.WriteRune(r)
			}
			escaped = false
		} else if r == '\\' {
			escaped = true
		} else if r == ',' {
			values = append(values, current.String())
			current.Reset()
		} else {
			current.WriteRune(r)
		}
	}
	values = append(values, current.String())
	return values
}

func ParseHeader(header string) (int, []string, int, error) {
	lBracket := strings.IndexByte(header, '[')
	rBracket := strings.IndexByte(header, ']')
	if lBracket == -1 || rBracket == -1 || rBracket < lBracket {
		return 0, nil, -1, ErrInvalidHeader
	}
	countStr := header[lBracket+1 : rBracket]
	count, err := strconv.Atoi(countStr)
	if err != nil {
		return 0, nil, -1, fmt.Errorf("invalid count: %w", err)
	}

	lBrace := strings.IndexByte(header, '{')
	rBrace := strings.IndexByte(header, '}')
	if lBrace == -1 || rBrace == -1 || rBrace < lBrace {
		return 0, nil, -1, ErrInvalidHeader
	}
	schemaStr := header[lBrace+1 : rBrace]
	schema := strings.Split(schemaStr, ",")

	idColumnIndex := -1
	for i, key := range schema {
		if key == "id" {
			idColumnIndex = i
			break
		}
	}
	if idColumnIndex == -1 {
		return 0, nil, -1, fmt.Errorf("schema missing 'id' key")
	}

	return count, schema, idColumnIndex, nil
}

func inferType(s string) interface{} {
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	if b, err := strconv.ParseBool(s); err == nil {
		return b
	}
	return s
}
