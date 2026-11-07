package audit

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// RFC 0020, «lo que se tomó de Docker»: las rutas se comprueban en una copia que solo tiene lo declarado, la huella incluye
// la versión de la herramienta y lo barato corre primero.

func (r repoDeSuites) config(t *testing.T, suites string) {
	t.Helper()
	writeFile(t, filepath.Join(r.raiz, ".qdd", "config.yaml"), "test_commands:\n"+suites)
}

func estadoDe(t *testing.T, r repoDeSuites) ResultadoRutas {
	t.Helper()
	resultados := VerificarRutas(r.raiz)
	if len(resultados) != 1 {
		t.Fatalf("una suite con rutas: %+v", resultados)
	}
	return resultados[0]
}

func TestVerificarRutas_ConTodoDeclaradoPasa(t *testing.T) {
	r := repoConSuite(t, "true")
	r.config(t, "  - comando: 'cd ui && cat boton.css'\n    rutas: [ui]\n")
	if got := estadoDe(t, r); got.Estado != RutasCompletas {
		t.Fatalf("%+v", got)
	}
}

func TestVerificarRutas_UnArchivoSinDeclararSeDetecta(t *testing.T) {
	r := repoConSuite(t, "true")
	r.config(t, "  - comando: 'cat reglas/negocio.go'\n    rutas: [ui]\n")
	if got := estadoDe(t, r); got.Estado != RutasIncompletas {
		t.Fatalf("en el árbol pasa, en la copia no: %+v", got)
	}
}

func TestVerificarRutas_UnaSuiteQueFallaEnElArbolNoDiceNadaDeSusRutas(t *testing.T) {
	r := repoConSuite(t, "true")
	r.config(t, "  - comando: 'exit 3'\n    rutas: [ui]\n")
	if got := estadoDe(t, r); got.Estado != SuiteFalla {
		t.Fatalf("%+v", got)
	}
}

func TestVerificarRutas_UnaDependenciaInstaladaNoSePuedeComprobar(t *testing.T) {
	r := repoConSuite(t, "true")
	r.config(t, `  - comando: '[ -f reglas/negocio.go ] || { echo "Error: Cannot find module \"vitest\""; exit 1; }'`+"\n    rutas: [ui]\n")
	if got := estadoDe(t, r); got.Estado != RutasSinComprobar {
		t.Fatalf("falta instalar, no falta declarar: %+v", got)
	}
}

func TestVerificarRutas_UnModuloRelativoEsUnaRutaQueFalta(t *testing.T) {
	r := repoConSuite(t, "true")
	r.config(t, `  - comando: '[ -f reglas/negocio.go ] || { echo "Error: Cannot find module \"../reglas/negocio\""; exit 1; }'`+"\n    rutas: [ui]\n")
	if got := estadoDe(t, r); got.Estado != RutasIncompletas {
		t.Fatalf("%+v", got)
	}
}

func TestVerificarRutas_SinSuHerramientaNoSePuedeComprobar(t *testing.T) {
	r := repoConSuite(t, "true")
	r.config(t, "  - comando: 'true'\n    rutas: [ui]\n    requiere: [qdd-programa-que-no-existe]\n")
	if got := estadoDe(t, r); got.Estado != RutasSinComprobar {
		t.Fatalf("%+v", got)
	}
}

func TestVerificarRutas_UnaRutaSinArchivosEstaMalDeclarada(t *testing.T) {
	r := repoConSuite(t, "true")
	r.config(t, "  - comando: 'true'\n    rutas: [ui, no-existe]\n")
	if got := estadoDe(t, r); got.Estado != RutasIncompletas || !strings.Contains(got.Detalle, "no-existe") {
		t.Fatalf("%+v", got)
	}
}

func TestVerificarRutas_NoMiraLasSuitesSinRutas(t *testing.T) {
	r := repoConSuite(t, "true")
	r.config(t, `  - 'echo x >> "`+r.cuenta+`"'`+"\n")
	if got := VerificarRutas(r.raiz); len(got) != 0 || r.corridas(t) != 0 {
		t.Fatalf("%+v", got)
	}
}

