package baul

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'

func diag(t *testing.T, ds []Diagnostico, nombre string) Diagnostico {
	t.Helper()
	for _, d := range ds {
		if d.Nombre == nombre {
			return d
		}
	}
	t.Fatalf("falta %q en %+v", nombre, ds)
	return Diagnostico{}
}

func TestDiagnosticar_SinBaulNiEnvNoDiceNada(t *testing.T) {
	root := entorno(t)
	if ds := Diagnosticar(root, ahora); len(ds) != 0 {
		t.Fatalf("sin baúl ni .env, doctor no muestra el grupo: %+v", ds)
	}
}

func TestDiagnosticar_VencidoRechazadoYSinUso(t *testing.T) {
	root := entorno(t)
	guardarCloudflare(t)
	vence := ahora.Add(3 * 24 * time.Hour)
	viejo := ahora.Add(-100 * 24 * time.Hour)
	if err := cambiarEntrada("cf", func(e *Entrada) { e.Vence, e.Guardado = &vence, viejo }); err != nil {
		t.Fatal(err)
	}
	ds := Diagnosticar(root, ahora)
	if d := diag(t, ds, ItemVigentes); d.OK || !strings.Contains(d.Detalle, "cf vence") {
		t.Fatalf("por vencer en 14 días: %+v", d)
	}
	if d := diag(t, ds, ItemSinUso); !d.OK || !strings.Contains(d.Detalle, "cf") {
		t.Fatalf("sin uso en 90 días es un aviso: %+v", d)
	}
	if d := diag(t, ds, ItemRegistro); !d.OK {
		t.Fatalf("registro intacto: %+v", d)
	}
}

func TestDiagnosticar_RegistroRotoYFixLoAparta(t *testing.T) {
	root := entorno(t)
	guardarCloudflare(t)
	Anotar(Linea{Accion: "usar", Token: "cf"}, ahora)
	Anotar(Linea{Accion: "usar", Token: "cf"}, ahora)
	ruta, _ := rutaEn("registro.jsonl")
	b, _ := os.ReadFile(ruta)
	lineas := strings.Split(strings.TrimSpace(string(b)), "\n")
	os.WriteFile(ruta, []byte(strings.Join(lineas[1:], "\n")+"\n"), 0o600)
	if d := diag(t, Diagnosticar(root, ahora), ItemRegistro); d.OK {
		t.Fatalf("roto: %+v", d)
	}
	apartado, err := ApartarRegistro(ahora)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(apartado); err != nil || filepath.Dir(apartado) != filepath.Dir(ruta) {
		t.Fatalf("el roto queda apartado como evidencia: %s %v", apartado, err)
	}
	if d := diag(t, Diagnosticar(root, ahora), ItemRegistro); !d.OK {
		t.Fatalf("la cadena nueva empieza anotando que se apartó: %+v", d)
	}
	l, _ := Registro()
	if len(l) != 1 || l[0].Accion != "apartar" {
		t.Fatalf("%+v", l)
	}
}

func TestDiagnosticar_EnvConValoresSoloNombres(t *testing.T) {
	root := entorno(t)
	os.WriteFile(filepath.Join(root, ".env"), []byte("RESEND_API_KEY="+secreto+"\n"), 0o600)
	d := diag(t, Diagnosticar(root, ahora), ItemEnv)
	if !d.OK || !strings.Contains(d.Detalle, "RESEND_API_KEY") || strings.Contains(d.Detalle, secreto) {
		t.Fatalf("un aviso con el nombre de la clave, nunca el valor: %+v", d)
	}
}
