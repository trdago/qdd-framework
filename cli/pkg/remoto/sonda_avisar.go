package remoto

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Rule: Prohibido usar 'else'
// La sonda prueba también el contrato de la Edge Function avisar (sugerencia de qdd-movile, 2026-11-15), que es por
// donde salen los push: un POST con un tópico y una firma que pasan su validación de formato pero no son de nadie, y
// tipo pregunta. Nunca lleva las credenciales de la máquina ni datos de un proyecto. Un relay al día responde 401
// «credenciales de la máquina inválidas»; 404 es que avisar no está desplegada, y 400 «tipo inválido», que es de antes
// de los push de preguntas (FND-208). Una llamada por sonda: una por hora.

const (
	TramoAvisar = "avisar" // la Edge Function de los push no está o no entiende lo que manda este PC

	MotivoAvisarAusente    = "avisar_ausente"
	MotivoAvisarAtrasado   = "avisar_atrasado"
	MotivoAvisarInesperado = "avisar_inesperado"

	// topicoSonda y firmaSonda tienen el formato que valida avisar y no son de ninguna máquina.
	topicoSonda = "qdd:pc:00000000-0000-0000-0000-000000000000:sonda_sin_credenciales"
	// respuestaSinCredenciales es lo que responde un avisar al día a unas credenciales que no son de nadie.
	respuestaSinCredenciales = "credenciales de la máquina inválidas"
)

var (
	firmaSonda = strings.Repeat("0", 64)

	textosAvisar = map[string]string{
		MotivoAvisarAusente:    "el relay no tiene la función de los push (avisar): los avisos no llegan al teléfono",
		MotivoAvisarAtrasado:   "la función de los push del relay es de una versión anterior: las preguntas y permisos no suenan",
		MotivoAvisarInesperado: "la función de los push del relay respondió algo que este PC no espera",
	}
)

// probarAvisar devuelve "" si avisar responde como un relay al día, o el código de lo que falla y el detalle crudo.
func probarAvisar(ctx context.Context, c ConfigRelay) (string, error) {
	cuerpo, _ := json.Marshal(map[string]string{"topico": topicoSonda, "firma": firmaSonda, "tipo": AvisoPregunta, "proyecto": "sonda", "clave": "sonda"})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.URL, "/")+"/functions/v1/avisar", bytes.NewReader(cuerpo))
	if err != nil {
		return MotivoAvisarInesperado, err
	}
	req.Header.Set("apikey", c.Clave)
	req.Header.Set("Authorization", "Bearer "+c.Clave)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return MotivoAvisarInesperado, err
	}
	defer resp.Body.Close()
	var e struct {
		Error string `json:"error"`
	}
	_ = json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&e)
	if resp.StatusCode == http.StatusUnauthorized && e.Error == respuestaSinCredenciales {
		return "", nil
	}
	detalle := fmt.Errorf("avisar respondió %d: %s", resp.StatusCode, e.Error)
	if resp.StatusCode == http.StatusNotFound {
		return MotivoAvisarAusente, detalle
	}
	if resp.StatusCode == http.StatusBadRequest && strings.HasPrefix(e.Error, "tipo inválido") {
		return MotivoAvisarAtrasado, detalle
	}
	return MotivoAvisarInesperado, detalle
}

// conAvisar suma la prueba de avisar a una sonda que salió bien: lo que ya midió (eco, teléfonos, mensajes) se conserva.
func conAvisar(ctx context.Context, c ConfigRelay, r ResultadoSonda) ResultadoSonda {
	if !r.OK {
		return r
	}
	codigo, err := probarAvisar(ctx, c)
	if codigo == "" {
		return r
	}
	r.OK, r.Tramo, r.Codigo, r.Motivo = false, TramoAvisar, codigo, textosAvisar[codigo]
	r.detalle = err.Error()
	return r
}