func TestVerificarRutas_LosIgnoradosNoSeCopian(t *testing.T) {
	r := repoConSuite(t, "true")
	writeFile(t, filepath.Join(r.raiz, ".gitignore"), "ui/generado.txt\n")
	writeFile(t, filepath.Join(r.raiz, "ui", "generado.txt"), "x\n")
	r.config(t, "  - comando: 'cat ui/generado.txt'\n    rutas: [ui]\n")
	if got := estadoDe(t, r); got.Estado != RutasIncompletas {
		t.Fatalf("%+v", got)
	}
}

func TestVerificarRutas_NoTocaElArbolYBorraLaCopia(t *testing.T) {
	r := repoConSuite(t, "true")
	donde := filepath.Join(t.TempDir(), "donde")
	r.config(t, `  - comando: 'pwd > "`+donde+`"; echo x > ui/escrito.txt'`+"\n    rutas: [ui]\n")
	if got := estadoDe(t, r); got.Estado != RutasCompletas {
		t.Fatalf("%+v", got)
	}
	if _, err := os.Stat(filepath.Join(r.raiz, "ui", "escrito.txt")); err == nil {
		t.Fatal("corrió en el árbol")
	}
	copia, err := os.ReadFile(donde)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(strings.TrimSpace(string(copia))); err == nil {
		t.Fatalf("la copia sigue ahí: %s", copia)
	}
}

