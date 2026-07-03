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
		// Analizamos solo archivos .go y saltamos carpetas comunes no relevantes
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".go") || strings.Contains(path, "node_modules") || strings.Contains(path, ".git") {
			return nil
		}
		
		node, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil
		}

		ast.Inspect(node, func(n ast.Node) bool {
			// OWASP-A03-INJECTION: Búsqueda de inyecciones SQL
			if call, ok := n.(*ast.CallExpr); ok {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
					funcName := sel.Sel.Name
					// Buscamos métodos comunes de base de datos
					if funcName == "Query" || funcName == "Exec" || funcName == "QueryRow" {
						for _, arg := range call.Args {
							// Caso 1: uso de fmt.Sprintf interno
							if argCall, isCall := arg.(*ast.CallExpr); isCall {
								if argSel, isSel := argCall.Fun.(*ast.SelectorExpr); isSel && argSel.Sel.Name == "Sprintf" {
									pos := fset.Position(n.Pos())
									violations = append(violations, Violation{
										Category:    "OWASP",
										RuleID:      "OWASP-A03-INJECTION",
										Description: "Posible SQL Injection detectada. Uso de Sprintf dentro de una query no parametrizada.",
										File:        path,
										Line:        pos.Line,
									})
								}
							}
							// Caso 2: concatenación de strings con +
							if _, isBin := arg.(*ast.BinaryExpr); isBin {
								pos := fset.Position(n.Pos())
								violations = append(violations, Violation{
										Category:    "OWASP",
									RuleID:      "OWASP-A03-INJECTION",
									Description: "Posible SQL Injection detectada. Concatenación de strings insegura dentro de la query.",
									File:        path,
									Line:        pos.Line,
								})
							}
						}
					}
				}
			}

			// OWASP-A02-CRYPTOGRAPHY: Secretos y contraseñas hardcodeados
			if assign, ok := n.(*ast.AssignStmt); ok {
				for i, lhs := range assign.Lhs {
					if ident, isIdent := lhs.(*ast.Ident); isIdent {
						name := strings.ToLower(ident.Name)
						// Detectamos variables sospechosas de ser secretos
						if strings.Contains(name, "password") || strings.Contains(name, "secret") || strings.Contains(name, "token") || strings.Contains(name, "api_key") {
							if i < len(assign.Rhs) {
								if lit, isLit := assign.Rhs[i].(*ast.BasicLit); isLit && lit.Kind == token.STRING {
									// Ignoramos strings vacíos
									if lit.Value != `""` && lit.Value != "``" {
										pos := fset.Position(n.Pos())
										violations = append(violations, Violation{
											Category:    "OWASP",
											RuleID:      "OWASP-A02-CRYPTOGRAPHY",
											Description: "Dato sensible hardcodeado en texto plano detectado en la variable '" + ident.Name + "'.",
											File:        path,
											Line:        pos.Line,
										})
									}
								}
							}
						}
					}
				}
			}
			return true
		})
		return nil
	})
	
	return violations
}
