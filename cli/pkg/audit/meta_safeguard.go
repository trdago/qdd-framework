package audit

import (
	"os"
	"path/filepath"
	"time"
)

// RunMetaSafeguardCheck verifica que el binario local no sea más antiguo
// que los archivos de configuración core.
func RunMetaSafeguardCheck(cwd string) []Violation {
	var violations []Violation

	if !isMetaRepository(cwd) {
		return violations
	}

	binTime, err := getBinaryTime(cwd)
	if err != nil {
		return []Violation{{
			Category:    "GOVERNANCE",
			RuleID:      "META-01-UNCOMPILED-CORE",
			Description: "Binario local 'qdd' no encontrado en repo de framework. Ejecuta 'make build'.",
			File:        "qdd",
		}}
	}

	return scanCoreModifications(cwd, binTime)
}

func isMetaRepository(cwd string) bool {
	cliMainPath := filepath.Join(cwd, "cli", "main.go")
	_, err := os.Stat(cliMainPath)
	return err == nil
}

func getBinaryTime(cwd string) (time.Time, error) {
	binaryPath := filepath.Join(cwd, "qdd")
	binInfo, err := os.Stat(binaryPath)
	if err != nil {
		return time.Time{}, err
	}
	return binInfo.ModTime(), nil
}

func scanCoreModifications(cwd string, binTime time.Time) []Violation {
	var violations []Violation
	coreDir := filepath.Join(cwd, ".qdd", "core")

	filepath.WalkDir(coreDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		if info.ModTime().After(binTime) {
			violations = append(violations, Violation{
				Category:    "GOVERNANCE",
				RuleID:      "META-01-UNCOMPILED-CORE",
				Description: "Meta-Development Safeguard: Archivo core es más reciente que el binario compilado. Ejecuta 'make build'.",
				File:        path,
			})
		}
		
		return nil
	})
	return violations
}
