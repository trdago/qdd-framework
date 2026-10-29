package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/qdd-framework/qdd/pkg/audit"
	"github.com/spf13/cobra"
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Revisa el proyecto contra las reglas de QDD (pruebas, OWASP, código limpio...)",
	Long: `Evalúa el código contra las reglas de gobernanza del QDD Framework.
Si algún problema bloquea el gate, termina con código distinto de cero (falla en CI).
En adopción gradual (lo que deja 'qdd init' en un proyecto nuevo) las prácticas de
madurez —E2E, ADR, Conventional Commits— se informan como recomendaciones sin bloquear.
No genera ni guarda el Certificado Histórico (eso es tarea de 'qdd certify').`,
	Run: runAudit,
}

func init() {
	rootCmd.AddCommand(auditCmd)
}

func runAudit(cmd *cobra.Command, args []string) {
	fmt.Println("[+] Iniciando Auditoría Técnica de QDD...")

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("[✗] Error obteniendo directorio actual: %v\n", err)
		os.Exit(1)
	}

	engine := audit.NewEngine(cwd)
	fmt.Println("[+] Ejecutando motor de auditoría...")
	resultado := engine.Evaluar()

	fmt.Print(formatAuditResult(resultado))
	if len(audit.Bloqueantes(resultado.Violaciones)) > 0 {
		os.Exit(1)
	}
}

// formatAuditResult separa lo que bloquea el gate de las recomendaciones, para
// que un proyecto sano no parezca roto (FND-040), y muestra el puntaje junto a
// su cobertura: qué principios se pudieron revisar (RFC 0005 §3.4). Lo usan
// audit, certify y MCP.
func formatAuditResult(r audit.Resultado) string {
	violations := r.Violaciones
	bloqueantes := audit.Bloqueantes(violations)
	out := ""
	if len(bloqueantes) > 0 {
		out += fmt.Sprintf("[✗] %d problema(s) bloquean el gate:\n", len(bloqueantes))
		out += listarViolaciones(bloqueantes)
	}
	if len(bloqueantes) == 0 {
		out += "[✓] Auditoría aprobada: ningún problema bloquea el gate.\n"
	}
	out += formatRecomendaciones(audit.Recomendaciones(violations))
	out += formatCobertura(r.Cobertura)
	out += fmt.Sprintf("[i] Puntaje QDD: %d/100 (%s)\n", audit.Score(violations), resumenDeCobertura(r.Cobertura))
	if len(bloqueantes) > 0 {
		out += "\nRepara los problemas que bloquean para poder certificar.\n"
	}
	return out
}

func resumenDeCobertura(c audit.Cobertura) string {
	return fmt.Sprintf("cobertura: %d de %d principios evaluados en todo el proyecto; %d parcial, %d no evaluado, %d desactivado",
		c.Evaluados, c.Aplicables, c.Parciales, c.NoEvaluados, c.Desactivados)
}

var ordenDeCategoria = map[string]int{"seguridad": 0, "correccion": 1, "auditabilidad": 2, "madurez": 3}

var marcaDeEstado = map[audit.EstadoCobertura]string{
	audit.Parcial:     "~",
	audit.NoEvaluado:  "?",
	audit.Desactivado: "-",
}

// formatCobertura lista los principios que no se revisaron en todo el
// proyecto, primero los de seguridad y corrección. Un principio sin
// violaciones que no se evaluó no está aprobado: no se sabe.
func formatCobertura(c audit.Cobertura) string {
	var pendientes []audit.CoberturaPrincipio
	for _, cp := range c.Principios {
		if cp.Estado != audit.Evaluado {
			pendientes = append(pendientes, cp)
		}
	}
	if len(pendientes) == 0 {
		return ""
	}
	sort.SliceStable(pendientes, func(i, j int) bool {
		return ordenDeCategoria[pendientes[i].Principio.Categoria] < ordenDeCategoria[pendientes[j].Principio.Categoria]
	})
	out := "[i] Lo que la auditoría no revisó (~ parcial, ? no evaluado, - desactivado en .qdd/policies.yaml):\n"
	for _, cp := range pendientes {
		out += fmt.Sprintf("    %s %-26s %-13s %s\n", marcaDeEstado[cp.Estado], cp.Principio.ID, cp.Principio.Categoria, detalleDeCobertura(cp))
	}
	return out
}

// coberturaDeCategoria resume una categoría para el certificado: certify no
// emite "certificado OWASP", dice cuántos principios de seguridad revisó.
func coberturaDeCategoria(c audit.Cobertura, categoria string) string {
	evaluados, aplicables := 0, 0
	for _, cp := range c.Principios {
		if cp.Principio.Categoria != categoria || cp.Estado == audit.Desactivado {
			continue
		}
		aplicables++
		if cp.Estado == audit.Evaluado {
			evaluados++
		}
	}
	return fmt.Sprintf("%d de %d principios evaluados en todo el proyecto", evaluados, aplicables)
}

func detalleDeCobertura(cp audit.CoberturaPrincipio) string {
	if cp.Estado == audit.Desactivado {
		return "desactivado (" + strings.Join(cp.Principio.Politicas, ": false, ") + ": false)"
	}
	var partes []string
	if len(cp.Evaluadas) > 0 {
		partes = append(partes, "evaluado en "+enumerar(cp.Evaluadas))
	}
	var motivos []string
	porMotivo := map[string][]string{}
	for _, u := range cp.NoEvaluadas {
		if porMotivo[u.Motivo] == nil {
			motivos = append(motivos, u.Motivo)
		}
		porMotivo[u.Motivo] = append(porMotivo[u.Motivo], u.Ruta)
	}
	for _, m := range motivos {
		partes = append(partes, fmt.Sprintf("no evaluado en %s: %s", unidades(porMotivo[m]), m))
	}
	return strings.Join(partes, "; ")
}

func unidades(rutas []string) string {
	if len(rutas) == 1 {
		return rutas[0]
	}
	return fmt.Sprintf("%d unidades (%s)", len(rutas), enumerar(rutas))
}

// enumerar muestra hasta tres elementos y cuántos más hay.
func enumerar(xs []string) string {
	if len(xs) <= 3 {
		return strings.Join(xs, ", ")
	}
	return fmt.Sprintf("%s y %d más", strings.Join(xs[:3], ", "), len(xs)-3)
}

func formatRecomendaciones(recomendaciones []audit.Violation) string {
	if len(recomendaciones) == 0 {
		return ""
	}
	out := fmt.Sprintf("[i] %d recomendación(es) para madurar el proyecto (no bloquean; para exigirlas: adopcion: estricta en .qdd/policies.yaml):\n", len(recomendaciones))
	return out + listarViolaciones(recomendaciones)
}

func listarViolaciones(violations []audit.Violation) string {
	out := ""
	for _, v := range violations {
		out += fmt.Sprintf("    - [%s] %s\n", v.Category, v.Format())
	}
	return out
}
