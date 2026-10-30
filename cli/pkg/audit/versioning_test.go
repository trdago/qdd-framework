package audit

import (
	"os/exec"
	"path/filepath"
	"testing"
)

// gitRepo crea un repo temporal con identidad propia para los commits.
func gitRepo(t *testing.T) (string, func(...string)) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git no disponible")
	}
	root := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-c", "user.name=qdd", "-c", "user.email=qdd@test", "-c", "commit.gpgsign=false", "-c", "tag.gpgsign=false"}, args...)...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q")
	return root, run
}

func setVersion(t *testing.T, root, version string) {
	writeFile(t, filepath.Join(root, "package.json"), `{"name":"x","version":"`+version+`"}`)
}

// releasedAt deja un repo con el tag v<version> publicado.
func releasedAt(t *testing.T, version string) (string, func(...string)) {
	root, run := gitRepo(t)
	setVersion(t, root, version)
	run("add", "-A")
	run("commit", "-qm", "chore(release): v"+version)
	run("tag", "v"+version)
	return root, run
}

// Caso real (QDD v1.9.23): dos 'feat' publicados como patch.
func TestVersioning_FeatShippedAsPatch(t *testing.T) {
	root, run := releasedAt(t, "1.9.22")
	run("commit", "-q", "--allow-empty", "-m", "feat(audit): gate veraz")
	run("commit", "-q", "--allow-empty", "-m", "fix(mcp): raíz del proyecto")
	setVersion(t, root, "1.9.23")
	run("commit", "-qam", "chore(release): v1.9.23")

	v := RunVersioningCheck(root)
	if !hasRule(v, "VER-03-UNDERSIZED-BUMP") {
		t.Fatalf("se esperaba VER-03, se obtuvo %v", v)
	}
}

func TestVersioning_FeatWithMinorBumpPasses(t *testing.T) {
	root, run := releasedAt(t, "1.9.22")
	run("commit", "-q", "--allow-empty", "-m", "feat(audit): gate veraz")
	setVersion(t, root, "1.10.0")
	run("commit", "-qam", "chore(release): v1.10.0")

	if v := RunVersioningCheck(root); len(v) != 0 {
		t.Fatalf("no se esperaban violaciones, se obtuvo %v", v)
	}
}

func TestVersioning_BreakingChangeRequiresMajor(t *testing.T) {
	for _, msg := range []string{"feat(api)!: elimina /v1", "refactor: nuevo formato\n\nBREAKING CHANGE: policies.yaml cambia"} {
		root, run := releasedAt(t, "2.3.1")
		run("commit", "-q", "--allow-empty", "-m", msg)
		setVersion(t, root, "2.4.0")
		run("commit", "-qam", "chore(release): v2.4.0")

		if !hasRule(RunVersioningCheck(root), "VER-03-UNDERSIZED-BUMP") {
			t.Errorf("'%s' debe exigir MAJOR", msg)
		}
	}
}

// En 0.y.z un cambio incompatible sube MINOR (SemVer §4).
func TestVersioning_PreOneBreakingNeedsOnlyMinor(t *testing.T) {
	root, run := releasedAt(t, "0.4.2")
	run("commit", "-q", "--allow-empty", "-m", "feat!: nuevo protocolo")
	setVersion(t, root, "0.5.0")
	run("commit", "-qam", "chore(release): v0.5.0")

	if v := RunVersioningCheck(root); len(v) != 0 {
		t.Fatalf("no se esperaban violaciones, se obtuvo %v", v)
	}
}

// Trabajo en curso: hay 'feat' después del tag pero aún no se prepara release.
func TestVersioning_UnreleasedWorkIsNotFlagged(t *testing.T) {
	root, run := releasedAt(t, "1.2.0")
	run("commit", "-q", "--allow-empty", "-m", "feat: algo nuevo")

	if v := RunVersioningCheck(root); len(v) != 0 {
		t.Fatalf("no se esperaban violaciones, se obtuvo %v", v)
	}
}

// CI de publicación: el tag apunta a HEAD y se evalúa el rango completo.
func TestVersioning_TagAtHeadIsEvaluated(t *testing.T) {
	root, run := releasedAt(t, "1.2.0")
	run("commit", "-q", "--allow-empty", "-m", "feat: algo nuevo")
	setVersion(t, root, "1.2.1")
	run("commit", "-qam", "chore(release): v1.2.1")
	run("tag", "v1.2.1")

	if !hasRule(RunVersioningCheck(root), "VER-03-UNDERSIZED-BUMP") {
		t.Fatal("un tag en HEAD con salto insuficiente debe fallar el gate de publicación")
	}
}

func TestVersioning_InvalidSemver(t *testing.T) {
	root := t.TempDir()
	setVersion(t, root, "1.9")
	if !hasRule(RunVersioningCheck(root), "VER-01-INVALID-SEMVER") {
		t.Fatal("se esperaba VER-01 para '1.9'")
	}
}

func TestVersioning_Regression(t *testing.T) {
	root, run := releasedAt(t, "2.0.0")
	setVersion(t, root, "1.9.9")
	run("commit", "-qam", "chore: vuelve atrás")

	if !hasRule(RunVersioningCheck(root), "VER-02-VERSION-REGRESSION") {
		t.Fatal("se esperaba VER-02")
	}
}

func TestVersioning_ChangelogMissingEntry(t *testing.T) {
	root, run := releasedAt(t, "1.0.0")
	writeFile(t, filepath.Join(root, "CHANGELOG.md"), "# Changelog\n\n## [1.0.0]\n- inicial\n")
	run("add", "-A")
	run("commit", "-qm", "docs: changelog")
	setVersion(t, root, "1.0.1")
	run("commit", "-qam", "chore(release): v1.0.1")

	if !hasRule(RunVersioningCheck(root), "VER-04-CHANGELOG-MISSING") {
		t.Fatal("se esperaba VER-04")
	}
}

// Sin tags (repo nuevo o checkout superficial de CI) no hay base que comparar.
func TestVersioning_NoTagsSkipsHistoryRules(t *testing.T) {
	root, run := gitRepo(t)
	setVersion(t, root, "3.1.4")
	run("add", "-A")
	run("commit", "-qm", "feat: inicial")

	if v := RunVersioningCheck(root); len(v) != 0 {
		t.Fatalf("no se esperaban violaciones, se obtuvo %v", v)
	}
}

func TestVersioning_PyprojectAndCargoSources(t *testing.T) {
	cases := map[string]string{
		"pyproject.toml": "[build-system]\nversion = \"9.9.9\"\n\n[project]\nname = \"x\"\nversion = \"1.2\"\n",
		"Cargo.toml":     "[package]\nname = \"x\"\nversion = \"1.2\"\n",
	}
	for name, content := range cases {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, name), content)
		if !hasRule(RunVersioningCheck(root), "VER-01-INVALID-SEMVER") {
			t.Errorf("%s: debe leer la versión de la sección del paquete", name)
		}
	}
}

func TestPolicies_VersioningDefaultsOnForLegacyPolicyFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".qdd", "policies.yaml"), "owasp: true\n")
	if !LoadPolicies(dir).VersioningEnabled() {
		t.Error("un policies.yaml sin la clave versioning debe dejarla activa")
	}
	writeFile(t, filepath.Join(dir, ".qdd", "policies.yaml"), "versioning: false\n")
	if LoadPolicies(dir).VersioningEnabled() {
		t.Error("versioning: false debe desactivarla explícitamente")
	}
}
