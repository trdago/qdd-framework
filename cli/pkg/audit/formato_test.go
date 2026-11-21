package audit

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// MEJ-061 / FND-203: en sfrc qdd audit dio 0 y el CI rechazó el commit por gofmt. El gate revisa el formato con el formateador
// que usa el proyecto, como el CI; sin el formateador lo dice y no bloquea.

const goSinFormato = "package x\n\nfunc  F( ) int {\nreturn 1\n}\n"
const goConFormato = "package x\n\nfunc F() int {\n\treturn 1\n}\n"

func violacionesDe(vs []Violation, regla string) []Violation {
	var out []Violation
	for _, v := range vs {
		if v.RuleID == regla {
			out = append(out, v)
		}
	}
	return out
}

// programaFalso deja un ejecutable que imprime salida y termina con codigo, como lo haría el formateador real.
func programaFalso(t *testing.T, ruta, salida string, codigo int) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("el formateador falso es un script de sh")
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		t.Fatal(err)
	}
	script := fmt.Sprintf("#!/bin/sh\ncat <<'FIN'\n%s\nFIN\nexit %d\n", salida, codigo)
	if err := os.WriteFile(ruta, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestFormato_GoSinFormatoBloqueaYNombraElArchivo(t *testing.T) {
	dir := proyectoCon(t, map[string]string{"go.mod": "module x\n", "a.go": goConFormato, "pkg/b.go": goSinFormato})
	vs := violacionesDe(RunFormatoCheck(dir), ReglaFormato)
	if len(vs) != 1 || vs[0].Recomendacion {
		t.Fatalf("un .go sin gofmt bloquea el gate: %+v", vs)
	}
	if !strings.Contains(vs[0].Description, "pkg/b.go") || strings.Contains(vs[0].Description, "a.go,") {
		t.Fatalf("nombra solo el archivo sin formato: %s", vs[0].Description)
	}
	if !strings.Contains(vs[0].Description, "gofmt -w pkg/b.go") {
		t.Fatalf("dice cómo arreglarlo: %s", vs[0].Description)
	}
}

func TestFormato_MuchosArchivosSeCuentanSinListarlosTodos(t *testing.T) {
	archivos := map[string]string{"go.mod": "module x\n"}
	for i := range 12 {
		archivos[fmt.Sprintf("p/f%02d.go", i)] = goSinFormato
	}
	vs := violacionesDe(RunFormatoCheck(proyectoCon(t, archivos)), ReglaFormato)
	if len(vs) != 1 || !strings.Contains(vs[0].Description, "12 archivo(s)") || !strings.Contains(vs[0].Description, "y 2 más") || strings.Contains(vs[0].Description, "p/f11.go") {
		t.Fatalf("una sola violación que cuenta todos y nombra 10: %+v", vs)
	}
}

func TestFormato_GoConFormatoNoDiceNada(t *testing.T) {
	dir := proyectoCon(t, map[string]string{"go.mod": "module x\n", "a.go": goConFormato})
	if vs := RunFormatoCheck(dir); len(vs) != 0 {
		t.Fatalf("todo formateado no da violaciones: %+v", vs)
	}
}

// FND-203: el caso de sfrc, con el módulo de Go en backend/ y el CI corriendo gofmt ahí.
func TestFormato_FND203_GoEnUnaSubcarpetaTambienSeRevisa(t *testing.T) {
	dir := proyectoCon(t, map[string]string{"backend/go.mod": "module x\n", "backend/internal/h.go": goSinFormato, "frontend/package.json": "{}"})
	vs := violacionesDe(RunFormatoCheck(dir), ReglaFormato)
	if len(vs) != 1 || !strings.Contains(vs[0].Description, "backend/internal/h.go") {
		t.Fatalf("el .go de backend/ sin formato bloquea: %+v", vs)
	}
}

func TestFormato_GoIgnoraTestdataVendorYLoQueGitIgnora(t *testing.T) {
	dir := proyectoCon(t, map[string]string{
		"go.mod":                  "module x\n",
		"a.go":                    goConFormato,
		"testdata/roto.go":        goSinFormato,
		"vendor/m/m.go":           goSinFormato,
		"node_modules/p/x.go":     goSinFormato,
		"otro/.git/HEAD":          "ref: refs/heads/main\n",
		"otro/sin_formato.go":     goSinFormato,
		"tmp/.gitignore":          "*\n",
		"generado/sin_formato.go": goSinFormato,
	})
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("generado/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitDePrueba(t, dir, "init", "-q")
	if vs := RunFormatoCheck(dir); len(vs) != 0 {
		t.Fatalf("testdata, vendor, node_modules, repos anidados y carpetas ignoradas no se revisan: %+v", vs)
	}
}

func TestFormato_SinGofmtLoDiceYNoBloquea(t *testing.T) {
	antes := binGofmt
	binGofmt = "gofmt-que-no-existe"
	t.Cleanup(func() { binGofmt = antes })
	dir := proyectoCon(t, map[string]string{"go.mod": "module x\n", "a.go": goSinFormato})
	vs := RunFormatoCheck(dir)
	if len(vs) != 1 || vs[0].RuleID != ReglaFormatoSinMedir || !vs[0].Recomendacion {
		t.Fatalf("sin gofmt queda sin medir, como recomendación: %+v", vs)
	}
	if len(violacionesDe(vs, ReglaFormato)) != 0 {
		t.Fatal("sin el formateador nunca se informa un formato malo ni bueno")
	}
}

func TestFormato_SinGoNiFormateadoresNoDiceNada(t *testing.T) {
	dir := proyectoCon(t, map[string]string{"README.md": "# x\n", "package.json": `{"scripts":{"test":"vitest"}}`})
	if vs := RunFormatoCheck(dir); len(vs) != 0 {
		t.Fatalf("un proyecto que no usa formateador no se mide: %+v", vs)
	}
}

func TestFormato_PrettierConfiguradoBloquea(t *testing.T) {
	dir := proyectoCon(t, map[string]string{"package.json": `{"devDependencies":{"prettier":"^3"}}`, "src/a.ts": "x"})
	programaFalso(t, filepath.Join(dir, "node_modules/.bin/prettier"), "Checking formatting...\n[warn] src/a.ts\n[warn] Code style issues found in the above file. Run Prettier with --write to fix.", 1)
	vs := violacionesDe(RunFormatoCheck(dir), ReglaFormato)
	if len(vs) != 1 || vs[0].Recomendacion || !strings.Contains(vs[0].Description, "src/a.ts") || strings.Contains(vs[0].Description, "Code style") {
		t.Fatalf("prettier que marca archivos bloquea y los nombra: %+v", vs)
	}
}

func TestFormato_PrettierEnLaSubcarpetaDelFrontend(t *testing.T) {
	dir := proyectoCon(t, map[string]string{"frontend/package.json": "{}", "frontend/.prettierrc": "{}"})
	programaFalso(t, filepath.Join(dir, "frontend/node_modules/.bin/prettier"), "[warn] src/App.tsx", 1)
	vs := violacionesDe(RunFormatoCheck(dir), ReglaFormato)
	if len(vs) != 1 || !strings.Contains(vs[0].Description, "frontend/src/App.tsx") {
		t.Fatalf("prettier de frontend/ corre ahí y nombra la ruta desde la raíz: %+v", vs)
	}
}

func TestFormato_PrettierAlDiaNoDiceNada(t *testing.T) {
	dir := proyectoCon(t, map[string]string{"package.json": "{}", "prettier.config.js": "export default {}"})
	programaFalso(t, filepath.Join(dir, "node_modules/.bin/prettier"), "All matched files use Prettier code style!", 0)
	if vs := RunFormatoCheck(dir); len(vs) != 0 {
		t.Fatalf("prettier sin reclamos no da violaciones: %+v", vs)
	}
}

func TestFormato_PrettierSinInstalarQuedaSinMedir(t *testing.T) {
	dir := proyectoCon(t, map[string]string{"package.json": "{}", ".prettierrc.json": "{}"})
	vs := RunFormatoCheck(dir)
	if len(vs) != 1 || vs[0].RuleID != ReglaFormatoSinMedir || !vs[0].Recomendacion || !strings.Contains(vs[0].Description, "prettier") {
		t.Fatalf("prettier configurado sin node_modules queda sin medir: %+v", vs)
	}
}

func TestFormato_PrettierInstaladoSinConfiguracionNoCorre(t *testing.T) {
	dir := proyectoCon(t, map[string]string{"package.json": `{"dependencies":{"react":"19"}}`})
	programaFalso(t, filepath.Join(dir, "node_modules/.bin/prettier"), "[warn] a.js", 1)
	if vs := RunFormatoCheck(dir); len(vs) != 0 {
		t.Fatalf("prettier que trae otra dependencia no es el formateador del proyecto: %+v", vs)
	}
}

func TestFormato_PrettierQueFallaPorOtraCosaNoInventaArchivos(t *testing.T) {
	dir := proyectoCon(t, map[string]string{"package.json": "{}", ".prettierrc": "{"})
	programaFalso(t, filepath.Join(dir, "node_modules/.bin/prettier"), "[error] Invalid configuration file .prettierrc", 2)
	vs := RunFormatoCheck(dir)
	if len(vs) != 1 || vs[0].RuleID != ReglaFormatoSinMedir || !strings.Contains(vs[0].Description, "Invalid configuration") {
		t.Fatalf("un prettier que no pudo revisar queda sin medir con su motivo: %+v", vs)
	}
}

func TestFormato_BiomeConfiguradoBloquea(t *testing.T) {
	dir := proyectoCon(t, map[string]string{"package.json": "{}", "biome.json": "{}"})
	programaFalso(t, filepath.Join(dir, "node_modules/.bin/biome"), "./src/a.ts format ━━━━━━━━━━\n  × Formatter would have printed the following content:\nChecked 4 files in 3ms. No fixes applied.\nFound 1 error.", 1)
	vs := violacionesDe(RunFormatoCheck(dir), ReglaFormato)
	if len(vs) != 1 || !strings.Contains(vs[0].Description, "src/a.ts") || !strings.Contains(vs[0].Description, "biome") {
		t.Fatalf("biome que marca archivos bloquea: %+v", vs)
	}
}

func TestFormato_RuffFormatDeclaradoBloquea(t *testing.T) {
	dir := proyectoCon(t, map[string]string{"pyproject.toml": "[project]\nname='x'\n[tool.ruff.format]\nquote-style='double'\n", "app/main.py": "x"})
	programaFalso(t, filepath.Join(dir, ".venv/bin/ruff"), "Would reformat: app/main.py\n1 file would be reformatted", 1)
	vs := violacionesDe(RunFormatoCheck(dir), ReglaFormato)
	if len(vs) != 1 || !strings.Contains(vs[0].Description, "app/main.py") || !strings.Contains(vs[0].Description, "ruff format") {
		t.Fatalf("ruff format que marca archivos bloquea: %+v", vs)
	}
}

func TestFormato_RuffSoloParaLintNoExigeFormato(t *testing.T) {
	dir := proyectoCon(t, map[string]string{"pyproject.toml": "[project]\nname='x'\n[tool.ruff.lint]\nselect=['E']\n"})
	programaFalso(t, filepath.Join(dir, ".venv/bin/ruff"), "Would reformat: a.py", 1)
	if vs := RunFormatoCheck(dir); len(vs) != 0 {
		t.Fatalf("un proyecto que usa ruff solo como linter no se mide con ruff format: %+v", vs)
	}
}

func TestFormato_RuffFormatEnElCI(t *testing.T) {
	dir := proyectoCon(t, map[string]string{"requirements.txt": "fastapi\n", ".github/workflows/ci.yml": "steps:\n  - run: ruff format --check .\n"})
	programaFalso(t, filepath.Join(dir, ".venv/bin/ruff"), "Would reformat: api.py", 1)
	if vs := violacionesDe(RunFormatoCheck(dir), ReglaFormato); len(vs) != 1 {
		t.Fatalf("lo que el CI corre, el gate también: %+v", vs)
	}
}

func TestFormato_BlackDeclaradoBloquea(t *testing.T) {
	dir := proyectoCon(t, map[string]string{"pyproject.toml": "[tool.black]\nline-length = 100\n"})
	programaFalso(t, filepath.Join(dir, "venv/bin/black"), "would reformat /tmp/x/svc.py\nOh no! 1 file would be reformatted.", 1)
	vs := violacionesDe(RunFormatoCheck(dir), ReglaFormato)
	if len(vs) != 1 || !strings.Contains(vs[0].Description, "svc.py") || !strings.Contains(vs[0].Description, "black") {
		t.Fatalf("black que marca archivos bloquea: %+v", vs)
	}
}

func TestFormato_DartFormatEnElCI(t *testing.T) {
	dir := proyectoCon(t, map[string]string{"pubspec.yaml": "name: app\n", ".github/workflows/flutter.yml": "steps:\n  - run: dart format --output=none --set-exit-if-changed .\n"})
	antes := binDart
	binDart = filepath.Join(dir, "bin-falso/dart")
	t.Cleanup(func() { binDart = antes })
	programaFalso(t, binDart, "Changed lib/main.dart\nFormatted 3 files (1 changed) in 0.1 seconds.", 1)
	vs := violacionesDe(RunFormatoCheck(dir), ReglaFormato)
	if len(vs) != 1 || !strings.Contains(vs[0].Description, "lib/main.dart") {
		t.Fatalf("dart format del CI bloquea: %+v", vs)
	}
}

func TestFormato_DartSinFormatoEnElCINoSeMide(t *testing.T) {
	dir := proyectoCon(t, map[string]string{"pubspec.yaml": "name: app\n"})
	antes := binDart
	binDart = filepath.Join(dir, "bin-falso/dart")
	t.Cleanup(func() { binDart = antes })
	programaFalso(t, binDart, "Changed lib/main.dart", 1)
	if vs := RunFormatoCheck(dir); len(vs) != 0 {
		t.Fatalf("Flutter sin dart format en el CI no se mide: %+v", vs)
	}
}

func TestFormato_EsReglaDelMotor(t *testing.T) {
	dir := proyectoCon(t, map[string]string{"go.mod": "module x\n", "a.go": goSinFormato, ".qdd/policies.yaml": "adopcion: gradual\n"})
	vs := violacionesDe(NewEngine(dir).ParandoAlBloquear().RunAll(), ReglaFormato)
	if len(Bloqueantes(vs)) != 1 {
		t.Fatalf("el gate bloquea por formato aunque la adopción sea gradual: %+v", vs)
	}
}
