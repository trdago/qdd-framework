package audit

import (
	"bufio"
	"os"
	"strings"
)

// FilterIgnoredViolations revisa la línea anterior de cada violación en el código fuente
// para ver si tiene un comentario de ignorado, ej: // qdd:ignore CERT-031
func FilterIgnoredViolations(violations []Violation) []Violation {
	var filtered []Violation

	for _, v := range violations {
		if v.File == "" || v.Line <= 1 {
			filtered = append(filtered, v)
			continue
		}

		// Leer el archivo para buscar la línea anterior
		file, err := os.Open(v.File)
		if err != nil {
			filtered = append(filtered, v)
			continue
		}

		scanner := bufio.NewScanner(file)
		currentLine := 1
		ignored := false
		for scanner.Scan() {
			if currentLine >= v.Line-5 && currentLine < v.Line {
				lineText := scanner.Text()
				if strings.Contains(lineText, "// qdd:ignore "+v.RuleID) {
					ignored = true
					break
				}
			}
			if currentLine >= v.Line {
				break
			}
			currentLine++
		}
		file.Close()

		if !ignored {
			filtered = append(filtered, v)
		}
	}

	return filtered
}
