package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Al3x-Myku/FlyDB/pkg/db"
)

func main() {
	const dataDir = "./flydb-data"

	_ = os.RemoveAll(dataDir)

	fmt.Println("=== FlyDB Demo: Memtable-on-TOON Architecture ===")
	fmt.Println()

	database, err := db.NewDB(dataDir)
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}
	defer database.Close()

	fmt.Println("✓ Database initialized at:", dataDir)

	users, err := database.GetCollection("users")
	if err != nil {
		log.Fatalf("Failed to get collection: %v", err)
	}
	fmt.Println("✓ Collection 'users' ready")
	fmt.Println()

	fmt.Println("--- Inserting Documents ---")

	doc1 := db.Document{
		"id":    "1",
		"name":  "Alice Johnson",
		"role":  "admin",
		"email": "alice@example.com",
		"age":   30,
	}
	id1, err := users.Insert(doc1)
	if err != nil {
		log.Fatalf("Insert failed: %v", err)
	}
	fmt.Printf("Inserted: ID=%s, Name=%s, Role=%s\n", id1, doc1["name"], doc1["role"])

	doc2 := db.Document{
		"id":    "2",
		"name":  "Bob Smith",
		"role":  "user",
		"email": "bob@example.com",
		"age":   25,
	}
	id2, err := users.Insert(doc2)
	if err != nil {
		log.Fatalf("Insert failed: %v", err)
	}
	fmt.Printf("Inserted: ID=%s, Name=%s, Role=%s\n", id2, doc2["name"], doc2["role"])

	fmt.Println("\n--- Committing to Disk ---")
	if err := users.Commit(); err != nil {
		log.Fatalf("Commit failed: %v", err)
	}
	fmt.Println("✓ Batch committed (Alice & Bob written to disk)")

	fmt.Println("\n--- Inserting More Documents (Memtable) ---")

	doc3 := db.Document{
		"id":    "3",
		"name":  "Charlie Brown",
		"role":  "user",
		"email": "charlie@example.com",
		"age":   28,
	}
	id3, err := users.Insert(doc3)
	if err != nil {
		log.Fatalf("Insert failed: %v", err)
	}
	fmt.Printf("Inserted: ID=%s, Name=%s (uncommitted)\n", id3, doc3["name"])

	fmt.Println("\n--- Querying Documents ---")

	found1, err := users.FindByID("1")
	if err != nil {
		log.Fatalf("FindByID failed: %v", err)
	}
	fmt.Printf("Found (from disk): %v\n", found1)

	found3, err := users.FindByID("3")
	if err != nil {
		log.Fatalf("FindByID failed: %v", err)
	}
	fmt.Printf("Found (from memtable): %v\n", found3)

	fmt.Println("\n--- Testing TOON Escaping ---")
	doc4 := db.Document{
		"id":          "4",
		"name":        "Dave, Jr.",
		"description": "Line 1\nLine 2\nLine 3",
		"role":        "admin\\user",
	}
	_, err = users.Insert(doc4)
	if err != nil {
		log.Fatalf("Insert failed: %v", err)
	}

	if err := users.Commit(); err != nil {
		log.Fatalf("Commit failed: %v", err)
	}

	found4, err := users.FindByID("4")
	if err != nil {
		log.Fatalf("FindByID failed: %v", err)
	}
	fmt.Printf("Document with special chars: %v\n", found4)

	fmt.Println("\n--- Inserting Uncommitted Document ---")
	doc5 := db.Document{
		"id":   "5",
		"name": "Eve (uncommitted)",
	}
	_, err = users.Insert(doc5)
	if err != nil {
		log.Fatalf("Insert failed: %v", err)
	}
	fmt.Println("Inserted doc5 but NOT committing")
	fmt.Println("\n--- Database Statistics ---")
	stats := database.GetStats()
	fmt.Printf("Data Directory: %s\n", stats.DataDir)
	fmt.Printf("Collections: %d\n", stats.CollectionsCount)
	for name, collStats := range stats.Collections {
		fmt.Printf("  - %s: %d in memtable, %d indexed on disk\n",
			name, collStats.MemtableSize, collStats.IndexSize)
	}

	fmt.Println("\n=== Simulating Database Restart ===")
	if err := database.Close(); err != nil {
		log.Fatalf("Close failed: %v", err)
	}
	fmt.Println("✓ Database closed")

	database2, err := db.NewDB(dataDir)
	if err != nil {
		log.Fatalf("Failed to restart database: %v", err)
	}
	defer database2.Close()

	users2, err := database2.GetCollection("users")
	if err != nil {
		log.Fatalf("Failed to get collection: %v", err)
	}
	fmt.Println("✓ Database reopened, index loaded from disk")

	fmt.Println("\n--- Querying After Restart ---")

	found1Again, err := users2.FindByID("1")
	if err != nil {
		log.Fatalf("FindByID failed: %v", err)
	}
	fmt.Printf("Alice (persisted): %v\n", found1Again)

	found4Again, err := users2.FindByID("4")
	if err != nil {
		log.Fatalf("FindByID failed: %v", err)
	}
	fmt.Printf("Dave (persisted): %v\n", found4Again)

	found3Again, err := users2.FindByID("3")
	if err != nil {
		log.Fatalf("FindByID failed: %v", err)
	}
	fmt.Printf("Charlie (persisted with Dave): %v\n", found3Again)

	_, err = users2.FindByID("5")
	if err == db.ErrNotFound {
		fmt.Printf("Eve (not committed): Not found ✓\n")
	} else {
		log.Fatalf("Expected ErrNotFound for ID 5, got: %v", err)
	}

	fmt.Println("\n=== Demo Complete ===")
}
