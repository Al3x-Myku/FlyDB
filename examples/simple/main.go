package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Al3x-Myku/FlyDB/pkg/db"
)

func main() {
	const dataDir = "./simple-data"
	_ = os.RemoveAll(dataDir)

	// Create database
	database, err := db.NewDB(dataDir)
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}
	defer database.Close()

	// Get collection
	todos, err := database.GetCollection("todos")
	if err != nil {
		log.Fatalf("Failed to get collection: %v", err)
	}

	fmt.Println("=== Simple Todo App ===")
	fmt.Println()

	// Create todos
	tasks := []db.Document{
		{"id": "1", "task": "Buy groceries", "done": false},
		{"id": "2", "task": "Write documentation", "done": false},
		{"id": "3", "task": "Deploy to production", "done": true},
	}

	fmt.Println("Creating todos:")
	for _, task := range tasks {
		if _, err := todos.Insert(task); err != nil {
			log.Fatalf("Insert failed: %v", err)
		}
		status := "☐"
		if task["done"].(bool) {
			status = "☑"
		}
		fmt.Printf("  %s %s\n", status, task["task"])
	}

	// Commit to disk
	if err := todos.Commit(); err != nil {
		log.Fatalf("Commit failed: %v", err)
	}
	fmt.Println("\n✓ Todos saved to disk")

	// Retrieve a todo
	fmt.Println("\nRetrieving todo #2:")
	todo, err := todos.FindByID("2")
	if err != nil {
		log.Fatalf("FindByID failed: %v", err)
	}
	fmt.Printf("  Task: %s\n", todo["task"])
	fmt.Printf("  Done: %v\n", todo["done"])

	fmt.Println("\n✓ Demo complete")
}
