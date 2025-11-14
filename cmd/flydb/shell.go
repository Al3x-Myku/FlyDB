package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Al3x-Myku/FlyDB/pkg/db"
	"github.com/Al3x-Myku/FlyDB/pkg/toon"
)

type Shell struct {
	db          *db.DB
	current     *db.Collection
	dbPath      string
	compression bool
}

func NewShell(dbPath string) (*Shell, error) {
	config := db.Config{Compression: true}
	database, err := db.NewDBWithConfig(dbPath, config)
	if err != nil {
		return nil, err
	}
	return &Shell{db: database, dbPath: dbPath, compression: true}, nil
}

func (s *Shell) Run() {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("FlyDB Shell v1.0")
	fmt.Println("Type 'help' for available commands")
	fmt.Println()

	for {
		if s.current != nil {
			fmt.Printf("flydb:%s> ", s.current.Name())
		} else {
			fmt.Print("flydb> ")
		}

		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if line == "exit" || line == "quit" {
			break
		}

		s.executeCommand(line)
	}

	s.db.Close()
	fmt.Println("\nBye!")
}

func (s *Shell) executeCommand(line string) {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return
	}

	cmd := parts[0]

	switch cmd {
	case "help":
		s.showHelp()
	case "show":
		if len(parts) < 2 {
			fmt.Println("Error: 'show' requires an argument (collections, dbs, stats)")
			return
		}
		s.handleShow(parts[1])
	case "use":
		if len(parts) < 2 {
			fmt.Println("Error: 'use' requires a collection name")
			return
		}
		s.handleUse(parts[1])
	case "insert":
		if s.current == nil {
			fmt.Println("Error: No collection selected. Use 'use <collection>' first")
			return
		}
		s.handleInsert(strings.TrimPrefix(line, "insert "))
	case "find":
		if s.current == nil {
			fmt.Println("Error: No collection selected. Use 'use <collection>' first")
			return
		}
		if len(parts) < 2 {
			fmt.Println("Error: 'find' requires a document ID")
			return
		}
		s.handleFind(parts[1])
	case "commit":
		if s.current == nil {
			fmt.Println("Error: No collection selected. Use 'use <collection>' first")
			return
		}
		s.handleCommit()
	case "count":
		if s.current == nil {
			fmt.Println("Error: No collection selected. Use 'use <collection>' first")
			return
		}
		s.handleCount()
	case "stats":
		s.handleStats()
	case "query":
		if s.current == nil {
			fmt.Println("Error: No collection selected. Use 'use <collection>' first")
			return
		}
		s.handleQuery(strings.TrimPrefix(line, "query "))
	case "compress":
		if len(parts) < 2 {
			fmt.Printf("Compression is currently: %s\n", onOff(s.compression))
			fmt.Println("Usage: compress on|off")
			return
		}
		s.handleCompress(parts[1])
	case "export":
		if s.current == nil {
			fmt.Println("Error: No collection selected. Use 'use <collection>' first")
			return
		}
		if len(parts) < 2 {
			fmt.Println("Error: 'export' requires a filename")
			return
		}
		s.handleExport(parts[1])
	default:
		fmt.Printf("Unknown command: %s (type 'help' for available commands)\n", cmd)
	}
}

func (s *Shell) showHelp() {
	fmt.Println("Available commands:")
	fmt.Println()
	fmt.Println("  Database Commands:")
	fmt.Println("    show collections       - List all collections")
	fmt.Println("    show stats             - Show database statistics")
	fmt.Println("    use <collection>       - Switch to a collection")
	fmt.Println()
	fmt.Println("  Collection Commands (require 'use <collection>' first):")
	fmt.Println("    insert <json>          - Insert a document (e.g., insert {\"id\":\"1\",\"name\":\"Alice\"})")
	fmt.Println("    find <id>              - Find a document by ID (outputs TOON format)")
	fmt.Println("    query <expr>           - Query documents (e.g., query age > 30) (outputs TOON format)")
	fmt.Println("    commit                 - Commit pending changes to disk")
	fmt.Println("    count                  - Show memtable and indexed document counts")
	fmt.Println("    stats                  - Show collection statistics")
	fmt.Println("    export <file>          - Export entire collection to TOON file (.toon or .toon.gz)")
	fmt.Println()
	fmt.Println("  Advanced:")
	fmt.Println("    compress on|off        - Enable/disable gzip compression")
	fmt.Println()
	fmt.Println("  Query Language:")
	fmt.Println("    field = value          - Exact match (e.g., name = Alice)")
	fmt.Println("    field > value          - Greater than (e.g., age > 30)")
	fmt.Println("    field < value          - Less than (e.g., price < 100)")
	fmt.Println("    field >= value         - Greater or equal")
	fmt.Println("    field <= value         - Less or equal")
	fmt.Println("    field != value         - Not equal")
	fmt.Println()
	fmt.Println("  General:")
	fmt.Println("    help                   - Show this help message")
	fmt.Println("    exit, quit             - Exit the shell")
	fmt.Println()
}

