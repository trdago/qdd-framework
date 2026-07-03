package audit

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// RunCleanCodeCheck evalúa el cumplimiento de las reglas de Clean Code,
// en particular la directiva CLEAN-01-NO-ELSE (Zero usage of 'else').
func RunCleanCodeCheck(cwd string) []Violation {
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
			if ifStmt, ok := n.(*ast.IfStmt); ok {
				if ifStmt.Else != nil {
					// Ignoramos 'else if', solo penalizamos el 'else' final de caída.
					if _, isElseIf := ifStmt.Else.(*ast.IfStmt); !isElseIf {
						pos := fset.Position(ifStmt.Else.Pos())
						violations = append(violations, Violation{
							Category:    "CLEAN-CODE",
							RuleID:      "CLEAN-01-NO-ELSE",
							Description: "Uso de 'else' detectado. QDD exige Early Returns (Cláusulas de Guarda).",
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
