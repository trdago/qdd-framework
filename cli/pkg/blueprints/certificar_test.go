package blueprints

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func recetaEnCarpeta(t *testing.T, contenido string) Blueprint {
	t.Helper()
	t.Setenv("QDD_HOME", t.TempDir())
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ManifestName), contenido)
	b, err := ParseDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func varianteDe(t *testing.T, k Kit, nombre string) Variante {
	t.Helper()
	for _, v := range k.Variantes {
		if v.Nombre == nombre {
			return v
		}
	}
	t.Fatalf("no hay variante %s: %+v", nombre, k.Variantes)
	return Variante{}
}

func leer(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestCertificar_CadaVarianteSoloConLaReceta(t *testing.T) {
	b := recetaEnCarpeta(t, recetaConLimites)
	k, err := PrepararCertificacion(b)
	if err != nil {
		t.Fatal(err)
	}
	sugerida, otra := varianteDe(t, k, VarianteSugerida), varianteDe(t, k, VarianteAdaptada)
	if sugerida.Respuestas["visibles"] != "4" || otra.Respuestas["visibles"] != "10" || otra.Respuestas["avisos"] != "si" {
		t.Fatalf("una variante con lo sugerido y otra con otros valores de negocio: %+v %+v", sugerida.Respuestas, otra.Respuestas)
	}
	receta := leer(t, filepath.Join(otra.Carpeta, RecetaFile))
	for _, want := range []string{"Leer las tareas cada 2 s y mostrar las 10 más nuevas", "P-02", "construccion.yaml", "✗ Editar tareas"} {
		if !strings.Contains(receta, want) {
			t.Fatalf("la receta de la variante trae sus valores, sus escenarios y cómo entregar: falta %q en\n%s", want, receta)
		}
	}
	if strings.Contains(leer(t, filepath.Join(sugerida.Carpeta, RecetaFile)), "P-02") {
		t.Fatal("sin avisos, la variante sugerida no tiene el escenario del aviso")
	}
	entradas, _ := os.ReadDir(otra.Carpeta)
	if len(entradas) != 1 || strings.HasPrefix(otra.Carpeta, b.Dir) {
		t.Fatalf("a ciegas: la carpeta tiene solo la receta y está fuera del blueprint: %v", entradas)
	}
}

func construir(t *testing.T, v Variante, comando string, escenarios ...string) {
	t.Helper()
	writeFile(t, filepath.Join(v.Carpeta, ConstruccionFile), "lenguaje: shell\ncomando: \""+comando+"\"\n")
	writeFile(t, filepath.Join(v.Carpeta, "pruebas", "pruebas.sh"), "# "+strings.Join(escenarios, " ")+"\n")
}

func TestCertificar_LoTecnicoSeReplicaYLoDeNegocioSeAdapta(t *testing.T) {
	b := recetaEnCarpeta(t, recetaConLimites)
	k, _ := PrepararCertificacion(b)
	c, err := VerificarCertificacion(b)
	if err != nil || c.Aprobada || !strings.Contains(c.Variantes[0].Resultado, "sin construir") {
		t.Fatalf("sin construir no se certifica: %+v %v", c, err)
	}
	construir(t, varianteDe(t, k, VarianteSugerida), "true", "P-01")
	construir(t, varianteDe(t, k, VarianteAdaptada), "true", "P-01")
	c, _ = VerificarCertificacion(b)
	if c.Aprobada || !strings.Contains(LineaCertificacion(c), "P-02") {
		t.Fatalf("un escenario sin prueba en una variante no se certifica, y se nombra: %q", LineaCertificacion(c))
	}
	construir(t, varianteDe(t, k, VarianteAdaptada), "false", "P-01", "P-02")
	if c, _ = VerificarCertificacion(b); c.Aprobada {
		t.Fatal("pruebas que fallan no se certifican")
	}
	construir(t, varianteDe(t, k, VarianteAdaptada), "true", "P-01", "P-02")
	c, _ = VerificarCertificacion(b)
	if !c.Aprobada || c.Huella == "" {
		t.Fatalf("las dos variantes construidas y probadas: certificada: %+v", c)
	}
	if guardada, ok := CertificacionVigente(b); !ok || guardada.Huella != c.Huella {
		t.Fatalf("la certificación queda junto al blueprint: %+v", guardada)
	}
	writeFile(t, filepath.Join(b.Dir, ManifestName), strings.Replace(recetaConLimites, "version: 1.0.0", "version: 2.0.0", 1))
	if _, ok := CertificacionVigente(b); !ok {
		t.Fatal("la versión la fija QDD al proponer: no vence la certificación")
	}
	writeFile(t, filepath.Join(b.Dir, ManifestName), strings.Replace(recetaConLimites, "la tabla con una tecla por tarea", "la tabla", 1))
	if _, ok := CertificacionVigente(b); ok {
		t.Fatal("un cambio en la receta vence la certificación")
	}
}

func TestProponer_UnaRecetaConPasosSeCertificaAntes(t *testing.T) {
	sinGH(t)
	repo := gitOrgRepo(t)
	root := t.TempDir()
	if _, problems := AddManaged(AddInput{Root: root, Origen: repo, Blueprints: []string{"auth"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	ap, err := Aportar(root, "panel", "")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(ap.Carpeta, ManifestName), recetaConLimites)
	if _, err := Proponer(root, "panel", "nuevo", ""); err == nil || !strings.Contains(err.Error(), "certify") {
		t.Fatalf("sin certificación ciega no se propone: %v", err)
	}
	certificarParaPrueba(t, ap.Carpeta)
	p, err := Proponer(root, "panel", "nuevo", "")
	if err != nil || !strings.Contains(p.Certificacion, "2/2") {
		t.Fatalf("certificada, se propone y el PR lo dice: %+v %v", p, err)
	}
}

// certificarParaPrueba construye las dos variantes como lo haría un agente a ciegas.
func certificarParaPrueba(t *testing.T, carpeta string) {
	t.Helper()
	b, err := ParseDir(carpeta)
	if err != nil {
		t.Fatal(err)
	}
	k, err := PrepararCertificacion(b)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range k.Variantes {
		construir(t, v, "true", v.Escenarios...)
	}
	if c, err := VerificarCertificacion(b); err != nil || !c.Aprobada {
		t.Fatalf("certificación de prueba: %+v %v", c, err)
	}
}

func TestCertificar_LaRecetaDiceQueSignificaCadaRespuesta(t *testing.T) {
	k, _ := PrepararCertificacion(recetaEnCarpeta(t, recetaConLimites))
	if receta := leer(t, filepath.Join(varianteDe(t, k, VarianteAdaptada).Carpeta, RecetaFile)); !strings.Contains(receta, "avisos → si: Avisa cuando llega una tarea.") {
		t.Fatalf("quien construye a ciegas no sabe qué significa «si» sin la descripción de la opción:\n%s", receta)
	}
}

func TestCertificar_UnLimiteDeUnaOpcionSoloConEsaOpcion(t *testing.T) {
	conLimite := strings.Replace(strings.Replace(recetaConLimites, `incluye: ["Ver las tareas y avisar cuando llega una"]`, `incluye: ["Ver las tareas"]`, 1),
		"        descripcion: Avisa cuando llega una tarea.\n", "        descripcion: Avisa cuando llega una tarea.\n        alcance: {incluye: [\"Avisar cuando llega una tarea\"]}\n", 1)
	conLimite = strings.Replace(conLimite, "        descripcion: Sin avisos.\n", "        descripcion: Sin avisos.\n        alcance: {excluye: [\"Avisos\"]}\n", 1)
	b := recetaEnCarpeta(t, conLimite)
	k, err := PrepararCertificacion(b)
	if err != nil {
		t.Fatal(err)
	}
	sugerida, otra := leer(t, filepath.Join(varianteDe(t, k, VarianteSugerida).Carpeta, RecetaFile)), leer(t, filepath.Join(varianteDe(t, k, VarianteAdaptada).Carpeta, RecetaFile))
	if strings.Contains(sugerida, "✓ Avisar") || !strings.Contains(sugerida, "✗ Avisos") || !strings.Contains(otra, "✓ Avisar cuando llega una tarea") || strings.Contains(otra, "✗ Avisos") {
		t.Fatalf("el alcance de cada variante es el suyo:\n%s\n---\n%s", sugerida, otra)
	}
	if s, _ := Seccion(b, "alcance"); !strings.Contains(s, "Avisar cuando llega una tarea") || !strings.Contains(s, "avisos=si") {
		t.Fatalf("la sección alcance muestra los límites de cada opción y cuándo: %q", s)
	}
}

func TestCertificar_LoQueQuienConstruyeSupusoVuelveALaPersona(t *testing.T) {
	b := recetaEnCarpeta(t, recetaConLimites)
	k, _ := PrepararCertificacion(b)
	if receta := leer(t, filepath.Join(k.Variantes[0].Carpeta, RecetaFile)); !strings.Contains(receta, "supuestos:") {
		t.Fatalf("la receta pide dejar lo que hubo que suponer:\n%s", receta)
	}
	for _, v := range k.Variantes {
		construir(t, v, "true", v.Escenarios...)
	}
	writeFile(t, filepath.Join(k.Variantes[1].Carpeta, ConstruccionFile), "lenguaje: shell\ncomando: \"true\"\nsupuestos: [\"P-01: qué es una tarea nueva\"]\n")
	c, err := VerificarCertificacion(b)
	if err != nil || !c.Aprobada {
		t.Fatalf("un supuesto no rompe lo construido: %+v %v", c, err)
	}
	if s := SupuestosDe(c); len(s) != 1 || !strings.Contains(s[0], "P-01: qué es una tarea nueva") || !strings.Contains(LineaCertificacion(c), "1 supuesto") {
		t.Fatalf("lo supuesto queda en la certificación para preguntárselo a la persona: %v %q", s, LineaCertificacion(c))
	}
}
