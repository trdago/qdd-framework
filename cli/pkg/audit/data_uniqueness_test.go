package audit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckDataUniqueness_DetectsUnjustifiedDuplication(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "qdd_ssot_test_*")
	if err != nil {
		t.Fatalf("Error creando directorio temporal: %v", err)
	}
	defer os.RemoveAll(tempDir)

	badSQL := `
CREATE TABLE orders (
    id INT PRIMARY KEY,
    order_number VARCHAR(50),
    customer_email VARCHAR(255),
    customer_phone VARCHAR(50)
);
`
	sqlPath := filepath.Join(tempDir, "schema.sql")
	if err := os.WriteFile(sqlPath, []byte(badSQL), 0644); err != nil {
		t.Fatalf("Error escribiendo archivo SQL: %v", err)
	}

	violations := CheckDataUniqueness(tempDir)
	if len(violations) == 0 {
		t.Fatalf("Se esperaba detectar violación de Unicidad del Dato (SSOT), pero se obtuvieron 0 violaciones")
	}

	found := false
	for _, v := range violations {
		if v.RuleID == "CERT-034-DATA-UNIQUENESS" {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("No se encontró la regla CERT-034-DATA-UNIQUENESS en las violaciones reportadas")
	}
}

func TestCheckDataUniqueness_AllowsNormalizedWithForeignKey(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "qdd_ssot_clean_*")
	if err != nil {
		t.Fatalf("Error creando directorio temporal: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cleanSQL := `
CREATE TABLE customers (
    id INT PRIMARY KEY,
    email VARCHAR(255) UNIQUE,
    phone VARCHAR(50)
);

CREATE TABLE orders (
    id INT PRIMARY KEY,
    order_number VARCHAR(50),
    customer_id INT REFERENCES customers(id)
);
`
	sqlPath := filepath.Join(tempDir, "schema.sql")
	if err := os.WriteFile(sqlPath, []byte(cleanSQL), 0644); err != nil {
		t.Fatalf("Error escribiendo archivo SQL: %v", err)
	}

	violations := CheckDataUniqueness(tempDir)
	if len(violations) != 0 {
		t.Fatalf("Se esperaba 0 violaciones en esquema normalizado con FK, pero se obtuvieron: %d (%+v)", len(violations), violations)
	}
}

func TestCheckDataUniqueness_AllowsWithADRJustification(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "qdd_ssot_adr_*")
	if err != nil {
		t.Fatalf("Error creando directorio temporal: %v", err)
	}
	defer os.RemoveAll(tempDir)

	adrDir := filepath.Join(tempDir, "docs", "adr")
	if err := os.MkdirAll(adrDir, 0755); err != nil {
		t.Fatalf("Error creando docs/adr: %v", err)
	}

	adrContent := `
# ADR-005: CQRS Denormalization for Invoices
- Decision: We justify intentional denormalization of customer_email in immutable invoice snapshots.
`
	if err := os.WriteFile(filepath.Join(adrDir, "ADR-005.md"), []byte(adrContent), 0644); err != nil {
		t.Fatalf("Error escribiendo ADR: %v", err)
	}

	denormalizedSQL := `
CREATE TABLE invoices (
    id INT PRIMARY KEY,
    customer_email VARCHAR(255)
);
`
	if err := os.WriteFile(filepath.Join(tempDir, "invoices.sql"), []byte(denormalizedSQL), 0644); err != nil {
		t.Fatalf("Error escribiendo SQL: %v", err)
	}

	violations := CheckDataUniqueness(tempDir)
	if len(violations) != 0 {
		t.Fatalf("Se esperaba 0 violaciones al existir justificación explícita en docs/adr/, pero se obtuvieron: %d", len(violations))
	}
}

func TestCheckDataUniqueness_AllowsInlineADRComment(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "qdd_ssot_inline_*")
	if err != nil {
		t.Fatalf("Error creando directorio temporal: %v", err)
	}
	defer os.RemoveAll(tempDir)

	inlineSQL := `
-- ADR-JUSTIFIED: Historical immutable snapshot
CREATE TABLE order_snapshots (
    id INT PRIMARY KEY,
    customer_email VARCHAR(255)
);
`
	if err := os.WriteFile(filepath.Join(tempDir, "snapshots.sql"), []byte(inlineSQL), 0644); err != nil {
		t.Fatalf("Error escribiendo SQL: %v", err)
	}

	violations := CheckDataUniqueness(tempDir)
	if len(violations) != 0 {
		t.Fatalf("Se esperaba 0 violaciones con comentario ADR-JUSTIFIED, pero se obtuvieron: %d", len(violations))
	}
}
