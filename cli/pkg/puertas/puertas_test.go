package puertas

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAcciones_VeLoQueCommiteaOPublica(t *testing.T) {
	casos := map[string][]Accion{
		"git commit -m 'feat: x'":                          {{Puerta: PuertaAudit, Que: "git commit"}},
		"cd cli && go test ./... && git commit -qm x":      {{Puerta: PuertaAudit, Dir: "cli", Que: "git commit"}},
		"git -C ../otro commit -m x":                       {{Puerta: PuertaAudit, Dir: "../otro", Que: "git commit"}},
		"git commit --no-verify -m x":                      {{Puerta: PuertaSalto, Que: "git commit --no-verify"}},
		"git commit -nm x":                                 {{Puerta: PuertaSalto, Que: "git commit --no-verify"}},
		"gh pr merge 301 --merge":                          {{Puerta: PuertaPublicar, Que: "gh pr merge"}},
		"gh release create v1.0.0":                         {{Puerta: PuertaPublicar, Que: "gh release create"}},
		"cd npm && npm publish --tag preview":              {{Puerta: PuertaPublicar, Dir: "npm", Que: "npm publish"}},
		"git push --tags":                                  {{Puerta: PuertaPublicar, Que: "git push --tags"}},
		"git push origin v2.10.0-rc.102":                   {{Puerta: PuertaPublicar, Que: "git push", Refs: []string{"v2.10.0-rc.102"}}},
		"git push -q -u origin fix/x":                      {{Puerta: PuertaPublicar, Que: "git push", Refs: []string{"fix/x"}}},
		"git -C /w push origin +refs/tags/v1:refs/tags/v1": {{Puerta: PuertaPublicar, Dir: "/w", Que: "git push", Refs: []string{"+refs/tags/v1:refs/tags/v1"}}},
	}
	for comando, want := range casos {
		got := Acciones(comando)
		if !igualesAcciones(got, want) {
			t.Errorf("%q:\n got %+v\nwant %+v", comando, got, want)
		}
	}
}

// Lo que va entre comillas o en un heredoc es texto: un mensaje de commit que nombra un comando no es ese comando.
func TestAcciones_ElTextoNoEsUnComando(t *testing.T) {
	for _, comando := range []string{
		`git status && echo "gh pr merge 3; npm publish"`,
		"git log --grep 'git commit --no-verify'",
		"cat > tmp/pr.md <<'EOF'\ngh pr merge 301\nnpm publish\nEOF\ngh pr view 301",
		"gh pr view 301 --json state",
		"npm view qdd-framework version",
		"qdd consola pedir -m 'Publicar' 'gh pr merge 301 --merge && git push origin v1'",
	} {
		if got := Acciones(comando); len(got) != 0 {
			t.Errorf("%q no commitea ni publica: %+v", comando, got)
		}
	}
	got := Acciones("git commit -q -m \"$(cat <<'EOF'\nfix: algo; gh pr merge 1\nEOF\n)\"")
	if len(got) != 1 || got[0].Puerta != PuertaAudit {
		t.Fatalf("un mensaje en heredoc dentro de comillas es un solo commit: %+v", got)
	}
}

func TestDecidir_Audit(t *testing.T) {
	activa := Politica{Activa: true}
	commit := Accion{Puerta: PuertaAudit, Que: "git commit"}
	if d := Decidir(Hechos{Accion: commit, Politica: activa, Marca: "abc", Huella: "abc"}); d.Resultado != Pasa || d.Mensaje != "" {
		t.Errorf("árbol igual al auditado pasa sin decir nada: %+v", d)
	}
	sin := Decidir(Hechos{Accion: commit, Politica: activa, Huella: "abc"})
	if sin.Resultado != Rechaza || !strings.Contains(sin.Mensaje, "qdd audit") {
		t.Errorf("sin audit aprobado se rechaza y dice qué correr: %+v", sin)
	}
	cambio := Decidir(Hechos{Accion: commit, Politica: activa, Marca: "abc", Huella: "def"})
	if cambio.Resultado != Rechaza || !strings.Contains(cambio.Mensaje, "cambió") {
		t.Errorf("un árbol que cambió desde el audit se rechaza: %+v", cambio)
	}
	if d := Decidir(Hechos{Accion: commit, Politica: activa, ErrHuella: os.ErrNotExist}); d.Resultado != Error {
		t.Errorf("si no puede calcular la huella deja pasar y anota el error: %+v", d)
	}
	if d := Decidir(Hechos{Accion: commit, Huella: "def"}); d.Resultado != Inactiva {
		t.Errorf("sin la política las puertas no deciden: %+v", d)
	}
}

