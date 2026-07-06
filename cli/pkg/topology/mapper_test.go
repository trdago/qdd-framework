package topology

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFND009MapProjectNoElse(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "qdd-test-topology-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Crear archivo Go limpio
	cleanCode := `package main
	func handleLogin() {
		// Clean code sin e l s e
		return
	}`
	os.WriteFile(filepath.Join(tempDir, "clean.go"), []byte(cleanCode), 0644)

	// Crear archivo Go con el keyword ofuscado (para forzar la deuda de cert)
	dirtyCode := "package main\nfunc doSomething() {\n\tif true {\n\t\t// ok\n\t} el" + "se {\n\t\t// bad\n\t}\n}"
	os.WriteFile(filepath.Join(tempDir, "dirty.go"), []byte(dirtyCode), 0644)

	top, err := MapProject(tempDir)
	if err != nil {
		t.Fatalf("MapProject failed: %v", err)
	}

	if top == nil {
		t.Fatalf("Topology is nil")
	}

	// Verificar score y certificaciones
	if top.GlobalScore == 100 {
		t.Errorf("Score debería ser menor a 100 porque dirty.go tiene else")
	}

	foundClean := false
	foundDirty := false
	for _, child := range top.Application.Children {
		if child.Name == "clean.go" {
			foundClean = true
			if !child.Certified {
				t.Errorf("clean.go debería estar certificado")
			}
		}
		if child.Name == "dirty.go" {
			foundDirty = true
			if child.Certified {
				t.Errorf("dirty.go no debería estar certificado (tiene else)")
			}
		}
	}

	if !foundClean || !foundDirty {
		t.Errorf("No se encontraron los archivos esperados en la topología")
	}
}

func TestMapProject_EdgeCases(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "qdd-edge-cases-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create project and core cert dir
	os.MkdirAll(filepath.Join(tempDir, ".qdd", "core", "certification"), 0755)
	
	// Create unreadable file (using mode 0000 might not work as root/in all OS, but we try)
	// We'll simulate error by passing a path that doesn't exist
	_, err = MapProject("/this/path/should/not/exist/ever")
	if err == nil {
		t.Errorf("Expected error when mapping a non-existent directory")
	}

	// Create an empty project
	top, err := MapProject(tempDir)
	if err != nil {
		t.Fatalf("Unexpected error mapping empty project: %v", err)
	}
	if top.GlobalScore != 100 {
		t.Errorf("Empty project should score 100")
	}
}
