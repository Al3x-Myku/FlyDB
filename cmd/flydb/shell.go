package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Al3x-Myku/FlyDB/pkg/db"
)

type Shell struct {
	db      *db.DB
	current *db.Collection
	dbPath  string
}

func NewShell(dbPath string) (*Shell, error) {
	database, err := db.NewDB(dbPath)
	if err != nil {
		return nil, err
	}
	return &Shell{db: database, dbPath: dbPath}, nil
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
	fmt.Println("    find <id>              - Find a document by ID")
	fmt.Println("    commit                 - Commit pending changes to disk")
	fmt.Println("    count                  - Show memtable and indexed document counts")
	fmt.Println("    stats                  - Show collection statistics")
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

	jsonBytes, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		fmt.Printf("Error formatting result: %v\n", err)
		return
	}

	fmt.Println(string(jsonBytes))
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