func (s *Shell) handleShow(what string) {
	switch what {
	case "collections", "cols":
		collections, err := s.db.ListCollections()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		if len(collections) == 0 {
			fmt.Println("No collections found")
			return
		}
		fmt.Println("Collections:")
		for _, name := range collections {
			fmt.Printf("  - %s\n", name)
		}
	case "stats":
		s.handleStats()
	default:
		fmt.Printf("Unknown option for 'show': %s\n", what)
		fmt.Println("Available: collections, stats")
	}
}

func (s *Shell) handleUse(collection string) {
	coll, err := s.db.GetCollection(collection)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	s.current = coll
	fmt.Printf("Switched to collection '%s'\n", collection)
}

func (s *Shell) handleInsert(jsonStr string) {
	jsonStr = strings.TrimSpace(jsonStr)

	var doc db.Document
	if err := json.Unmarshal([]byte(jsonStr), &doc); err != nil {
		fmt.Printf("Error: Invalid JSON: %v\n", err)
		return
	}

	id, err := s.current.Insert(doc)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Inserted document with ID: %s (not yet committed)\n", id)
}

func (s *Shell) handleFind(id string) {
	doc, err := s.current.FindByID(id)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	toonBytes, err := toon.Encode(s.current.Name(), []db.Document{doc})
	if err != nil {
		fmt.Printf("Error formatting result: %v\n", err)
		return
	}

	fmt.Println(string(toonBytes))
}

func (s *Shell) handleCommit() {
	size := s.current.Size()
	if size == 0 {
		fmt.Println("Nothing to commit (memtable is empty)")
		return
	}

	if err := s.current.Commit(); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Committed %d document(s) to disk\n", size)
}

func (s *Shell) handleCount() {
	memSize := s.current.Size()
	indexSize := s.current.IndexSize()

	fmt.Printf("Memtable (uncommitted): %d\n", memSize)
	fmt.Printf("Indexed (on disk):      %d\n", indexSize)
	fmt.Printf("Total:                  %d\n", memSize+indexSize)
}

func (s *Shell) handleStats() {
	stats := s.db.GetStats()

	fmt.Printf("Database: %s\n", stats.DataDir)
	fmt.Printf("Collections: %d\n", stats.CollectionsCount)
	fmt.Printf("Compression: %s\n", onOff(s.compression))
	fmt.Println()

	if len(stats.Collections) == 0 {
		fmt.Println("No collections loaded")
		return
	}

	fmt.Println("Collection Details:")
	for name, coll := range stats.Collections {
		fmt.Printf("  %s:\n", name)
		fmt.Printf("    Memtable:  %d documents\n", coll.MemtableSize)
		fmt.Printf("    Indexed:   %d documents\n", coll.IndexSize)
		fmt.Printf("    File:      %s\n", coll.FilePath)
	}
}

func (s *Shell) handleQuery(expr string) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		fmt.Println("Error: Query expression is required")
		fmt.Println("Example: query age > 30")
		return
	}

	field, op, value, err := parseQuery(expr)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	memSize := s.current.Size()
	indexSize := s.current.IndexSize()

	fmt.Printf("Searching %d documents (memtable: %d, indexed: %d)...\n", memSize+indexSize, memSize, indexSize)

	if memSize+indexSize == 0 {
		fmt.Println("No documents found in collection")
		return
	}

	allDocs, err := s.current.All()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	var results []db.Document
	for _, doc := range allDocs {
		if matchesQuery(doc, field, op, value) {
			results = append(results, doc)
		}
	}

	if len(results) == 0 {
		fmt.Println("No documents matched the query")
		return
	}

	fmt.Printf("Found %d matching document(s):\n\n", len(results))

	toonBytes, err := toon.Encode(s.current.Name(), results)
	if err != nil {
		fmt.Printf("Error formatting results: %v\n", err)
		return
	}

	fmt.Println(string(toonBytes))
}

func matchesQuery(doc db.Document, field, operator, value string) bool {
	fieldVal, ok := doc[field]
	if !ok {
		return false
	}

	switch operator {
	case "=":
		return fmt.Sprint(fieldVal) == value
	case "!=":
		return fmt.Sprint(fieldVal) != value
	case ">":
		return compareValues(fieldVal, value) > 0
	case "<":
		return compareValues(fieldVal, value) < 0
	case ">=":
		return compareValues(fieldVal, value) >= 0
	case "<=":
		return compareValues(fieldVal, value) <= 0
	default:
		return false
	}
}

