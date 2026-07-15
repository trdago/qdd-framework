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
		if isIgnoredPath(path, d, err) {
			return nil
		}

		node, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil
		}

		checkNodeForElse(node, fset, path, &violations)
		if !strings.HasSuffix(d.Name(), "_test.go") && !strings.Contains(filepath.ToSlash(path), "/pkg/goldenset/") {
			checkNodeForTestCode(node, fset, path, &violations)
		}
		return nil
	})

	return violations
}

func isIgnoredPath(path string, d os.DirEntry, err error) bool {
	if err != nil || d.IsDir() {
		return true
	}
	if !strings.HasSuffix(d.Name(), ".go") {
		return true
	}
	return isIgnoredCleanCodeDir(path)
}

func isIgnoredCleanCodeDir(path string) bool {
	return strings.Contains(path, "node_modules") || strings.Contains(path, ".git")
}

func checkNodeForElse(node *ast.File, fset *token.FileSet, path string, violations *[]Violation) {
	ast.Inspect(node, func(n ast.Node) bool {
		if ifStmt, ok := n.(*ast.IfStmt); ok && ifStmt.Else != nil {
			if _, isElseIf := ifStmt.Else.(*ast.IfStmt); !isElseIf {
				pos := fset.Position(ifStmt.Else.Pos())
				*violations = append(*violations, Violation{
					Category:    "CLEAN-CODE",
					RuleID:      "CLEAN-01-NO-ELSE",
					Description: "Uso de 'else' detectado. QDD exige Early Returns (Cláusulas de Guarda).",
					File:        path,
					Line:        pos.Line,
				})
			}
		}
		return true
	})
}

func checkNodeForTestCode(node *ast.File, fset *token.FileSet, path string, violations *[]Violation) {
	checkImportsForTesting(node, fset, path, violations)
	checkASTForTesting(node, fset, path, violations)
}

func checkImportsForTesting(node *ast.File, fset *token.FileSet, path string, violations *[]Violation) {
	for _, imp := range node.Imports {
		pkgPath := strings.Trim(imp.Path.Value, "\"")
		if pkgPath == "testing" || strings.Contains(pkgPath, "mock") || strings.Contains(pkgPath, "testify") {
			pos := fset.Position(imp.Pos())
			*violations = append(*violations, Violation{
				Category:    "CLEAN-CODE",
				RuleID:      "CLEAN-02-NO-TEST-IN-PROD",
				Description: "Zero-Mocks: Importación de paquete de testing (" + pkgPath + ") detectada en código de producción.",
				File:        path,
				Line:        pos.Line,
			})
		}
	}
}

func checkASTForTesting(node *ast.File, fset *token.FileSet, path string, violations *[]Violation) {
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.TypeSpec:
			if strings.Contains(x.Name.Name, "Mock") {
				pos := fset.Position(x.Pos())
				*violations = append(*violations, Violation{
					Category:    "CLEAN-CODE",
					RuleID:      "CLEAN-02-NO-TEST-IN-PROD",
					Description: "Zero-Mocks: Estructura/Tipo con palabra 'Mock' detectada en código de producción (" + x.Name.Name + ").",
					File:        path,
					Line:        pos.Line,
				})
			}
		case *ast.FuncDecl:
			if strings.Contains(x.Name.Name, "Mock") {
				pos := fset.Position(x.Pos())
				*violations = append(*violations, Violation{
					Category:    "CLEAN-CODE",
					RuleID:      "CLEAN-02-NO-TEST-IN-PROD",
					Description: "Zero-Mocks: Función con palabra 'Mock' detectada en código de producción (" + x.Name.Name + ").",
					File:        path,
					Line:        pos.Line,
				})
			}
		}
		return true
	})
}
