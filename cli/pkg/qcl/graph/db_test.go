package graph

import (
	"context"
	"os"
	"testing"
)

func TestInitDB(t *testing.T) {
	tempDir := t.TempDir()

	origWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origWd)

	db, err := InitDB()
	if err != nil {
		t.Fatalf("Expected successful DB init, got %v", err)
	}
	defer db.Close()

	if db.GetDB() == nil {
		t.Fatalf("Expected non-nil sql.DB")
	}

	testDBUpsertConflict(t, db)
}

func testDBUpsertConflict(t *testing.T, db *GraphDB) {
	query := `
		INSERT INTO nodes (id, type, name, content, metadata) 
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET 
			name=excluded.name, 
			content=excluded.content;
	`
	_, err := db.GetDB().ExecContext(context.Background(), query, "node1", "doc", "First Name", "content1", "{}")
	if err != nil {
		t.Fatalf("Initial insert failed: %v", err)
	}

	_, err = db.GetDB().ExecContext(context.Background(), query, "node1", "doc", "Second Name", "content2", "{}")
	if err != nil {
		t.Fatalf("Upsert on conflict failed: %v", err)
	}

	verifyDBUpsert(t, db)
}

func verifyDBUpsert(t *testing.T, db *GraphDB) {
	var name string
	err := db.GetDB().QueryRowContext(context.Background(), "SELECT name FROM nodes WHERE id = ?", "node1").Scan(&name)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if name != "Second Name" {
		t.Errorf("Expected 'Second Name', got %v", name)
	}
}

func TestInitDB_PermissionError(t *testing.T) {
	tempDir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origWd)

	// Creamos un archivo donde debería ir la carpeta .qdd para forzar error en MkdirAll o Open
	os.WriteFile(".qdd", []byte("file"), 0644)

	_, err := InitDB()
	if err == nil {
		t.Errorf("Expected error when .qdd is a file, got nil")
	}
}