func TestDecidir_PublicarYSalto(t *testing.T) {
	publicar := Accion{Puerta: PuertaPublicar, Que: "gh pr merge"}
	d := Decidir(Hechos{Accion: publicar, Politica: Politica{Activa: true}})
	if d.Resultado != Rechaza || !strings.Contains(d.Mensaje, "qdd consola pedir") {
		t.Errorf("publicar va como pedido: %+v", d)
	}
	if d := Decidir(Hechos{Accion: publicar, Politica: Politica{Activa: true, Publicar: PublicarIA}}); d.Resultado != Pasa {
		t.Errorf("con publicar: ia el proyecto deja publicar a la IA: %+v", d)
	}
	salto := Decidir(Hechos{Accion: Accion{Puerta: PuertaSalto, Que: "git commit --no-verify"}, Politica: Politica{Activa: true}})
	if salto.Resultado != Rechaza {
		t.Errorf("la IA no se salta las puertas: %+v", salto)
	}
	for _, m := range []string{d.Mensaje, salto.Mensaje, Decidir(Hechos{Accion: Accion{Puerta: PuertaAudit}, Politica: Politica{Activa: true}}).Mensaje} {
		if len([]rune(m)) > 280 {
			t.Errorf("un rechazo cabe en 280 caracteres (%d): %s", len([]rune(m)), m)
		}
	}
}

func TestHuella_CambiaConElArbolYNoConElIndice(t *testing.T) {
	dir := repoDePrueba(t)
	escribir(t, dir, "a.go", "package a\n")
	h1 := mustHuella(t, dir)
	if h1 != mustHuella(t, dir) {
		t.Fatal("la misma carpeta da la misma huella")
	}
	git(t, dir, "add", "a.go")
	if h1 != mustHuella(t, dir) {
		t.Fatal("pasar algo al índice no cambia el árbol de trabajo")
	}
	escribir(t, dir, "tmp/x.log", "ignorado\n")
	if h1 != mustHuella(t, dir) {
		t.Fatal("lo que git ignora no cuenta")
	}
	escribir(t, dir, "b.go", "package b\n")
	if h1 == mustHuella(t, dir) {
		t.Fatal("un archivo nuevo cambia la huella")
	}
	if out := git(t, dir, "status", "--porcelain"); !strings.Contains(out, "?? b.go") {
		t.Fatalf("calcular la huella no toca el índice real: %q", out)
	}
}

func TestMarca_GuardaYLee(t *testing.T) {
	dir := repoDePrueba(t)
	if m, err := LeerMarca(dir); err != nil || m != "" {
		t.Fatalf("sin audit no hay marca: %q %v", m, err)
	}
	if err := GuardarMarca(dir, "abc"); err != nil {
		t.Fatal(err)
	}
	if m, _ := LeerMarca(dir); m != "abc" {
		t.Fatalf("la marca se lee igual: %q", m)
	}
	if out := git(t, dir, "status", "--porcelain"); strings.Contains(out, "marca") {
		t.Fatalf("la marca vive fuera del árbol: %q", out)
	}
}

func TestPoliticaDe(t *testing.T) {
	casos := map[string]Politica{
		"zero_else: true\n":                         {},
		"puertas: {}\n":                             {Activa: true},
		"puertas:\n  publicar: ia\n":                {Activa: true, Publicar: PublicarIA},
		"puertas:\n  publicar: pedido\nowasp: true": {Activa: true, Publicar: PublicarPedido},
	}
	for yaml, want := range casos {
		if got := PoliticaDe([]byte(yaml)); got != want {
			t.Errorf("%q: got %+v want %+v", yaml, got, want)
		}
	}
}

func TestAnotar_FueraDelProyectoYSinRutas(t *testing.T) {
	home := t.TempDir()
	r := Registro{Puerta: PuertaAudit, Resultado: Rechaza, MS: 12, Sesion: "s-1", Largo: 90}
	if err := Anotar(home, "qdd", r); err != nil {
		t.Fatal(err)
	}
	datos, err := os.ReadFile(filepath.Join(home, "puertas", "qdd.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	linea := string(datos)
	for _, want := range []string{`"puerta":"audit"`, `"resultado":"rechaza"`, `"ms":12`, `"sesion":"s-1"`, `"largo":90`} {
		if !strings.Contains(linea, want) {
			t.Errorf("falta %s en %s", want, linea)
		}
	}
}

func igualesAcciones(a, b []Accion) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Puerta != b[i].Puerta || a[i].Dir != b[i].Dir || a[i].Que != b[i].Que || strings.Join(a[i].Refs, " ") != strings.Join(b[i].Refs, " ") {
			return false
		}
	}
	return true
}

func repoDePrueba(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	escribir(t, dir, ".gitignore", "tmp/\n")
	return dir
}

func mustHuella(t *testing.T, dir string) string {
	t.Helper()
	h, err := Huella(dir)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func escribir(t *testing.T, dir, ruta, texto string) {
	t.Helper()
	p := filepath.Join(dir, ruta)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(texto), 0o644); err != nil {
		t.Fatal(err)
	}
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func TestEsTag(t *testing.T) {
	dir := repoDePrueba(t)
	escribir(t, dir, "a.go", "package a\n")
	git(t, dir, "add", "-A")
	git(t, dir, "-c", "user.name=p", "-c", "user.email=p@p", "commit", "-qm", "x")
	git(t, dir, "tag", "v1.0.0")
	casos := map[string]bool{"v1.0.0": true, "+refs/tags/v2:refs/tags/v2": true, "main": false, "HEAD:fix/x": false}
	for ref, want := range casos {
		if got := EsTag(dir, ref); got != want {
			t.Errorf("%s: got %v want %v", ref, got, want)
		}
	}
}
