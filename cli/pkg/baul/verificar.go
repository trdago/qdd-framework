package baul

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// Rule: Prohibido usar 'else'
// Una llamada barata de solo lectura dice de quién es el token, qué alcanza y cuándo vence.

// ErrRechazado: la API no acepta el token.
var ErrRechazado = errors.New("la API rechazó el token")

// apis: las pruebas las cambian por un servidor de prueba.
var apis = map[string]string{
	"cloudflare": "https://api.cloudflare.com/client/v4/user/tokens/verify",
	"github":     "https://api.github.com/user",
	"supabase":   "https://api.supabase.com/v1/projects",
	"resend":     "https://api.resend.com/domains",
}

var binAWS = "aws"

// Verificacion es lo que se supo del token.
type Verificacion struct {
	Duenio  string
	Alcance string
	Vence   *time.Time
}

// VerificarValor llama a la API del tipo. «otro» no se verifica.
func VerificarValor(tipo, valor string) (Verificacion, error) {
	if tipo == "aws-sso" {
		return verificarAWS(valor)
	}
	url, ok := apis[tipo]
	if !ok {
		return Verificacion{}, fmt.Errorf("el tipo %s no se verifica", tipo)
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Verificacion{}, err
	}
	req.Header.Set("Authorization", "Bearer "+valor)
	req.Header.Set("Accept", "application/json")
	res, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return Verificacion{}, fmt.Errorf("no se pudo verificar (sin red?): %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden {
		return Verificacion{}, fmt.Errorf("%w (%d)", ErrRechazado, res.StatusCode)
	}
	if res.StatusCode >= 300 {
		return Verificacion{}, fmt.Errorf("la API respondió %d", res.StatusCode)
	}
	var cuerpo json.RawMessage
	if err := json.NewDecoder(res.Body).Decode(&cuerpo); err != nil {
		return Verificacion{}, fmt.Errorf("respuesta ilegible: %v", err)
	}
	return leerVerificacion(tipo, cuerpo, res.Header), nil
}

func leerVerificacion(tipo string, cuerpo json.RawMessage, h http.Header) Verificacion {
	switch tipo {
	case "cloudflare":
		var r struct {
			Result struct {
				ID      string `json:"id"`
				Status  string `json:"status"`
				Expires string `json:"expires_on"`
			} `json:"result"`
		}
		_ = json.Unmarshal(cuerpo, &r)
		v := Verificacion{Duenio: r.Result.ID, Alcance: "token " + r.Result.Status + " (los permisos se ven en el panel de Cloudflare)"}
		if t, err := time.Parse(time.RFC3339, r.Result.Expires); err == nil {
			v.Vence = &t
		}
		return v
	case "github":
		var r struct {
			Login string `json:"login"`
		}
		_ = json.Unmarshal(cuerpo, &r)
		alcance := h.Get("X-OAuth-Scopes")
		if alcance == "" {
			alcance = "fine-grained (los permisos se ven en GitHub)"
		}
		v := Verificacion{Duenio: r.Login, Alcance: alcance}
		if t, err := time.Parse("2006-01-02 15:04:05 MST", h.Get("GitHub-Authentication-Token-Expiration")); err == nil {
			v.Vence = &t
		}
		return v
	}
	var lista []json.RawMessage
	if err := json.Unmarshal(cuerpo, &lista); err == nil {
		return Verificacion{Alcance: fmt.Sprintf("%d proyectos", len(lista))}
	}
	var r struct {
		Data []json.RawMessage `json:"data"`
	}
	_ = json.Unmarshal(cuerpo, &r)
	return Verificacion{Alcance: fmt.Sprintf("%d dominios", len(r.Data))}
}

func verificarAWS(perfil string) (Verificacion, error) {
	var salida, errSalida bytes.Buffer
	c := exec.Command(binAWS, "sts", "get-caller-identity", "--profile", perfil, "--output", "json")
	c.Stdout, c.Stderr = &salida, &errSalida
	if err := c.Run(); err != nil {
		return Verificacion{}, fmt.Errorf("aws no respondió (¿aws sso login --profile %s?): %s", perfil, strings.TrimSpace(errSalida.String()))
	}
	var r struct {
		Arn string `json:"Arn"`
	}
	_ = json.Unmarshal(salida.Bytes(), &r)
	return Verificacion{Duenio: r.Arn, Alcance: "perfil " + perfil + " (credenciales de IAM Identity Center, vencen solas)"}, nil
}

// Verificar prueba el token guardado y anota en el índice lo que se supo.
func Verificar(nombre string, ahora time.Time) (Entrada, error) {
	e, err := Buscar(nombre)
	if err != nil {
		return e, err
	}
	if e.Tipo == "otro" {
		return e, fmt.Errorf("un token «otro» no se verifica: QDD no sabe a qué API va")
	}
	valor, err := leerValor(e)
	if err != nil {
		return e, err
	}
	v, errV := VerificarValor(e.Tipo, valor)
	if errV != nil && !errors.Is(errV, ErrRechazado) {
		return e, errV
	}
	err = cambiarEntrada(nombre, func(x *Entrada) {
		t := ahora.UTC()
		x.Verificado, x.Rechazado = &t, errV != nil
		if errV != nil {
			return
		}
		x.Duenio, x.Alcance, x.Vence = v.Duenio, v.Alcance, v.Vence
	})
	if err != nil {
		return e, err
	}
	_ = Anotar(Linea{Accion: "verificar", Token: nombre}, ahora)
	e, _ = Buscar(nombre)
	return e, errV
}
