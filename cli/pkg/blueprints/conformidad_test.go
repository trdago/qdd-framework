package blueprints

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/modules"
)

func TestConformidad_CadaEscenarioDeLaVarianteTieneSuPrueba(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "src", "panel", "panel_test.go"), "func TestPanel_SeVenLasMasNuevas_P_01(t *testing.T) {}\n")
	writeFile(t, filepath.Join(root, "src", "panel", "panel.go"), "// P-02: el aviso (un comentario del código no es una prueba)\n")
	writeFile(t, filepath.Join(root, "src", "panel", "viejo_test.go"), "// P-020 es otro escenario\n")
	b, rutas := mustParse(t, recetaConLimites), []string{"src/panel"}

	c := ConformidadDe(root, "panel", b, modules.Implementacion{Blueprint: "panel", Respuestas: map[string]string{"avisos": "no"}}, rutas)
	if strings.Join(c.Escenarios, ",") != "P-01" || len(c.Faltan) != 0 {
		t.Fatalf("con sus respuestas, solo los escenarios de su variante; el id con «_» en el nombre de la prueba vale: %+v", c)
	}
	c = ConformidadDe(root, "panel", b, modules.Implementacion{Blueprint: "panel"}, rutas)
	if strings.Join(c.Escenarios, ",") != "P-01,P-02" || strings.Join(c.Faltan, ",") != "P-02" {
		t.Fatalf("sin respuestas (la implementación de origen), todas las opciones; ni el código ni P-020 cuentan como prueba de P-02: %+v", c)
	}
	writeFile(t, filepath.Join(root, "src", "panel", "tests", "aviso.test.ts"), "test('P-02 el aviso no quita el foco', () => {})\n")
	if c = ConformidadDe(root, "panel", b, modules.Implementacion{Blueprint: "panel"}, rutas); len(c.Faltan) != 0 || !strings.Contains(LineaConformidad(c), "2/2") {
		t.Fatalf("con su prueba, conforme: %+v %q", c, LineaConformidad(c))
	}
}

