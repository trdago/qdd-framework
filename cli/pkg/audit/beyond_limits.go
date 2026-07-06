package audit

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// RunBeyondLimitsCheck evalúa el cumplimiento de certificaciones nivel 4 (NASA/Netflix-Grade)
func RunBeyondLimitsCheck(cwd string) []Violation {
	var violations []Violation
	fset := token.NewFileSet()

	filepath.WalkDir(cwd, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".go") || strings.Contains(path, "node_modules") || strings.Contains(path, ".git") {
			return nil
		}

		node, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil
		}

		ast.Inspect(node, func(n ast.Node) bool {
			// Buscar llamadas a funciones
			if callExpr, ok := n.(*ast.CallExpr); ok {
				if ident, ok := callExpr.Fun.(*ast.Ident); ok {
					// NASA-Grade: Zero-Panic
					if ident.Name == "panic" {
						pos := fset.Position(ident.Pos())
						violations = append(violations, Violation{
							Category:    "RELIABILITY",
							RuleID:      "CERT-020-ZERO-PANIC",
							Description: "NASA-Grade: Uso de panic() está estrictamente prohibido. Maneje el error matemáticamente (Graceful Degradation).",
							File:        path,
							Line:        pos.Line,
						})
					}
				}
				
				if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
					// Netflix-Grade: Chaos-Tolerance (Prohibir Query, QueryRow, Exec sin contexto)
					method := selExpr.Sel.Name
					if method == "Query" || method == "QueryRow" || method == "Exec" {
						pos := fset.Position(selExpr.Pos())
						violations = append(violations, Violation{
							Category:    "SECURITY",
							RuleID:      "CERT-021-CHAOS-TOLERANCE",
							Description: "Netflix-Grade: " + method + " sin Context detectado. Vulnerable a saturación de red. Use " + method + "Context.",
							File:        path,
							Line:        pos.Line,
						})
					}
				}
			}
			return true
		})
		return nil
	})

	return violations
}
