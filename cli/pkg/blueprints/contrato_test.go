package blueprints

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/modules"
)

// Rule: Prohibido usar 'else'

// accesoConPreguntaNueva es acceso (que reemplaza a auth) con una pregunta que auth no tenía.
var accesoConPreguntaNueva = strings.Replace(accesoQueReemplazaAAuth, "preguntas:\n", `preguntas:
  - id: segundo_factor
    pregunta: ¿Se pide un segundo factor al iniciar sesión?
    por_defecto: "no"
    opciones:
      - valor: "no"
      - valor: totp
`, 1)

// Migrar a un blueprint que reemplaza a otro conserva el id del módulo y sus respuestas, y la pregunta nueva queda
// para la persona: QDD no la responde, ni con su valor sugerido, y no escribe la ficha hasta que ella responda.
// contrato: blueprints#043f2707
func TestReemplazo_LaPreguntaNuevaEsDeLaPersonaYElModuloConservaSuID(t *testing.T) {
	repo := gitOrgRepo(t)
	root := proyectoConAuthAplicado(t, repo)
	writeFile(t, filepath.Join(repo, "acceso", ManifestName), accesoConPreguntaNueva)
	gitIn(t, repo, "add", ".")
	gitIn(t, repo, "commit", "-q", "-m", "acceso reemplaza a auth y pregunta por el segundo factor")
	gitIn(t, repo, "tag", "v2.0.0")
	if _, err := Sincronizar(root); err != nil {
		t.Fatal(err)
	}
	fichaAntes, err := os.ReadFile(filepath.Join(root, ".qdd", "project", "modules", "crm.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	if _, problems := AddManaged(AddInput{Root: root, Blueprints: []string{"acceso"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	a, err := LoadApplication(root, "crm")
	if err != nil {
		t.Fatal(err)
	}
	if a.Modulo != "crm" || a.Blueprint != "acceso" {
		t.Fatalf("la migración cambió el módulo: %+v", a)
	}
	if v, ok := a.Respuestas["segundo_factor"]; ok {
		t.Fatalf("la migración respondió por la persona la pregunta nueva: segundo_factor=%q", v)
	}

	res, problems := Apply(ApplyInput{Root: root, Blueprint: "acceso", Modulo: "crm"})
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	if len(res.Pendientes) != 1 || res.Pendientes[0].ID != "segundo_factor" {
		t.Fatalf("volver a aplicarlo pregunta lo nuevo a la persona: %+v", res.Pendientes)
	}
	fichaDespues, _ := os.ReadFile(filepath.Join(root, ".qdd", "project", "modules", "crm.yaml"))
	if string(fichaAntes) != string(fichaDespues) {
		t.Fatal("con una pregunta sin responder se escribió la ficha")
	}
	if m, ok := modules.Find(root, "crm"); !ok || m.ID != "crm" {
		t.Fatalf("la ficha sigue siendo crm: %+v %v", m, ok)
	}
}

// Una receta certificada que cambia después, o cuya certificación no se aprobó, no se propone.
// contrato: blueprints#9eda5071
func TestProponer_UnaCertificacionVencidaONoAprobadaNoAlcanza(t *testing.T) {
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
	manifest := filepath.Join(ap.Carpeta, ManifestName)
	writeFile(t, manifest, recetaConLimites)
	certificarParaPrueba(t, ap.Carpeta)
	cert := filepath.Join(ap.Carpeta, CertificacionFile)
	certificada, err := os.ReadFile(cert)
	if err != nil {
		t.Fatal(err)
	}

	writeFile(t, manifest, strings.Replace(recetaConLimites, "la tabla con una tecla por tarea", "la tabla", 1))
	if _, err := Proponer(root, "panel", "nuevo", ""); err == nil || !strings.Contains(err.Error(), "certify") {
		t.Fatalf("una receta que cambió después de certificarse se propuso: %v", err)
	}

	writeFile(t, manifest, recetaConLimites)
	writeFile(t, cert, strings.Replace(string(certificada), "aprobada: true", "aprobada: false", 1))
	if _, err := Proponer(root, "panel", "nuevo", ""); err == nil || !strings.Contains(err.Error(), "certify") {
		t.Fatalf("una certificación no aprobada alcanzó para proponer: %v", err)
	}
}

// Un valor de negocio sin respuesta queda para la persona aunque tenga un sugerido: Apply no lo elige ni escribe la
// ficha con un {{id}} pendiente.
// contrato: blueprints#9eda5071
func TestApply_SinElValorDeNegocioNoEscribeNada(t *testing.T) {
	root := t.TempDir()
	if _, problems := Add(AddInput{Root: root, Origen: sourceBlueprint(t, recetaConLimites)}); len(problems) > 0 {
		t.Fatal(problems)
	}
	res, problems := Apply(ApplyInput{Root: root, Blueprint: "panel", Rutas: []string{"internal/panel"}, Respuestas: map[string]string{"avisos": "si"}})
	if len(problems) > 0 || len(res.Pendientes) != 1 {
		t.Fatalf("el valor de negocio queda pendiente: %+v %v", res.Pendientes, problems)
	}
	if _, ok := modules.Find(root, "panel"); ok {
		t.Fatal("sin el valor de negocio se escribió la ficha (con el sugerido o con {{visibles}})")
	}
	if res.Ficha != "" {
		t.Fatalf("Apply dice que escribió %s con una decisión pendiente", res.Ficha)
	}
}

// El arreglo que sugiere cada hallazgo de calidad no quita una regla, una opción ni un escenario: solo quita lo que
// sobra (una regla repetida de la base, un término, parámetro o constante sin uso).
// contrato: blueprints#16089f78
func TestCalidad_NingunArregloQuitaReglaOpcionNiEscenario(t *testing.T) {
	quitanLoQueSobra := map[string]bool{CalRepetido: true, CalTerminoSinUso: true, CalSinUso: true}
	vistos := 0
	for _, receta := range []string{recetaConDefectos, recetaConLimites, authBlueprint} {
		for _, h := range CalidadDe(mustParse(t, receta)).Hallazgos {
			vistos++
			arreglo := strings.ReplaceAll(strings.ToLower(h.Arreglo), "sin quitar", "")
			quita := strings.Contains(arreglo, "quít") || strings.Contains(arreglo, "quit") || strings.Contains(arreglo, "borra") || strings.Contains(arreglo, "elimina")
			if quita && !quitanLoQueSobra[h.Codigo] {
				t.Errorf("%s %s sugiere quitar algo de la receta: %q", h.Codigo, h.Donde, h.Arreglo)
			}
		}
	}
	if vistos == 0 {
		t.Fatal("las recetas de prueba no tienen hallazgos que revisar")
	}
}
