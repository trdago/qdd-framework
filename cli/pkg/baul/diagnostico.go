package baul

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Rule: Prohibido usar 'else'
// doctor (RFC 0023): tokens por vencer o rechazados, tokens sin uso, un registro roto y .env con valores reales.

const (
	ItemRegistro = "Registro del baúl intacto"
	ItemVigentes = "Tokens del baúl vigentes"
	ItemSinUso   = "Tokens del baúl en uso"
	ItemEnv      = ".env con secretos fuera del baúl"

	porVencer = 14 * 24 * time.Hour
	sinUso    = 90 * 24 * time.Hour
)

// Diagnostico es un ítem de doctor. OK con Detalle es un aviso.
type Diagnostico struct {
	Nombre  string
	OK      bool
	Detalle string
}

// Diagnosticar revisa el baúl y los .env del proyecto. Sin baúl ni .env con valores no dice nada.
func Diagnosticar(root string, ahora time.Time) []Diagnostico {
	var ds []Diagnostico
	indice, err := Indice()
	if err != nil {
		ds = append(ds, Diagnostico{Nombre: ItemVigentes, Detalle: "índice ilegible: " + err.Error()})
	}
	if len(indice) > 0 {
		ds = append(ds, diagnosticarRegistro(), diagnosticarVigentes(indice, ahora), diagnosticarUso(indice, ahora))
	}
	for _, e := range EnvsConValores(root) {
		ds = append(ds, Diagnostico{Nombre: ItemEnv, OK: true, Detalle: fmt.Sprintf("%s: %s · pásalos al baúl (qdd baul guardar) y bórralos del archivo", e.Archivo, strings.Join(e.Claves, ", "))})
	}
	return ds
}

func diagnosticarRegistro() Diagnostico {
	if err := VerificarRegistro(); err != nil {
		return Diagnostico{Nombre: ItemRegistro, Detalle: err.Error() + " · qdd doctor --fix lo aparta como evidencia y empieza otro"}
	}
	return Diagnostico{Nombre: ItemRegistro, OK: true}
}

func diagnosticarVigentes(indice []Entrada, ahora time.Time) Diagnostico {
	var malos []string
	for _, e := range indice {
		if e.Rechazado {
			malos = append(malos, e.Nombre+" rechazado")
			continue
		}
		if e.Vence != nil && e.Vence.Sub(ahora) < porVencer {
			malos = append(malos, e.Nombre+" vence el "+e.Vence.Local().Format("2006-01-02"))
		}
	}
	if len(malos) == 0 {
		return Diagnostico{Nombre: ItemVigentes, OK: true, Detalle: fmt.Sprintf("%d tokens", len(indice))}
	}
	return Diagnostico{Nombre: ItemVigentes, Detalle: strings.Join(malos, ", ") + " · crea uno nuevo y guárdalo en tu terminal (qdd baul guardar)"}
}

func diagnosticarUso(indice []Entrada, ahora time.Time) Diagnostico {
	var quietos []string
	for _, e := range indice {
		ultimo := e.Guardado
		if e.Usado != nil {
			ultimo = *e.Usado
		}
		if ahora.Sub(ultimo) > sinUso {
			quietos = append(quietos, e.Nombre)
		}
	}
	if len(quietos) == 0 {
		return Diagnostico{Nombre: ItemSinUso, OK: true}
	}
	return Diagnostico{Nombre: ItemSinUso, OK: true, Detalle: "sin uso en 90 días: " + strings.Join(quietos, ", ") + " · si ya no sirven, revócalos en su proveedor"}
}

// ApartarRegistro renombra un registro roto (queda como evidencia) y empieza otro que lo anota.
func ApartarRegistro(ahora time.Time) (string, error) {
	ruta, err := rutaEn("registro.jsonl")
	if err != nil {
		return "", err
	}
	apartado := strings.TrimSuffix(ruta, ".jsonl") + "-roto-" + ahora.UTC().Format("20060102T150405") + ".jsonl"
	if err := conBloqueo(func() error { return os.Rename(ruta, apartado) }); err != nil {
		return "", err
	}
	return apartado, Anotar(Linea{Accion: "apartar", Token: "-", Fin: "registro roto apartado en " + apartado}, ahora)
}
