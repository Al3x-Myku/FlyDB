package toon

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

func escapeTOON(s string) string {
	var replacer = strings.NewReplacer(
		`\`, `\\`,
		`,`, `\,`,
		"\n", `\n`,
		"\r", `\r`,
	)
	return replacer.Replace(s)
}

func Encode(name string, docs []Document) ([]byte, error) {
	if len(docs) == 0 {
		return nil, nil
	}

	keyMap := make(map[string]bool)
	for _, doc := range docs {
		if _, ok := doc["id"]; !ok {
			return nil, ErrMissingID
		}
		for k := range doc {
			keyMap[k] = true
		}
	}

	schema := make([]string, 0, len(keyMap))
	for k := range keyMap {
		schema = append(schema, k)
	}
	sort.Slice(schema, func(i, j int) bool {
		if schema[i] == "id" {
			return true
		}
		if schema[j] == "id" {
			return false
		}
		return schema[i] < schema[j]
	})

	var dataBuf bytes.Buffer
	values := make([]string, len(schema))

	for _, doc := range docs {
		for i, key := range schema {
			val, _ := doc[key]
			valStr := fmt.Sprint(val)
			values[i] = escapeTOON(valStr)
		}
		dataBuf.WriteString(strings.Join(values, ","))
		dataBuf.WriteByte('\n')
	}

	header := fmt.Sprintf("%s[%d]{%s}:\n",
		name,
		len(docs),
		strings.Join(schema, ","),
	)

	var finalBuf bytes.Buffer
	finalBuf.WriteString(header)
	finalBuf.Write(dataBuf.Bytes())

	return finalBuf.Bytes(), nil
}