func TestVerificarRutas_ConservaPermisosYEnlaces(t *testing.T) {
	r := repoConSuite(t, "true")
	guion := filepath.Join(r.raiz, "ui", "correr.sh")
	writeFile(t, guion, "#!/bin/sh\nexit 0\n")
	if err := os.Chmod(guion, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("boton.css", filepath.Join(r.raiz, "ui", "enlace.css")); err != nil {
		t.Skip("este sistema no crea enlaces")
	}
	r.config(t, "  - comando: 'ui/correr.sh && [ -L ui/enlace.css ] && cat ui/enlace.css'\n    rutas: [ui]\n")
	if got := estadoDe(t, r); got.Estado != RutasCompletas {
		t.Fatalf("%+v", got)
	}
}

func TestInformeDeRutas_DiceCadaCasoYSoloFallaLoQueHayQueArreglar(t *testing.T) {
	var salida bytes.Buffer
	falla := InformeDeRutas(&salida, []ResultadoRutas{
		{Suite: "a", Estado: RutasCompletas},
		{Suite: "b", Estado: RutasSinComprobar, Detalle: "falta instalar"},
	})
	if falla || !strings.Contains(salida.String(), "a") || !strings.Contains(salida.String(), "falta instalar") {
		t.Fatalf("%v %s", falla, salida.String())
	}
	if !InformeDeRutas(&salida, []ResultadoRutas{{Suite: "c", Estado: RutasIncompletas}}) {
		t.Fatal("una ruta que falta hace fallar el comando")
	}
	if !InformeDeRutas(&salida, []ResultadoRutas{{Suite: "d", Estado: SuiteFalla}}) {
		t.Fatal("una suite que falla hace fallar el comando")
	}
}

// Como fijar el digest en FROM: al actualizar la herramienta no hay que esperar a que venza lo recordado.
func TestSuite_CambiarLaVersionDeLaHerramientaLaRepite(t *testing.T) {
	r := repoConSuite(t, "true")
	bin := t.TempDir()
	version := filepath.Join(bin, "version")
	writeFile(t, version, "1.0\n")
	writeFile(t, filepath.Join(bin, "herramienta"), "#!/bin/sh\ncat \""+version+"\"\n")
	if err := os.Chmod(filepath.Join(bin, "herramienta"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	r.config(t, `  - comando: 'echo x >> "`+r.cuenta+`"'`+"\n    rutas: [ui]\n    requiere: [herramienta]\n")
	r.auditar(t)
	r.auditar(t)
	if n := r.corridas(t); n != 1 {
		t.Fatalf("misma versión: corrió %d veces", n)
	}
	writeFile(t, version, "2.0\n")
	r.auditar(t)
	if n := r.corridas(t); n != 2 {
		t.Fatalf("versión nueva: corrió %d veces", n)
	}
}

func TestSuite_SiNoSeSabeLaVersionSiempreCorre(t *testing.T) {
	r := repoConSuite(t, "true")
	bin := t.TempDir()
	writeFile(t, filepath.Join(bin, "herramienta"), "#!/bin/sh\nexit 1\n")
	if err := os.Chmod(filepath.Join(bin, "herramienta"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	r.config(t, `  - comando: 'echo x >> "`+r.cuenta+`"'`+"\n    rutas: [ui]\n    requiere: [herramienta]\n")
	r.auditar(t)
	r.auditar(t)
	if n := r.corridas(t); n != 2 {
		t.Fatalf("corrió %d veces", n)
	}
}

func fichaRota(t *testing.T, raiz string) {
	t.Helper()
	writeFile(t, filepath.Join(raiz, ".qdd", "project", "modules", "rota.yaml"), "id: [sin cerrar\n")
}

func tieneRegla(violaciones []Violation, regla string) bool {
	for _, v := range violaciones {
		if v.RuleID == regla {
			return true
		}
	}
	return false
}

func TestParandoAlBloquear_UnaReglaRotaNoEsperaALasPruebas(t *testing.T) {
	r := repoConSuite(t, "true")
	fichaRota(t, r.raiz)
	violaciones := NewEngine(r.raiz).ParandoAlBloquear().RunAll()
	if r.corridas(t) != 0 {
		t.Fatal("las pruebas no corren mientras una regla bloquea")
	}
	if !tieneRegla(violaciones, "HEART-01-INVALID-RECORD") {
		t.Fatalf("dice lo que bloquea: %+v", violaciones)
	}
	if !tieneRegla(violaciones, ReglaPruebasPendientes) {
		t.Fatalf("dice que faltan las pruebas: %+v", violaciones)
	}
	if len(Bloqueantes(violaciones)) == 0 {
		t.Fatal("el gate sigue sin aprobar")
	}
	for _, v := range Bloqueantes(violaciones) {
		if v.RuleID == ReglaPruebasPendientes {
			t.Fatal("el aviso no bloquea por sí mismo: lo que bloquea es la regla")
		}
	}
}

// El peor error posible aquí: que el gate apruebe sin correr las pruebas.
func TestParandoAlBloquear_SinReglasQueBloqueenLasPruebasCorren(t *testing.T) {
	r := repoConSuite(t, "exit 3")
	motor := NewEngine(r.raiz).ParandoAlBloquear()
	aviso := Violation{RuleID: "X", Recomendacion: true}
	for _, reglas := range [][]Violation{nil, {aviso}} {
		violaciones := motor.pruebasOPendientes(QDDPolicies{}, reglas)
		if !tieneRegla(violaciones, "QA-01-ALL-TESTS-PASS") {
			t.Fatalf("con %+v las pruebas tenían que correr y bloquear: %+v", reglas, violaciones)
		}
	}
	if n := r.corridas(t); n != 2 {
		t.Fatalf("corrió %d veces", n)
	}
}

func TestParandoAlBloquear_EnElCIyConCompletoCorreTodo(t *testing.T) {
	for _, variable := range []string{"CI", envPruebasCompletas} {
		t.Run(variable, func(t *testing.T) {
			r := repoConSuite(t, "true")
			fichaRota(t, r.raiz)
			t.Setenv(variable, "1")
			violaciones := NewEngine(r.raiz).ParandoAlBloquear().RunAll()
			if r.corridas(t) != 1 || tieneRegla(violaciones, ReglaPruebasPendientes) {
				t.Fatalf("corrió %d veces: %+v", r.corridas(t), violaciones)
			}
		})
	}
}

// qdd report, qdd score y compañía miden: no se detienen, o el puntaje saldría de un gate a medias.
func TestRunAll_SinPedirloNoSeDetiene(t *testing.T) {
	r := repoConSuite(t, "true")
	fichaRota(t, r.raiz)
	violaciones := NewEngine(r.raiz).RunAll()
	if r.corridas(t) != 1 || tieneRegla(violaciones, ReglaPruebasPendientes) {
		t.Fatalf("corrió %d veces: %+v", r.corridas(t), violaciones)
	}
}

func TestRunAll_LasPruebasCorrenDespuesDeLasReglas(t *testing.T) {
	r := repoConSuite(t, "exit 3")
	fichaRota(t, r.raiz)
	violaciones := NewEngine(r.raiz).RunAll()
	ficha, prueba := -1, -1
	for i, v := range violaciones {
		if v.RuleID == "HEART-01-INVALID-RECORD" {
			ficha = i
		}
		if v.RuleID == "QA-01-ALL-TESTS-PASS" {
			prueba = i
		}
	}
	if ficha < 0 || prueba < 0 || ficha > prueba {
		t.Fatalf("ficha en %d, prueba en %d", ficha, prueba)
	}
}
