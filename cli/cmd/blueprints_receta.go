package cmd

import (
	"fmt"
	"strings"

	"github.com/qdd-framework/qdd/pkg/blueprints"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'

var blueprintDistillCmd = &cobra.Command{
	Use:   "distill <modulo> --id <blueprint>",
	Short: "Destila un blueprint desde la ficha aprobada de un módulo y dice qué preguntarle a la persona para delimitarlo",
	Args:  cobra.ExactArgs(1),
	Run:   runBlueprintDistill,
}

var blueprintCertifyCmd = &cobra.Command{
	Use:   "certify <id|carpeta> [--verificar]",
	Short: "Certificación ciega: prepara una carpeta por variante con solo la receta y, con --verificar, prueba lo construido",
	Args:  cobra.ExactArgs(1),
	Run:   runBlueprintCertify,
}

var (
	blueprintDestino   string
	blueprintVerificar bool
)

func init() {
	blueprints.VersionQDD = strings.TrimSpace(embeddedVersion)
	blueprintDistillCmd.Flags().StringVar(&blueprintDestino, "id", "", "id del blueprint nuevo")
	blueprintDistillCmd.Flags().StringVar(&blueprintOrigen, "origen", "", "fuente (repositorio) del blueprint nuevo; por defecto, la de la organización")
	blueprintCertifyCmd.Flags().BoolVar(&blueprintVerificar, "verificar", false, "corre las pruebas de cada variante construida y deja certificacion.yaml")
	blueprintCmd.AddCommand(blueprintDistillCmd, blueprintCertifyCmd)
}

func runBlueprintDistill(cmd *cobra.Command, args []string) {
	if strings.TrimSpace(blueprintDestino) == "" {
		salirCon("indica el id del blueprint nuevo con --id")
	}
	d, err := blueprints.Destilar(modulesRoot(), args[0], blueprintDestino, blueprintOrigen)
	if err != nil {
		salirCon("%v", err)
	}
	fmt.Print(textoDeDestilado(d))
}

// textoDeDestilado dice dónde quedó el borrador, cómo está y qué preguntarle a la persona.
func textoDeDestilado(d blueprints.Destilado) string {
	var s strings.Builder
	fmt.Fprintf(&s, "Borrador en %s (rama %s).\n%s\n- %s\n", d.Aporte.Carpeta, d.Aporte.Rama, blueprints.ReglasDeReceta, blueprints.LineaCalidad(d.Calidad))
	s.WriteString(textoDeEntrevista(d.Calidad))
	s.WriteString(d.Siguiente + "\n")
	return s.String()
}

func textoDeEntrevista(c blueprints.Calidad) string {
	preguntas := blueprints.Entrevista(c)
	if len(preguntas) == 0 {
		return ""
	}
	return "Pregúntale a la persona (no lo respondas tú):\n  ❓ " + strings.Join(preguntas, "\n  ❓ ") + "\n"
}

func runBlueprintCertify(cmd *cobra.Command, args []string) {
	texto, aprobada, err := certificar(modulesRoot(), args[0], blueprintVerificar)
	if err != nil {
		salirCon("%v", err)
	}
	fmt.Print(texto)
	if blueprintVerificar && !aprobada {
		salirCon("la certificación ciega no pasa")
	}
}

// certificar prepara las variantes o verifica lo construido, en estilo chat.
func certificar(root, pedido string, verificar bool) (string, bool, error) {
	b, err := blueprints.CarpetaDeAporte(root, pedido)
	if err != nil {
		return "", false, err
	}
	if verificar {
		c, err := blueprints.VerificarCertificacion(b)
		texto := blueprints.LineaCertificacion(c) + "\n"
		if s := blueprints.SupuestosDe(c); len(s) > 0 {
			texto += "Lo que la receta no decía y se supuso. Pregúntale a la persona, uno por uno, si se fija en la receta (decisión, parámetro o constante) o queda libre a quien construye:\n  ❓ " + strings.Join(s, "\n  ❓ ") + "\n"
		}
		return texto, c.Aprobada, err
	}
	k, err := blueprints.PrepararCertificacion(b)
	if err != nil {
		return "", false, err
	}
	var s strings.Builder
	fmt.Fprintf(&s, "Certificación ciega de %s: %d variante(s), cada una con solo %s.\n", k.Ref, len(k.Variantes), blueprints.RecetaFile)
	for _, v := range k.Variantes {
		fmt.Fprintf(&s, "- %s · %s · %d escenarios · %s\n", v.Nombre, respuestasEnLinea(v.Respuestas), len(v.Escenarios), v.Carpeta)
	}
	s.WriteString("Por cada variante, lanza un agente NUEVO (sin esta conversación ni el código original) con solo su carpeta: «construye lo que dice RECETA.md». Mejor en lenguajes distintos. Después, verifica (qdd blueprint certify " + pedido + " --verificar o certificar=verificar).\n")
	return s.String(), false, nil
}

func respuestasEnLinea(r map[string]string) string {
	partes := make([]string, 0, len(r))
	for _, k := range sortedKeysDe(r) {
		partes = append(partes, k+"="+r[k])
	}
	return strings.Join(partes, " ")
}

func sortedKeysDe(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}
