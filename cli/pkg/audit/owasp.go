package audit

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// RunOwaspCheck escanea estáticamente (AST) el código Go en busca de vulnerabilidades OWASP.
func RunOwaspCheck(cwd string) []Violation {
	var violations []Violation
	fset := token.NewFileSet()

	filepath.WalkDir(cwd, func(path string, d os.DirEntry, err error) error {
		if isIgnoredOwaspPath(path, d, err) {
			return nil
		}
		
		node, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil
		}

		checkOwaspNode(path, node, fset, &violations)
		return nil
	})
	
	return violations
}

func isIgnoredOwaspPath(path string, d os.DirEntry, err error) bool {
	if err != nil || d.IsDir() {
		return true
	}
	if !strings.HasSuffix(d.Name(), ".go") {
		return true
	}
	return isIgnoredOwaspDir(path)
}

func isIgnoredOwaspDir(path string) bool {
	return strings.Contains(path, "node_modules") || strings.Contains(path, ".git")
}

func checkOwaspNode(path string, node *ast.File, fset *token.FileSet, violations *[]Violation) {
	ast.Inspect(node, func(n ast.Node) bool {
		checkSqlInjection(path, n, fset, violations)
		checkHardcodedSecrets(path, n, fset, violations)
		return true
	})
}

func checkSqlInjection(path string, n ast.Node, fset *token.FileSet, violations *[]Violation) {
	call, ok := n.(*ast.CallExpr)
	if !ok {
		return
	}
	
	if !isTargetSqlFunc(call) {
		return
	}

	for _, arg := range call.Args {
		checkSqlArg(path, n, arg, fset, violations)
	}
}

func isTargetSqlFunc(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	
	funcName := sel.Sel.Name
	return funcName == "Query" || funcName == "Exec" || funcName == "QueryRow"
}

func checkSqlArg(path string, n ast.Node, arg ast.Expr, fset *token.FileSet, violations *[]Violation) {
	if isSprintfCall(arg) {
		addInjectionViolation(path, n, fset, violations, "Posible SQL Injection detectada. Uso de Sprintf dentro de una query no parametrizada.")
	}

	if _, isBin := arg.(*ast.BinaryExpr); isBin {
		addInjectionViolation(path, n, fset, violations, "Posible SQL Injection detectada. Concatenación de strings insegura dentro de la query.")
	}
}

func addInjectionViolation(path string, n ast.Node, fset *token.FileSet, violations *[]Violation, description string) {
	pos := fset.Position(n.Pos())
	*violations = append(*violations, Violation{
		Category:    "OWASP",
		RuleID:      "OWASP-A03-INJECTION",
		Description: description,
		File:        path,
		Line:        pos.Line,
	})
}

func isSprintfCall(arg ast.Expr) bool {
	argCall, isCall := arg.(*ast.CallExpr)
	if !isCall {
		return false
	}
	argSel, isSel := argCall.Fun.(*ast.SelectorExpr)
	if !isSel {
		return false
	}
	return argSel.Sel.Name == "Sprintf"
}

func checkHardcodedSecrets(path string, n ast.Node, fset *token.FileSet, violations *[]Violation) {
	assign, ok := n.(*ast.AssignStmt)
	if !ok {
		return
	}
	for i, lhs := range assign.Lhs {
		checkLhsForSecret(path, n, assign, i, lhs, fset, violations)
	}
}

func checkLhsForSecret(path string, n ast.Node, assign *ast.AssignStmt, i int, lhs ast.Expr, fset *token.FileSet, violations *[]Violation) {
	ident, isIdent := lhs.(*ast.Ident)
	if !isIdent {
		return
	}
	name := strings.ToLower(ident.Name)
	if isSecretName(name) {
		checkSecretAssignment(path, n, assign, i, ident.Name, fset, violations)
	}
}

func isSecretName(name string) bool {
	return strings.Contains(name, "password") || strings.Contains(name, "secret") || strings.Contains(name, "token") || strings.Contains(name, "api_key")
}

func checkSecretAssignment(path string, n ast.Node, assign *ast.AssignStmt, i int, varName string, fset *token.FileSet, violations *[]Violation) {
	if i >= len(assign.Rhs) {
		return
	}
	
	if !isPlaintextStringLit(assign.Rhs[i]) {
		return
	}

	pos := fset.Position(n.Pos())
	*violations = append(*violations, Violation{
		Category:    "OWASP",
		RuleID:      "OWASP-A02-CRYPTOGRAPHY",
		Description: "Dato sensible hardcodeado en texto plano detectado en la variable '" + varName + "'.",
		File:        path,
		Line:        pos.Line,
	})
}

func isPlaintextStringLit(rhs ast.Expr) bool {
	lit, isLit := rhs.(*ast.BasicLit)
	if !isLit || lit.Kind != token.STRING {
		return false
	}
	return lit.Value != `""` && lit.Value != "``"
}
