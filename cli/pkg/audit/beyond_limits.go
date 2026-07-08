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
		if shouldSkipBeyondLimitsPath(path, d, err) {
			return nil
		}

		node, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil
		}

		checkBeyondLimitsNode(path, node, fset, &violations)
		return nil
	})

	return violations
}

func shouldSkipBeyondLimitsPath(path string, d os.DirEntry, err error) bool {
	if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".go") {
		return true
	}
	return strings.Contains(path, "node_modules") || strings.Contains(path, ".git")
}

func checkBeyondLimitsNode(path string, node *ast.File, fset *token.FileSet, violations *[]Violation) {
	ast.Inspect(node, func(n ast.Node) bool {
		callExpr, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		
		checkPanicUsage(path, callExpr, fset, violations)
		checkContextlessDBQuery(path, callExpr, fset, violations)
		
		return true
	})
}

func checkPanicUsage(path string, callExpr *ast.CallExpr, fset *token.FileSet, violations *[]Violation) {
	ident, ok := callExpr.Fun.(*ast.Ident)
	if !ok {
		return
	}
	
	if ident.Name == "panic" {
		pos := fset.Position(ident.Pos())
		*violations = append(*violations, Violation{
			Category:    "RELIABILITY",
			RuleID:      "CERT-020-ZERO-PANIC",
			Description: "NASA-Grade: Uso de panic() está estrictamente prohibido. Maneje el error matemáticamente (Graceful Degradation).",
			File:        path,
			Line:        pos.Line,
		})
	}
}

func checkContextlessDBQuery(path string, callExpr *ast.CallExpr, fset *token.FileSet, violations *[]Violation) {
	selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}
	
	method := selExpr.Sel.Name
	if method == "Query" || method == "QueryRow" || method == "Exec" {
		pos := fset.Position(selExpr.Pos())
		*violations = append(*violations, Violation{
			Category:    "SECURITY",
			RuleID:      "CERT-021-CHAOS-TOLERANCE",
			Description: "Netflix-Grade: " + method + " sin Context detectado. Vulnerable a saturación de red. Use " + method + "Context.",
			File:        path,
			Line:        pos.Line,
		})
	}
}