func compareValues(fieldVal interface{}, valueStr string) int {
	switch v := fieldVal.(type) {
	case int64:
		if intVal, err := fmt.Sscanf(valueStr, "%d", new(int64)); err == nil && intVal == 1 {
			var parsedInt int64
			fmt.Sscanf(valueStr, "%d", &parsedInt)
			if v > parsedInt {
				return 1
			} else if v < parsedInt {
				return -1
			}
			return 0
		}
	case float64:
		if floatVal, err := fmt.Sscanf(valueStr, "%f", new(float64)); err == nil && floatVal == 1 {
			var parsedFloat float64
			fmt.Sscanf(valueStr, "%f", &parsedFloat)
			if v > parsedFloat {
				return 1
			} else if v < parsedFloat {
				return -1
			}
			return 0
		}
	case string:
		if v > valueStr {
			return 1
		} else if v < valueStr {
			return -1
		}
		return 0
	}

	return strings.Compare(fmt.Sprint(fieldVal), valueStr)
}

func (s *Shell) handleCompress(mode string) {
	mode = strings.ToLower(mode)
	switch mode {
	case "on", "true", "1", "yes":
		s.compression = true
		s.db.SetCompression(true)
		fmt.Println("✓ Compression enabled (gzip)")
		fmt.Println("Note: New commits will be compressed, existing blocks unchanged")
	case "off", "false", "0", "no":
		s.compression = false
		s.db.SetCompression(false)
		fmt.Println("✓ Compression disabled")
		fmt.Println("Note: New commits will be uncompressed, existing blocks unchanged")
	default:
		fmt.Printf("Unknown mode: %s. Use 'on' or 'off'\n", mode)
	}
}

func (s *Shell) handleExport(filename string) {
	indexSize := s.current.IndexSize()
	memSize := s.current.Size()

	if indexSize == 0 && memSize == 0 {
		fmt.Println("No documents to export (collection is empty)")
		return
	}

	fmt.Printf("Exporting %d documents (memtable: %d, indexed: %d)...\n",
		memSize+indexSize, memSize, indexSize)

	// Retrieve all documents from the collection
	allDocs, err := s.current.All()
	if err != nil {
		fmt.Printf("Error retrieving documents: %v\n", err)
		return
	}

	if len(allDocs) == 0 {
		fmt.Println("No documents to export (collection is empty)")
		return
	}

	// Encode all documents to TOON format
	toonData, err := toon.Encode(s.current.Name(), allDocs)
	if err != nil {
		fmt.Printf("Error encoding TOON: %v\n", err)
		return
	}

	if s.compression {
		if !strings.HasSuffix(filename, ".toon.gz") {
			filename = filename + ".toon.gz"
		}
		fmt.Printf("Compressing output to %s\n", filename)

		var buf bytes.Buffer
		gzWriter := gzip.NewWriter(&buf)
		_, err = gzWriter.Write(toonData)
		if err != nil {
			fmt.Printf("Error compressing data: %v\n", err)
			return
		}
		err = gzWriter.Close()
		if err != nil {
			fmt.Printf("Error closing compressor: %v\n", err)
			return
		}

		err = os.WriteFile(filename, buf.Bytes(), 0644)
		if err != nil {
			fmt.Printf("Error writing compressed file: %v\n", err)
			return
		}
		fmt.Printf("✓ Exported %d documents to compressed TOON: %s (%d bytes compressed)\n",
			len(allDocs), filename, buf.Len())
	} else {
		if !strings.HasSuffix(filename, ".toon") {
			filename = filename + ".toon"
		}

		err = os.WriteFile(filename, toonData, 0644)
		if err != nil {
			fmt.Printf("Error writing file: %v\n", err)
			return
		}
		fmt.Printf("✓ Exported %d documents to TOON: %s (%d bytes)\n",
			len(allDocs), filename, len(toonData))

		// Show preview for uncompressed exports (first 5 lines)
		lines := strings.Split(string(toonData), "\n")
		previewLines := 5
		if len(lines) < previewLines {
			previewLines = len(lines)
		}
		fmt.Printf("\nPreview (first %d lines):\n", previewLines)
		for i := 0; i < previewLines; i++ {
			fmt.Println(lines[i])
		}
		if len(lines) > previewLines {
			fmt.Printf("... (%d more lines)\n", len(lines)-previewLines)
		}
	}
}

func onOff(b bool) string {
	if b {
		return "ON"
	}
	return "OFF"
}

func parseQuery(expr string) (field, operator, value string, err error) {
	operators := []string{">=", "<=", "!=", "=", ">", "<"}

	for _, op := range operators {
		if idx := strings.Index(expr, op); idx != -1 {
			field = strings.TrimSpace(expr[:idx])
			operator = op
			value = strings.TrimSpace(expr[idx+len(op):])

			if field == "" || value == "" {
				err = fmt.Errorf("invalid query format")
				return
			}

			value = strings.Trim(value, "\"'")
			return
		}
	}

	err = fmt.Errorf("no valid operator found (supported: =, !=, >, <, >=, <=)")
	return
}

func main() {
	dbPath := "./flydb-shell-data"
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}

	shell, err := NewShell(dbPath)
	if err != nil {
		fmt.Printf("Error initializing shell: %v\n", err)
		os.Exit(1)
	}

	shell.Run()
}
