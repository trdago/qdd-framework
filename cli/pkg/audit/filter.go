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
		if shouldKeepViolation(v) {
			filtered = append(filtered, v)
		}
	}

	return filtered
}

func shouldKeepViolation(v Violation) bool {
	if v.File == "" || v.Line <= 1 {
		return true
	}

	file, err := os.Open(v.File)
	if err != nil {
		return true
	}
	defer file.Close()

	return !hasIgnoreComment(file, v)
}

func hasIgnoreComment(file *os.File, v Violation) bool {
	scanner := bufio.NewScanner(file)
	currentLine := 1
	for scanner.Scan() {
		if currentLine >= v.Line {
			break
		}
		if currentLine >= v.Line-5 {
			if strings.Contains(scanner.Text(), "// qdd:ignore "+v.RuleID) {
				return true
			}
		}
		currentLine++
	}
	return false
}
