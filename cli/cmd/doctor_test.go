package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunDoctorCheck_FailsWhenIncomplete verifica que el doctor falle si el entorno
// no está completamente preparado, cumpliendo con la regla Zero-Else (sin else en la implementación).
func TestRunDoctorCheck_FailsWhenIncomplete(t *testing.T) {
	tempDir := t.TempDir()

	qddDir := filepath.Join(tempDir, ".qdd")
	if err := os.Mkdir(qddDir, 0755); err != nil {
		t.Fatalf("Error creando directorio temporal: %v", err)
	}

	success, _ := RunDoctorCheck(tempDir, false)
	if success {
		t.Errorf("🚨 Regla violada: RunDoctorCheck debía fallar porque faltan archivos críticos (config.yaml, state.json, etc.), pero retornó verdadero.")
		return
	}

	validateDoctorReportContains(t, qddDir, "CRITICAL_FAILURES")
}

func TestRunDoctorCheck_SucceedsWhenComplete(t *testing.T) {
	tempDir := t.TempDir()

	qddDir := filepath.Join(tempDir, ".qdd")
	_ = os.Mkdir(qddDir, 0755)
	
	_ = os.WriteFile(filepath.Join(qddDir, "config.yaml"), []byte(""), 0644)
	_ = os.WriteFile(filepath.Join(qddDir, "state.json"), []byte(""), 0644)
	
	cursorDir := filepath.Join(tempDir, ".cursor")
	_ = os.Mkdir(cursorDir, 0755)
	_ = os.WriteFile(filepath.Join(cursorDir, "mcp.json"), []byte(""), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, ".clauderc"), []byte(""), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, ".antigravityrules"), []byte(""), 0644)

	success, _ := RunDoctorCheck(tempDir, false)
	if !success {
		t.Errorf("🚨 Regla violada: RunDoctorCheck falló a pesar de que el entorno estaba completo.")
		return
	}

	validateDoctorReportContains(t, qddDir, "HEALTHY")
}

func validateDoctorReportContains(t *testing.T, qddDir, expectedStatus string) {
	evidenceDir := filepath.Join(qddDir, "project", "evidence", "doctor")
	files, err := os.ReadDir(evidenceDir)
	if err != nil || len(files) == 0 {
		t.Errorf("🚨 Regla violada: RunDoctorCheck no generó el reporte de evidencia en %s", evidenceDir)
		return
	}

	reportContent, _ := os.ReadFile(filepath.Join(evidenceDir, files[0].Name()))
	if !strings.Contains(string(reportContent), expectedStatus) {
		t.Errorf("El reporte no marcó '%s'. Contenido:\n%s", expectedStatus, reportContent)
	}
}
