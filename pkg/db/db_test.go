package db

import (
	"os"
	"testing"
)

func TestBasicOperations(t *testing.T) {

	dataDir := "./test-data"
	defer os.RemoveAll(dataDir)

	db, err := NewDB(dataDir)
	if err != nil {
		t.Fatalf("NewDB failed: %v", err)
	}
	defer db.Close()

	users, err := db.GetCollection("users")
	if err != nil {
		t.Fatalf("GetCollection failed: %v", err)
	}

	doc := Document{
		"id":   "1",
		"name": "Alice",
		"age":  30,
	}
	id, err := users.Insert(doc)
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}
	if id != "1" {
		t.Errorf("Expected id=1, got %s", id)
	}

	if err := users.Commit(); err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	found, err := users.FindByID("1")
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if found["name"] != "Alice" {
		t.Errorf("Expected name=Alice, got %v", found["name"])
	}
}

func TestMemtableQuery(t *testing.T) {
	dataDir := "./test-memtable"
	defer os.RemoveAll(dataDir)

	db, _ := NewDB(dataDir)
	defer db.Close()

	users, _ := db.GetCollection("users")

	doc := Document{"id": "1", "name": "Bob"}
	users.Insert(doc)

	found, err := users.FindByID("1")
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if found["name"] != "Bob" {
		t.Errorf("Expected name=Bob, got %v", found["name"])
	}
}

func TestPersistence(t *testing.T) {
	dataDir := "./test-persistence"
	defer os.RemoveAll(dataDir)

	{
		db, _ := NewDB(dataDir)
		users, _ := db.GetCollection("users")
		users.Insert(Document{"id": "1", "name": "Charlie"})
		users.Commit()
		db.Close()
	}

	{
		db, _ := NewDB(dataDir)
		users, _ := db.GetCollection("users")
		found, err := users.FindByID("1")
		if err != nil {
			t.Fatalf("FindByID after restart failed: %v", err)
		}
		if found["name"] != "Charlie" {
			t.Errorf("Expected name=Charlie, got %v", found["name"])
		}
		db.Close()
	}
}

func TestNotFound(t *testing.T) {
	dataDir := "./test-notfound"
	defer os.RemoveAll(dataDir)

	db, _ := NewDB(dataDir)
	defer db.Close()

	users, _ := db.GetCollection("users")

	_, err := users.FindByID("nonexistent")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestBatchInsert(t *testing.T) {
	dataDir := "./test-batch"
	defer os.RemoveAll(dataDir)

	db, _ := NewDB(dataDir)
	defer db.Close()

	products, _ := db.GetCollection("products")

	for i := 0; i < 100; i++ {
		doc := Document{
			"id":    string(rune('0'+(i%10))) + string(rune('0'+(i/10))),
			"name":  "Product",
			"price": float64(i) * 1.99,
		}
		products.Insert(doc)
	}

	if err := products.Commit(); err != nil {
		t.Fatalf("Batch commit failed: %v", err)
	}

	if products.IndexSize() != 100 {
		t.Errorf("Expected 100 indexed documents, got %d", products.IndexSize())
	}
}

func TestUpdate(t *testing.T) {
	dataDir := "./test-update"
	defer os.RemoveAll(dataDir)

	db, _ := NewDB(dataDir)
	defer db.Close()

	users, _ := db.GetCollection("users")

	users.Insert(Document{"id": "1", "name": "Dave", "version": 1})
	users.Commit()

	users.Insert(Document{"id": "1", "name": "Dave", "version": 2})
	users.Commit()

	found, _ := users.FindByID("1")
	if found["version"] != int64(2) {
		t.Errorf("Expected version=2, got %v", found["version"])
	}
}
