package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/goldenset"
)

type DoctorInput struct {
	DirsToCreate  []string `json:"dirs_to_create"`
	FilesToCreate []string `json:"files_to_create"`
	AutoFix       bool     `json:"auto_fix"`
}

type DoctorOutput struct {
	Success  bool `json:"success"`
	Failures int  `json:"failures"`
}

func TestDoctorWithGoldenSet(t *testing.T) {
	goldenset.RunSuite(t, "cli_doctor", func(in DoctorInput) (DoctorOutput, error) {
		tempDir := t.TempDir()

		for _, d := range in.DirsToCreate {
			_ = os.MkdirAll(filepath.Join(tempDir, d), 0755)
		}

		for _, f := range in.FilesToCreate {
			_ = os.MkdirAll(filepath.Dir(filepath.Join(tempDir, f)), 0755)
			_ = os.WriteFile(filepath.Join(tempDir, f), []byte(""), 0644)
		}

		success, failures := RunDoctorCheck(tempDir, in.AutoFix)

		return DoctorOutput{
			Success:  success,
			Failures: failures,
		}, nil
	})
}

// Para validaciones específicas de output visual (Evidencia generada)
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
