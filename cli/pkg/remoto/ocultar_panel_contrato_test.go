package remoto

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'
// Inv. 16 campo por campo: ocultarDelPanel tapa cada texto que se publica, uno a uno. Un campo nuevo del panel salía
// sin ocultar y ninguna prueba fallaba. Esta prueba llena por reflexión todo campo de texto de consola.DatosConsola
// (con sus listas, mapas y punteros) con un token y exige que, después del ocultador, el token solo quede en los campos
// que qdd llena con ids, claves o valores cerrados (estructurales). Un campo nuevo falla aquí hasta que se oculta o se
// declara estructural a conciencia.

const tokenDePrueba = "ghp_abcdefghijklmnopqrstuvwxyz0123456789"

// estructurales son los campos que qdd llena con ids, rutas, claves, horas o valores cerrados; nunca con texto libre.
var estructurales = map[string]bool{
	".Version": true, ".Carpeta": true,
	".Estado.Actualizado": true, ".Estado.Etapas[]": true, ".Estado.Abreviaturas[]": true, ".Estado.Actual": true, ".Estado.Hace": true,
	".Estado.Esperas[].Etapa": true, ".Estado.Fallas[].Etapa": true,
	".Estado.Objetivos[].ID": true, ".Estado.Objetivos[].Simbolo": true, ".Estado.Objetivos[].Ciclo": true,
	".Estado.Ciclos[].Nombre": true, ".Estado.Ciclos[].Etapas[]": true, ".Estado.Ciclos[].Abreviaturas[]": true,
	".Efectos.Estilo.Acento": true, ".Efectos.Estilo.Origen": true, ".Ejecutar.Sugeridos[]": true,
	".Goal.Estado": true, ".Goal.Elegidas[].Tipo": true,
	".EsperasPersona[].Pregunta": true, ".EsperasPersona[].Atender": true, ".EsperasPersona[].Donde": true,
	".Logrado[].Tipo": true, ".Logrado[].ID": true, ".Logrado[].En": true, ".Logrado[].Origen": true, ".Logrado[].Estado": true,
	".Otros[].Proyecto": true, ".Otros[].Carpeta": true,
	".Otros[].Esperas[].Pregunta": true, ".Otros[].Esperas[].Atender": true, ".Otros[].Esperas[].Donde": true,
}

// estructuralesDeAccion son los de una acción, en el panel y en los demás proyectos.
var estructuralesDeAccion = []string{"ID", "Tipo", "Carpeta", "Tecla", "Estado", "Sesion", "VersionNueva", "DeTarea", "Origen", "ResponderEn",
	"RespondidaEn", "Preguntas[].ID", "Preguntas[].Opciones[].Valor", "Permiso.Herramienta", "Dispositivo", "Solicitud"}

// estructuralesDeFila son los de una fila del tablero: las visibles y las ocultas que esperan a la persona (FND-296).
var estructuralesDeFila = []string{"ID", "Estado", "Simbolos[]", "Objetivo", "Prefijo", "Actividad.Pulso", "Actividad.Fuente"}

func init() {
	for _, c := range estructuralesDeFila {
		estructurales[".Estado.Frentes[]."+c] = true
		estructurales[".Estado.OcultasTuyas[]."+c] = true
	}
	for _, c := range estructuralesDeAccion {
		estructurales[".Acciones[]."+c] = true
		estructurales[".Otros[].Acciones[]."+c] = true
	}
}

// llenarConToken pone en cada campo de texto alcanzable «token·ruta» y devuelve las rutas que llenó.
func llenarConToken(v reflect.Value, ruta string, rutas *[]string, prof int) {
	if prof > 10 {
		return
	}
	switch v.Kind() {
	case reflect.String:
		v.SetString(tokenDePrueba + "·" + ruta)
		*rutas = append(*rutas, ruta)
	case reflect.Ptr:
		v.Set(reflect.New(v.Type().Elem()))
		llenarConToken(v.Elem(), ruta, rutas, prof+1)
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() {
				llenarConToken(v.Field(i), ruta+"."+v.Type().Field(i).Name, rutas, prof+1)
			}
		}
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			return
		}
		lista := reflect.MakeSlice(v.Type(), 1, 1)
		llenarConToken(lista.Index(0), ruta+"[]", rutas, prof+1)
		v.Set(lista)
	case reflect.Map:
		if v.Type().Key().Kind() != reflect.String {
			return
		}
		mapa := reflect.MakeMap(v.Type())
		valor := reflect.New(v.Type().Elem()).Elem()
		llenarConToken(valor, ruta+"{}", rutas, prof+1)
		mapa.SetMapIndex(reflect.ValueOf("k"), valor)
		v.Set(mapa)
	}
}

// contrato: remoto#1958cd48
func TestPanel_TodoCampoDeTextoSaleSinSecretos(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	var panel consola.DatosConsola
	var rutas []string
	llenarConToken(reflect.ValueOf(&panel).Elem(), "", &rutas, 0)
	if len(rutas) < 50 {
		t.Fatalf("la reflexión no llenó el panel: %d campos", len(rutas))
	}
	ocultarDelPanel(t.TempDir(), &panel)
	b, err := json.Marshal(panel)
	if err != nil {
		t.Fatal(err)
	}
	var salen []string
	for _, r := range rutas {
		if !estructurales[r] && strings.Contains(string(b), tokenDePrueba+"·"+r+`"`) {
			salen = append(salen, r)
		}
	}
	sort.Strings(salen)
	if len(salen) > 0 {
		t.Fatalf("inv. 16: estos campos salen al teléfono sin pasar por el ocultador (ocúltalos en ocultarDelPanel o, si qdd los llena solo con ids o valores cerrados, decláralos estructurales): %v", salen)
	}
}

// FND-250: un estado.yaml que no se lee dejaba su error en el motivo del panel sin pasar por el ocultador, y el error
// de YAML cita el valor (entero si tiene hasta 10 caracteres): la clave de un .env del proyecto escrita en el tablero
// salía tal cual al teléfono.
// contrato: remoto#1958cd48
func TestArmar_ElMotivoDeUnTableroIlegibleSaleSinSecretos_FND250(t *testing.T) {
	homeTemporal(t)
	root := proyecto(t, "app", false)
	const clave = "Xk9vQ2mZpL"
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("DB_CLAVE="+clave+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".qdd", "project", "estado.yaml"), []byte("formato: 1\netapas: "+clave+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	reg := registro.Registro{Proyectos: []registro.Proyecto{{Ruta: root, Clave: "app-1"}}}
	p := Armar("v", reg, time.Now(), 0).Proyectos[0]
	if p.Panel == nil || !strings.Contains(p.Panel.Motivo, "ilegible") {
		t.Fatalf("la prueba necesita un tablero que no se lee: %+v", p.Panel)
	}
	b, _ := json.Marshal(p)
	if strings.Contains(string(b), clave) {
		t.Fatal("inv. 16: el motivo lleva el valor de un .env del proyecto")
	}
}
