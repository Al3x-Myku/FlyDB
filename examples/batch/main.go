package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Al3x-Myku/FlyDB/pkg/db"
)

func main() {
	const dataDir = "./benchmark-data"
	_ = os.RemoveAll(dataDir)

	database, err := db.NewDB(dataDir)
	if err != nil {
		log.Fatalf("Failed to create DB: %v", err)
	}
	defer database.Close()

	products, err := database.GetCollection("products")
	if err != nil {
		log.Fatalf("Failed to get collection: %v", err)
	}

	fmt.Println("=== Batch Insert Benchmark ===")

	const batchSize = 1000
	fmt.Printf("Inserting %d documents...\n", batchSize)

	for i := 1; i <= batchSize; i++ {
		doc := db.Document{
			"id":          fmt.Sprintf("prod_%d", i),
			"name":        fmt.Sprintf("Product %d", i),
			"category":    fmt.Sprintf("Category %d", i%10),
			"price":       float64(i) * 1.99,
			"in_stock":    i%2 == 0,
			"description": fmt.Sprintf("This is a detailed description for product number %d", i),
		}

		if _, err := products.Insert(doc); err != nil {
			log.Fatalf("Insert failed: %v", err)
		}

		if i%100 == 0 {
			fmt.Printf("  %d inserted...\n", i)
		}
	}

	fmt.Println("\nCommitting batch to disk...")
	if err := products.Commit(); err != nil {
		log.Fatalf("Commit failed: %v", err)
	}
	fmt.Println("✓ Batch committed")

	fmt.Println("\n=== Query Performance ===")
	testIDs := []string{"prod_1", "prod_500", "prod_999"}

	for _, id := range testIDs {
		doc, err := products.FindByID(id)
		if err != nil {
			log.Fatalf("FindByID(%s) failed: %v", id, err)
		}
		fmt.Printf("Found %s: %s - $%.2f\n", id, doc["name"], doc["price"])
	}

	fmt.Println("\n=== Statistics ===")
	stats := database.GetStats()
	for name, collStats := range stats.Collections {
		fmt.Printf("Collection: %s\n", name)
		fmt.Printf("  Indexed documents: %d\n", collStats.IndexSize)
		fmt.Printf("  Memtable size: %d\n", collStats.MemtableSize)
	}

	fmt.Println("\n✓ Benchmark complete")
}