func TestConformar_LoAplicadoYLoQueUnaFichaImplementa(t *testing.T) {
	root := t.TempDir()
	if _, problems := AddManaged(AddInput{Root: root, Origen: gitOrgRepo(t), Blueprints: []string{"auth"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	for _, id := range []string{"origen", "cliente"} {
		writeFile(t, filepath.Join(root, "internal", id, "auth_test.go"), "func TestLogin_AUTH_01(t *testing.T) {}\n")
		m := modules.Module{ID: id, Nombre: id, Rutas: []string{"internal/" + id},
			ObjetivoOperativo:   modules.ObjetivoOperativo{Problema: "Entrar.", CriterioDeExito: "Entra."},
			ObjetivoTecnologico: modules.ObjetivoTecnologico{Arquitectura: "Un login."},
			Pruebas:             modules.Pruebas{Comandos: []string{"go test ./internal/..."}}}
		if id == "origen" {
			m.Implementa = []modules.Implementacion{{Blueprint: "auth"}}
		}
		if _, problems := modules.Save(root, m); len(problems) > 0 {
			t.Fatal(problems)
		}
	}
	if _, err := saveApplication(root, Aplicacion{Blueprint: "auth", Version: "1.0.0", Modulo: "cliente", Respuestas: map[string]string{"identificador": "correo", "recuperacion": "correo"}}); err != nil {
		t.Fatal(err)
	}
	cs, err := Conformar(root)
	if err != nil || len(cs) != 2 {
		t.Fatalf("una conformidad por módulo que aplica o implementa un blueprint: %+v %v", cs, err)
	}
	porModulo := map[string]Conformidad{}
	for _, c := range cs {
		porModulo[c.Modulo] = c
	}
	if c := porModulo["cliente"]; len(c.Faltan) != 0 || c.Blueprint != "auth@1.0.0" {
		t.Fatalf("lo aplicado se mide con sus respuestas (correo): no le toca RUT-01: %+v", c)
	}
	if c := porModulo["origen"]; !strings.Contains(strings.Join(c.Faltan, ","), "RUT-01") {
		t.Fatalf("la implementación de origen cubre todas las opciones: le falta RUT-01: %+v", c)
	}
}

func TestConformidad_UnEscenarioFueraDelCodigoSeDiceConSuMotivo(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "src", "panel", "panel_test.go"), "func TestPanel_P_01(t *testing.T) {}\n")
	b, rutas := mustParse(t, recetaConLimites), []string{"src/panel"}
	impl := modules.Implementacion{Blueprint: "panel", Fuera: map[string]string{"P-02": "el aviso lo da el sistema operativo"}}
	c := ConformidadDe(root, "panel", b, impl, rutas)
	linea := LineaConformidad(c)
	if len(c.Faltan) != 0 || !strings.HasPrefix(linea, "✅") || !strings.Contains(linea, "1/1") || !strings.Contains(linea, "fuera P-02 (el aviso lo da el sistema operativo)") {
		t.Fatalf("lo que no hace este código no falta, pero siempre se ve con su motivo: %+v %q", c, linea)
	}
	impl.Fuera = map[string]string{"P-02": " ", "X-99": "no existe"}
	c = ConformidadDe(root, "panel", b, impl, rutas)
	if strings.Join(c.Faltan, ",") != "P-02,X-99" {
		t.Fatalf("sin motivo, o de un escenario que no existe, no vale: %+v", c)
	}
}

// fichaQueImplementa guarda la ficha de un módulo que declara implementar el blueprint.
func fichaQueImplementa(t *testing.T, root, id, blueprint string) {
	t.Helper()
	m := modules.Module{ID: id, Nombre: id, Rutas: []string{"internal/" + id},
		ObjetivoOperativo:   modules.ObjetivoOperativo{Problema: "Entrar.", CriterioDeExito: "Entra."},
		ObjetivoTecnologico: modules.ObjetivoTecnologico{Arquitectura: "Un login."},
		Pruebas:             modules.Pruebas{Comandos: []string{"go test ./internal/..."}},
		Implementa:          []modules.Implementacion{{Blueprint: blueprint}}}
	if _, problems := modules.Save(root, m); len(problems) > 0 {
		t.Fatal(problems)
	}
}

func TestCheckConformidad_UnEscenarioSinPruebaNoPasaElGate_BP08(t *testing.T) {
	root := t.TempDir()
	if _, problems := AddManaged(AddInput{Root: root, Origen: gitOrgRepo(t), Blueprints: []string{"auth"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	writeFile(t, filepath.Join(root, "internal", "acceso", "auth_test.go"), "func TestLogin_AUTH_01(t *testing.T) {}\n")
	fichaQueImplementa(t, root, "acceso", "auth")

	bloquean, sinMedir := CheckConformidad(root)
	if len(bloquean) != 1 || bloquean[0].Regla != "BP-08-CONFORMIDAD" || len(sinMedir) != 0 {
		t.Fatalf("al módulo le falta la prueba de RUT-01: un problema que bloquea: %+v %+v", bloquean, sinMedir)
	}
	if p := bloquean[0]; !strings.Contains(p.Descripcion, "acceso") || !strings.Contains(p.Descripcion, "RUT-01") || !strings.Contains(p.Descripcion, "auth@1.0.0") || !strings.HasPrefix(p.Archivo, modules.Dir) {
		t.Fatalf("dice qué módulo, qué blueprint y qué escenario, y apunta a su ficha: %+v", p)
	}
	writeFile(t, filepath.Join(root, "internal", "acceso", "rut_test.go"), "func TestLogin_ConRut_RUT_01(t *testing.T) {}\n")
	if bloquean, sinMedir = CheckConformidad(root); len(bloquean)+len(sinMedir) != 0 {
		t.Fatalf("con todas sus pruebas, pasa: %+v %+v", bloquean, sinMedir)
	}
}

func TestCheckConformidad_SinElBlueprintEnLaMaquinaNoSeMideNiUsaLaRed_BP08(t *testing.T) {
	home := freshHome(t)
	repo := gitOrgRepo(t)
	root := t.TempDir()
	writeFile(t, filepath.Join(root, configFile), "blueprints:\n  origen: "+repo+"\n")
	writeFile(t, filepath.Join(root, "internal", "acceso", "auth_test.go"), "func TestLogin_AUTH_01(t *testing.T) {}\n")
	fichaQueImplementa(t, root, "acceso", "auth")

	bloquean, sinMedir := CheckConformidad(root)
	if len(bloquean) != 0 || len(sinMedir) != 1 || sinMedir[0].Regla != "BP-08-SIN-MEDIR" || !strings.Contains(sinMedir[0].Descripcion, "auth") {
		t.Fatalf("ni importado ni con la copia local del repositorio (el CI): sin medir, sin bloquear: %+v %+v", bloquean, sinMedir)
	}
	if entradas, _ := os.ReadDir(filepath.Join(home, "blueprints", "mirrors")); len(entradas) != 0 {
		t.Fatalf("el gate no trae nada de la red: %v", entradas)
	}
	if _, _, err := publicados(root, []string{"auth"}); err != nil {
		t.Fatal(err)
	}
	if bloquean, sinMedir = CheckConformidad(root); len(bloquean) != 1 || !strings.Contains(bloquean[0].Descripcion, "RUT-01") || len(sinMedir) != 0 {
		t.Fatalf("con la copia local del repositorio, se mide contra lo último publicado ahí: %+v %+v", bloquean, sinMedir)
	}
}

// conImplementados reemplaza, durante la prueba, lo que este binario declara implementar.
func conImplementados(t *testing.T, tabla map[string]string) {
	t.Helper()
	antes := implementadosPorQDD
	implementadosPorQDD = tabla
	t.Cleanup(func() { implementadosPorQDD = antes })
}

// proyectoQueAplicaAuth importa auth@1.0.0 y lo aplica a un módulo que solo lo usa: en sus rutas no hay pruebas.
func proyectoQueAplicaAuth(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if _, problems := AddManaged(AddInput{Root: root, Origen: gitOrgRepo(t), Blueprints: []string{"auth"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	m := modules.Module{ID: "tablero", Nombre: "tablero", Rutas: []string{".qdd/project/estado.yaml"},
		ObjetivoOperativo:   modules.ObjetivoOperativo{Problema: "Entrar.", CriterioDeExito: "Entra."},
		ObjetivoTecnologico: modules.ObjetivoTecnologico{Arquitectura: "Lo hace qdd."},
		Pruebas:             modules.Pruebas{Comandos: []string{"qdd estado check"}}}
	if _, problems := modules.Save(root, m); len(problems) > 0 {
		t.Fatal(problems)
	}
	if _, err := saveApplication(root, Aplicacion{Blueprint: "auth", Version: "1.0.0", Modulo: "tablero", Respuestas: map[string]string{"identificador": "correo", "recuperacion": "correo"}}); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestCheckConformidad_UnBlueprintQueImplementaQDDNoPidePruebasAlProyecto_FND102(t *testing.T) {
	root := proyectoQueAplicaAuth(t)
	conImplementados(t, map[string]string{})
	if bloquean, _ := CheckConformidad(root); len(bloquean) != 1 {
		t.Fatalf("sin que QDD lo implemente, al módulo le faltan sus pruebas: %+v", bloquean)
	}

	conImplementados(t, map[string]string{"auth": "1.2.0"})
	bloquean, sinMedir := CheckConformidad(root)
	if len(bloquean)+len(sinMedir) != 0 {
		t.Fatalf("lo implementa el qdd instalado (1.2.0 cubre 1.0.0): no se le piden pruebas al proyecto: %+v %+v", bloquean, sinMedir)
	}
	cs, err := Conformar(root)
	if err != nil || len(cs) != 1 || !strings.HasPrefix(LineaConformidad(cs[0]), "✅") || !strings.Contains(LineaConformidad(cs[0]), "lo implementa qdd (auth@1.2.0)") {
		t.Fatalf("la conformidad dice que la cumple qdd y en qué versión: %+v %v", cs, err)
	}
}

func TestCheckConformidad_UnQDDQueNoCubreLaVersionAplicadaBloquea_FND102(t *testing.T) {
	root := proyectoQueAplicaAuth(t)
	for implementa, pista := range map[string]string{"0.9.0": "actualiza qdd", "2.0.0": "qdd blueprint add auth"} {
		conImplementados(t, map[string]string{"auth": implementa})
		bloquean, _ := CheckConformidad(root)
		if len(bloquean) != 1 || bloquean[0].Regla != "BP-08-CONFORMIDAD" || !strings.Contains(bloquean[0].Descripcion, "auth@1.0.0") ||
			!strings.Contains(bloquean[0].Descripcion, "auth@"+implementa) || !strings.Contains(bloquean[0].Descripcion, pista) {
			t.Fatalf("qdd implementa auth@%s y el módulo aplica 1.0.0: bloquea y dice qué actualizar (%s): %+v", implementa, pista, bloquean)
		}
	}
}

func TestCheckConformidad_LaImplementacionDeOrigenSiempreSeMideConSusPruebas_FND102(t *testing.T) {
	root := t.TempDir()
	if _, problems := AddManaged(AddInput{Root: root, Origen: gitOrgRepo(t), Blueprints: []string{"auth"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	writeFile(t, filepath.Join(root, "internal", "acceso", "auth_test.go"), "func TestLogin_AUTH_01(t *testing.T) {}\n")
	fichaQueImplementa(t, root, "acceso", "auth")
	conImplementados(t, map[string]string{"auth": "1.0.0"})
	if bloquean, _ := CheckConformidad(root); len(bloquean) != 1 || !strings.Contains(bloquean[0].Descripcion, "RUT-01") {
		t.Fatalf("el código que lo implementa (el repositorio de qdd) prueba cada escenario aunque esté en la tabla: %+v", bloquean)
	}
}

func TestImplementadosPorQDD_SonLosQueImplementanSusFichas_FND102(t *testing.T) {
	repo := filepath.Join("..", "..", "..")
	if _, err := os.Stat(filepath.Join(repo, ".qdd", "project", "modules")); err != nil {
		t.Skip("fuera del repositorio de qdd")
	}
	fichas, _ := modules.Load(repo)
	implementan := map[string]bool{}
	for _, m := range fichas {
		for _, impl := range m.Implementa {
			implementan[impl.Blueprint] = true
		}
	}
	for id := range implementan {
		if _, ok := implementadosPorQDD[id]; !ok {
			t.Errorf("una ficha de qdd implementa %s, pero el binario no lo declara en implementadosPorQDD", id)
		}
	}
	for id, v := range implementadosPorQDD {
		if !implementan[id] {
			t.Errorf("implementadosPorQDD declara %s@%s, pero ninguna ficha de qdd lo implementa", id, v)
		}
		if _, ok := parseVersion(v); !ok {
			t.Errorf("%s: «%s» no es una versión semver", id, v)
		}
	}
}

func TestNombraEscenario_ElIdPegadoATestEnGoCuenta_FND103(t *testing.T) {
	for _, texto := range []string{"func TestCVA_C01_LasFases(t *testing.T) {}", "func TestCVAC01(t *testing.T) {}", "def test_CVA_C01(): pass", "test('CVA-C01 las fases', () => {})"} {
		if texto == "func TestCVAC01(t *testing.T) {}" {
			if nombraEscenario(texto, "CVA-C01") {
				t.Fatalf("sin separador entre las partes del id no es el id: %q", texto)
			}
			continue
		}
		if !nombraEscenario(texto, "CVA-C01") {
			t.Fatalf("el id del escenario va pegado al prefijo de la prueba (TestCVA_C01, la convención de Go): %q", texto)
		}
	}
	for _, texto := range []string{"func TestXCVA_C01(t *testing.T) {}", "func TestCVA_C010(t *testing.T) {}", "func TestOAUTH_01(t *testing.T) {}"} {
		if nombraEscenario(texto, "CVA-C01") || nombraEscenario(texto, "AUTH-01") {
			t.Fatalf("otra palabra en mayúsculas pegada al id, o un id más largo, no es este escenario: %q", texto)
		}
	}
}

// FND-121: a un módulo que aplica un blueprint (registro de aplicación) no le servía «implementa: fuera»: sumarlo a
// su ficha creaba otra medición, la de origen, con todas las variantes. Su ficha lo declara en la raíz, «fuera».
func TestCheckConformidad_UnModuloQueAplicaDeclaraFueraEnSuFicha_FND121(t *testing.T) {
	root := t.TempDir()
	if _, problems := AddManaged(AddInput{Root: root, Origen: gitOrgRepo(t), Blueprints: []string{"auth"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	writeFile(t, filepath.Join(root, "internal", "cliente", "auth_test.go"), "func TestLogin_AUTH_01(t *testing.T) {}\n")
	m := modules.Module{ID: "cliente", Nombre: "cliente", Rutas: []string{"internal/cliente"}, Blueprint: "auth@1.0.0",
		ObjetivoOperativo:   modules.ObjetivoOperativo{Problema: "Entrar.", CriterioDeExito: "Entra."},
		ObjetivoTecnologico: modules.ObjetivoTecnologico{Arquitectura: "Un login."},
		Pruebas:             modules.Pruebas{Comandos: []string{"go test ./internal/..."}}}
	if _, problems := modules.Save(root, m); len(problems) > 0 {
		t.Fatal(problems)
	}
	if _, err := saveApplication(root, Aplicacion{Blueprint: "auth", Version: "1.0.0", Modulo: "cliente", Respuestas: map[string]string{"identificador": "rut", "recuperacion": "correo"}}); err != nil {
		t.Fatal(err)
	}
	bloquean, _ := CheckConformidad(root)
	if len(bloquean) != 1 || !strings.Contains(bloquean[0].Descripcion, "RUT-01") || !strings.Contains(bloquean[0].Descripcion, "fuera: {ID: motivo}") {
		t.Fatalf("con rut le falta RUT-01, y el gate dice cómo declararlo fuera en la ficha de un módulo que aplica: %+v", bloquean)
	}

	m.Fuera = map[string]string{"RUT-01": "el RUT lo valida el registro civil"}
	if _, problems := modules.Save(root, m); len(problems) > 0 {
		t.Fatal(problems)
	}
	if bloquean, _ = CheckConformidad(root); len(bloquean) != 0 {
		t.Fatalf("declarado fuera con su motivo, pasa: %+v", bloquean)
	}
	cs, _ := Conformar(root)
	if len(cs) != 1 || !strings.Contains(LineaConformidad(cs[0]), "fuera RUT-01 (el RUT lo valida el registro civil)") {
		t.Fatalf("una sola medición, la de lo aplicado, que muestra lo de fuera: %+v", cs)
	}

	m.Fuera = map[string]string{"RUT-1": "con una errata"}
	if _, problems := modules.Save(root, m); len(problems) > 0 {
		t.Fatal(problems)
	}
	if bloquean, _ = CheckConformidad(root); len(bloquean) != 1 || !strings.Contains(bloquean[0].Descripcion, "RUT-1") {
		t.Fatalf("un id que el blueprint no tiene falta, igual que en implementa: %+v", bloquean)
	}
}

// MEJ-022 (issue #143): un módulo que aplica un blueprint y lo declaró fuera en «implementa» seguía bloqueado, y el gate
// solo decía «en la raíz de la ficha», sin nombrar la entrada equivocada. Ahora la nombra.
func TestCheckConformidad_FueraEnImplementaDeUnModuloQueAplicaSeNombra_MEJ022(t *testing.T) {
	root := t.TempDir()
	if _, problems := AddManaged(AddInput{Root: root, Origen: gitOrgRepo(t), Blueprints: []string{"auth"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	writeFile(t, filepath.Join(root, "internal", "cliente", "auth_test.go"), "func TestLogin_AUTH_01(t *testing.T) {}\n")
	m := modules.Module{ID: "cliente", Nombre: "cliente", Rutas: []string{"internal/cliente"}, Blueprint: "auth@1.0.0",
		ObjetivoOperativo:   modules.ObjetivoOperativo{Problema: "Entrar.", CriterioDeExito: "Entra."},
		ObjetivoTecnologico: modules.ObjetivoTecnologico{Arquitectura: "Un login."},
		Pruebas:             modules.Pruebas{Comandos: []string{"go test ./internal/..."}},
		Implementa:          []modules.Implementacion{{Blueprint: "auth", Fuera: map[string]string{"RUT-01": "el RUT lo valida el registro civil"}}}}
	if _, problems := modules.Save(root, m); len(problems) > 0 {
		t.Fatal(problems)
	}
	if _, err := saveApplication(root, Aplicacion{Blueprint: "auth", Version: "1.0.0", Modulo: "cliente", Respuestas: map[string]string{"identificador": "rut", "recuperacion": "correo"}}); err != nil {
		t.Fatal(err)
	}
	bloquean, _ := CheckConformidad(root)
	aplicado := ""
	for _, b := range bloquean {
		if strings.HasPrefix(filepath.ToSlash(b.Archivo), ApplicationsDir) {
			aplicado = b.Descripcion
		}
	}
	if !strings.Contains(aplicado, "RUT-01") || !strings.Contains(aplicado, "implementa") || !strings.Contains(aplicado, "raíz") {
		t.Fatalf("el gate nombra la entrada de implementa que no vale para lo aplicado: %+v", bloquean)
	}
}
