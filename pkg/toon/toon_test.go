package toon

import (
	"reflect"
	"testing"
)

func TestEncodeDecode(t *testing.T) {
	docs := []Document{
		{"id": "1", "name": "Alice", "age": int64(30)},
		{"id": "2", "name": "Bob", "age": int64(25)},
	}

	encoded, err := Encode("users", docs)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	found, err := Decode(encoded, "1")
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if found["name"] != "Alice" {
		t.Errorf("Expected name=Alice, got %v", found["name"])
	}
}

func TestEscaping(t *testing.T) {
	docs := []Document{
		{
			"id":   "1",
			"name": "O'Neill, Jack",
			"bio":  "Line 1\nLine 2",
			"path": "C:\\Users\\Admin",
		},
	}

	encoded, err := Encode("test", docs)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	decoded, err := Decode(encoded, "1")
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if decoded["name"] != "O'Neill, Jack" {
		t.Errorf("Comma escaping failed: %v", decoded["name"])
	}

	if decoded["bio"] != "Line 1\nLine 2" {
		t.Errorf("Newline escaping failed: %v", decoded["bio"])
	}

	if decoded["path"] != "C:\\Users\\Admin" {
		t.Errorf("Backslash escaping failed: %v", decoded["path"])
	}
}

func TestDecodeAll(t *testing.T) {
	docs := []Document{
		{"id": "1", "name": "Alice"},
		{"id": "2", "name": "Bob"},
		{"id": "3", "name": "Charlie"},
	}

	encoded, _ := Encode("test", docs)
	decoded, err := DecodeAll(encoded)

	if err != nil {
		t.Fatalf("DecodeAll failed: %v", err)
	}

	if len(decoded) != 3 {
		t.Errorf("Expected 3 documents, got %d", len(decoded))
	}
}

func TestExtractIDs(t *testing.T) {
	docs := []Document{
		{"id": "1", "name": "Alice"},
		{"id": "2", "name": "Bob"},
	}

	encoded, _ := Encode("test", docs)
	ids, err := ExtractIDs(encoded)

	if err != nil {
		t.Fatalf("ExtractIDs failed: %v", err)
	}

	expected := []string{"1", "2"}
	if !reflect.DeepEqual(ids, expected) {
		t.Errorf("Expected IDs %v, got %v", expected, ids)
	}
}

func TestMissingID(t *testing.T) {
	docs := []Document{
		{"name": "Alice"},
	}

	_, err := Encode("test", docs)
	if err != ErrMissingID {
		t.Errorf("Expected ErrMissingID, got %v", err)
	}
}

func TestTypeInference(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		{"42", int64(42)},
		{"3.14", float64(3.14)},
		{"true", true},
		{"false", false},
		{"hello", "hello"},
	}

	for _, tt := range tests {
		result := inferType(tt.input)
		if !reflect.DeepEqual(result, tt.expected) {
			t.Errorf("inferType(%q) = %v (%T), want %v (%T)",
				tt.input, result, result, tt.expected, tt.expected)
		}
	}
}
