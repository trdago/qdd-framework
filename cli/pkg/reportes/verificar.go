package reportes

import (
	"fmt"
	"regexp"
	"strings"
)

// Rule: Prohibido usar 'else'
// MEJ-087 (REP-503, propuesta de la persona): quien reporta valida. Un bug dice desde el inicio cómo lo verificará
// quien lo reporta (la prueba de aceptación de quien lo arregla), y la verificación vuelve como otro reporte que nombra
// al original: «Verificado REP-N en <versión>» o «Sigue fallando REP-N en <versión>», con la evidencia en el detalle.
// Todo viaja en el texto firmado, así que el relay no cambia; la cola de QDD lo lee (mejoras.VerificacionDe).

const marcaVerifico = "Cómo lo verifico quien reporta: "

var formatoREP = regexp.MustCompile(`^REP-\d+$`)

// VerificoDe es el «cómo lo verifico» de un detalle, o vacío si no lo trae.
func VerificoDe(detalle string) string {
	i := strings.LastIndex(detalle, marcaVerifico)
	if i < 0 {
		return ""
	}
	return strings.TrimSpace(detalle[i+len(marcaVerifico):])
}

// TituloVerificacion es el título que la cola de QDD reconoce como la verificación de rep.
func TituloVerificacion(rep string, sigueFallando bool, version string) string {
	resultado := "Verificado"
	if sigueFallando {
		resultado = "Sigue fallando"
	}
	return fmt.Sprintf("%s %s en %s", resultado, rep, strings.TrimPrefix(strings.TrimSpace(version), "v"))
}

// conVerificacion revisa lo que MEJ-087 pide a un reporte y deja los datos listos para armarlo.
func conVerificacion(d Datos) (Datos, error) {
	if strings.TrimSpace(d.VerificaDe) != "" {
		return datosDeVerificacion(d)
	}
	verifico := strings.TrimSpace(d.Verifico)
	if verifico == "" && (d.Tipo == "" || d.Tipo == "bug") {
		return d, fmt.Errorf("quien reporta valida: un bug dice cómo lo vas a verificar cuando salga el arreglo (--verifico \"qué hago y qué espero ver\")")
	}
	if verifico == "" {
		return d, nil
	}
	d.Detalle = strings.TrimSpace(strings.TrimSpace(d.Detalle) + "\n\n" + marcaVerifico + verifico)
	return d, nil
}

func datosDeVerificacion(d Datos) (Datos, error) {
	rep := strings.TrimSpace(d.VerificaDe)
	if !formatoREP.MatchString(rep) {
		return d, fmt.Errorf("--verifique nombra el reporte como lo dio el relay (REP-123): %q", rep)
	}
	if strings.TrimSpace(d.Evidencia) == "" {
		return d, fmt.Errorf("se cierra solo con evidencia: --evidencia \"qué corriste y qué viste\"")
	}
	tipo := "mejora"
	if d.SigueFallando {
		tipo = "bug"
	}
	return Datos{Titulo: TituloVerificacion(rep, d.SigueFallando, d.Version), Detalle: d.Evidencia, Tipo: tipo, Version: d.Version, SO: d.SO}, nil
}
