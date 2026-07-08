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

	checkADR(cwd, &violations)

	filepath.WalkDir(cwd, func(path string, d os.DirEntry, err error) error {
		if isIgnoredEnterprisePath(path, d, err) {
			return nil
		}

		node, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err == nil {
			checkDomainIsolation(path, node, fset, &violations)
		}

		nodeFull, err := parser.ParseFile(fset, path, nil, 0)
		if err == nil {
			checkCyclomaticComplexity(path, nodeFull, fset, &violations)
		}
		return nil
	})

	return violations
}

func checkADR(cwd string, violations *[]Violation) {
	adrPath := filepath.Join(cwd, "docs", "adr")
	if info, err := os.Stat(adrPath); err != nil || !info.IsDir() {
		*violations = append(*violations, Violation{
			Category:    "ARCHITECTURE",
			RuleID:      "CERT-030-MANDATORY-ADR",
			Description: "Enterprise-Grade: Faltan Architecture Decision Records. Debe existir el directorio docs/adr/.",
			File:        "docs/adr",
			Line:        0,
		})
	}
}

func isIgnoredEnterprisePath(path string, d os.DirEntry, err error) bool {
	if err != nil || d.IsDir() {
		return true
	}
	if !strings.HasSuffix(d.Name(), ".go") {
		return true
	}
	return isIgnoredEnterpriseDir(path)
}

func isIgnoredEnterpriseDir(path string) bool {
	return strings.Contains(path, "node_modules") || strings.Contains(path, ".git")
}

func checkDomainIsolation(path string, node *ast.File, fset *token.FileSet, violations *[]Violation) {
	if !isDomainOrCorePath(path) {
		return
	}
	for _, imp := range node.Imports {
		checkImportForDomainIsolation(path, imp, fset, violations)
	}
}

func isDomainOrCorePath(path string) bool {
	return strings.Contains(path, "/domain/") || strings.Contains(path, "/core/")
}

func checkImportForDomainIsolation(path string, imp *ast.ImportSpec, fset *token.FileSet, violations *[]Violation) {
	importPath := strings.Trim(imp.Path.Value, "\"")
	if isForbiddenDomainImport(importPath) {
		pos := fset.Position(imp.Pos())
		addDomainIsolationViolation(path, importPath, pos.Line, violations)
	}
}

func isForbiddenDomainImport(importPath string) bool {
	return strings.Contains(importPath, "database/sql") || 
		strings.Contains(importPath, "github.com/labstack/echo") || 
		strings.Contains(importPath, "gorm.io") || 
		strings.Contains(importPath, "net/http")
}

func addDomainIsolationViolation(path, importPath string, line int, violations *[]Violation) {
	*violations = append(*violations, Violation{
		Category:    "ARCHITECTURE",
		RuleID:      "CERT-032-DOMAIN-ISOLATION",
		Description: "Enterprise-Grade: Aislación del dominio violada. " + importPath + " no debe ser importado en la capa de dominio.",
		File:        path,
		Line:        line,
	})
}

func checkCyclomaticComplexity(path string, node *ast.File, fset *token.FileSet, violations *[]Violation) {
	ast.Inspect(node, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok {
			complexity := calculateComplexity(fn)
			if complexity > 5 {
				pos := fset.Position(fn.Pos())
				*violations = append(*violations, Violation{
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
}

func calculateComplexity(fn *ast.FuncDecl) int {
	complexity := 1
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
	return complexity
}
