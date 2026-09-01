package audit

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// CheckDataUniqueness audita el código y esquemas para garantizar el principio
// de Unicidad del Dato (Single Source of Truth - SSOT), previniendo la duplicación
// redundante de datos en distintas tablas sin normalizar ni justificación (ADR).
func CheckDataUniqueness(cwd string) []Violation {
	var violations []Violation
	adrs := loadKnownADRJustifications(cwd)

	redundantFieldRegex := regexp.MustCompile(`(?i)(customer_email|user_email|customer_phone|user_phone|customer_address|user_address)\s+(VARCHAR|TEXT|CHAR|VARCHAR2)`)
	hasForeignKeyRegex := regexp.MustCompile(`(?i)(FOREIGN\s+KEY|REFERENCES\s+[a-zA-Z0-9_]+)`)

	err := filepath.WalkDir(cwd, func(path string, d fs.DirEntry, err error) error {
		return processDataUniquenessPath(path, d, err, redundantFieldRegex, hasForeignKeyRegex, adrs, &violations)
	})

	if err != nil {
		fmt.Printf("Error auditando Unicidad del Dato en %v: %v\n", cwd, err)
	}

	return violations
}

func processDataUniquenessPath(path string, d fs.DirEntry, err error, redundantRegex, fkRegex *regexp.Regexp, adrs map[string]bool, violations *[]Violation) error {
	if isIgnoredDataUniquenessPath(path, d, err) {
		return skipIgnoredDir(d)
	}

	if !isCheckableSchemaFile(path, d) {
		return nil
	}

	checkFileForDataUniqueness(path, redundantRegex, fkRegex, adrs, violations)
	return nil
}

func skipIgnoredDir(d fs.DirEntry) error {
	if d != nil && d.IsDir() {
		return filepath.SkipDir
	}
	return nil
}

func isIgnoredDataUniquenessPath(path string, d fs.DirEntry, err error) bool {
	if err != nil {
		return true
	}
	return d.IsDir() && isIgnoredDirName(d.Name())
}

func isIgnoredDirName(name string) bool {
	return name == "vendor" || name == ".git" || name == ".qdd" || name == "node_modules"
}

func isCheckableSchemaFile(path string, d fs.DirEntry) bool {
	if d.IsDir() {
		return false
	}
	ext := filepath.Ext(path)
	return ext == ".sql" || ext == ".ddl" || strings.HasSuffix(path, "_schema.go") || strings.HasSuffix(path, "_models.go")
}

func loadKnownADRJustifications(cwd string) map[string]bool {
	adrs := make(map[string]bool)
	adrDir := filepath.Join(cwd, "docs", "adr")
	entries, err := os.ReadDir(adrDir)
	if err != nil {
		return adrs
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		content, readErr := os.ReadFile(filepath.Join(adrDir, entry.Name()))
		if readErr != nil {
			continue
		}

		str := strings.ToLower(string(content))
		if strings.Contains(str, "denormalization") || strings.Contains(str, "desnormalizacion") || strings.Contains(str, "cqrs") || strings.Contains(str, "snapshot") {
			adrs[strings.ToLower(entry.Name())] = true
			adrs["global_denormalization_justified"] = true
		}
	}

	return adrs
}

func checkFileForDataUniqueness(path string, redundantRegex, fkRegex *regexp.Regexp, adrs map[string]bool, violations *[]Violation) {
	content, readErr := os.ReadFile(path)
	if readErr != nil {
		return
	}

	strContent := string(content)
	if isADRJustifiedInFile(strContent, adrs) {
		return
	}

	lines := strings.Split(strContent, "\n")
	hasFK := fkRegex.MatchString(strContent)

	for i, line := range lines {
		if !redundantRegex.MatchString(line) {
			continue
		}

		if hasFK && strings.Contains(strings.ToLower(line), "references") {
			continue
		}

		*violations = append(*violations, Violation{
			Category:    "DATABASE",
			RuleID:      "CERT-034-DATA-UNIQUENESS",
			Description: "Unicidad del Dato (SSOT) violada: Atributo de identidad duplicado directamente en tabla auxiliar. Normalice o use clave foránea (FK). Para desnormalizaciones intencionales (CQRS/Snapshot), registre un ADR en docs/adr/.",
			File:        path,
			Line:        i + 1,
		})
	}
}

func isADRJustifiedInFile(content string, adrs map[string]bool) bool {
	if adrs["global_denormalization_justified"] {
		return true
	}
	lower := strings.ToLower(content)
	return strings.Contains(lower, "adr-justified") || strings.Contains(lower, "ssot-exception")
}
