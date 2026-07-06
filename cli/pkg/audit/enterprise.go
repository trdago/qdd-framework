package audit

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// RunEnterpriseCheck evalúa el cumplimiento de certificaciones nivel 5 (Monolith/Enterprise-Grade)
func RunEnterpriseCheck(cwd string) []Violation {
	var violations []Violation
	fset := token.NewFileSet()

	// 1. Mandatory ADR Check (CERT-030)
	adrPath := filepath.Join(cwd, "docs", "adr")
	if info, err := os.Stat(adrPath); err != nil || !info.IsDir() {
		violations = append(violations, Violation{
			Category:    "ARCHITECTURE",
			RuleID:      "CERT-030-MANDATORY-ADR",
			Description: "Enterprise-Grade: Faltan Architecture Decision Records. Debe existir el directorio docs/adr/.",
			File:        "docs/adr",
			Line:        0,
		})
	}

	filepath.WalkDir(cwd, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".go") || strings.Contains(path, "node_modules") || strings.Contains(path, ".git") {
			return nil
		}

		node, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err == nil {
			// 3. Domain Isolation Check (CERT-032)
			// Si el archivo pertenece al núcleo del dominio (e.g., internal/domain), no puede importar infra
			if strings.Contains(path, "/domain/") || strings.Contains(path, "/core/") {
				for _, imp := range node.Imports {
					importPath := strings.Trim(imp.Path.Value, "\"")
					// Regla: Prohibir imports de bases de datos, web frameworks o librerías que no sean stdlib o del propio proyecto de manera abstracta
					if strings.Contains(importPath, "database/sql") || strings.Contains(importPath, "github.com/labstack/echo") || strings.Contains(importPath, "gorm.io") || strings.Contains(importPath, "net/http") {
						pos := fset.Position(imp.Pos())
						violations = append(violations, Violation{
							Category:    "ARCHITECTURE",
							RuleID:      "CERT-032-DOMAIN-ISOLATION",
							Description: "Enterprise-Grade: Aislación del dominio violada. " + importPath + " no debe ser importado en la capa de dominio.",
							File:        path,
							Line:        pos.Line,
						})
					}
				}
			}
		}

		// Parse complete file for AST analysis
		nodeFull, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil
		}

		// 2. Cyclomatic Complexity Check (CERT-031)
		ast.Inspect(nodeFull, func(n ast.Node) bool {
			if fn, ok := n.(*ast.FuncDecl); ok {
				complexity := 1 // Base complexity
				
				ast.Inspect(fn.Body, func(bn ast.Node) bool {
					switch bn.(type) {
					case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.CaseClause, *ast.CommClause:
						complexity++
					case *ast.BinaryExpr:
						be := bn.(*ast.BinaryExpr)
						if be.Op == token.LAND || be.Op == token.LOR {
							complexity++
						}
					}
					return true
				})

				if complexity > 5 {
					pos := fset.Position(fn.Pos())
					violations = append(violations, Violation{
						Category:    "CLEAN-CODE",
						RuleID:      "CERT-031-CYCLOMATIC-LIMIT",
						Description: fmt.Sprintf("Enterprise-Grade: Complejidad ciclomática extrema detectada (%d > 5) en función %s. Refactorice dividiendo la lógica.", complexity, fn.Name.Name),
						File:        path,
						Line:        pos.Line,
					})
				}
			}
			return true
		})
		return nil
	})

	return violations
}
