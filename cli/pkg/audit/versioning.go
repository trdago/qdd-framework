package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Reglas de versionamiento semántico (CERT-024-SEMVER-RELEASES):
//   - VER-01-INVALID-SEMVER: la versión declarada no es SemVer válido.
//   - VER-02-VERSION-REGRESSION: la versión declarada es menor que un tag ya publicado.
//   - VER-03-UNDERSIZED-BUMP: los commits desde el último tag exigen un salto mayor
//     que el declarado (feat → minor, cambio incompatible → major).
//   - VER-04-CHANGELOG-MISSING: el CHANGELOG lleva entradas por versión y falta
//     la de la versión que se va a publicar.
//   - VER-05-UNPROVEN-RELEASE (uso_release.go): con uso_minimo_horas, una minor o
//     una major pasa de preview a latest solo con sus horas de uso real medidas.
// Sin tags alcanzables (repo nuevo o clon superficial) solo se valida VER-01.

var semverPattern = regexp.MustCompile(`^v?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-([0-9A-Za-z.-]+))?(?:\+[0-9A-Za-z.-]+)?$`)

var conventionalPattern = regexp.MustCompile(`^(\w+)(\([^)]*\))?(!)?:\s`)

var breakingFooterPattern = regexp.MustCompile(`(?m)^BREAKING[ -]CHANGE:`)

const (
	bumpNone = iota
	bumpPatch
	bumpMinor
	bumpMajor
)

var bumpNames = map[int]string{bumpNone: "ninguno", bumpPatch: "patch", bumpMinor: "minor", bumpMajor: "major"}

type semver struct {
	raw   string
	core  [3]int
	isPre bool
}

func parseSemver(s string) (semver, bool) {
	m := semverPattern.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return semver{}, false
	}
	v := semver{raw: strings.TrimPrefix(strings.TrimSpace(s), "v"), isPre: m[4] != ""}
	for i := 0; i < 3; i++ {
		v.core[i], _ = strconv.Atoi(m[i+1])
	}
	return v, true
}

// cmpCore compara solo major.minor.patch.
func cmpCore(a, b semver) int {
	for i := 0; i < 3; i++ {
		if a.core[i] != b.core[i] {
			return a.core[i] - b.core[i]
		}
	}
	return 0
}

func (v semver) String() string {
	return fmt.Sprintf("%d.%d.%d", v.core[0], v.core[1], v.core[2])
}

// bumpTo devuelve la siguiente versión exigida por el tipo de salto.
func (v semver) bumpTo(kind int) semver {
	next := semver{core: v.core}
	if kind == bumpMajor {
		next.core = [3]int{v.core[0] + 1, 0, 0}
		return next
	}
	if kind == bumpMinor {
		next.core = [3]int{v.core[0], v.core[1] + 1, 0}
		return next
	}
	next.core[2]++
	return next
}

// RunVersioningCheck valida la versión declarada del proyecto contra SemVer y
// contra los cambios acumulados desde el último tag.
func RunVersioningCheck(cwd string) []Violation {
	source, raw := declaredVersion(cwd)
	if raw == "" {
		return nil
	}
	current, ok := parseSemver(raw)
	if !ok {
		return []Violation{versionViolation("VER-01-INVALID-SEMVER", source,
			fmt.Sprintf("La versión '%s' no es SemVer válido (MAJOR.MINOR.PATCH, p. ej. 1.4.0).", raw))}
	}

	tags := mergedSemverTags(cwd)
	if len(tags) == 0 {
		return nil
	}
	if latest := highestTag(tags); cmpCore(latest, current) > 0 {
		return []Violation{versionViolation("VER-02-VERSION-REGRESSION", source,
			fmt.Sprintf("La versión declarada %s es menor que el tag ya publicado v%s.", current.raw, latest.raw))}
	}
	if !releasePending(cwd, current, tags) {
		return nil
	}
	base, ok := baseTag(tags, current)
	if !ok {
		return nil
	}

	violations := checkBumpSize(cwd, source, base, current)
	return append(violations, checkChangelogEntry(cwd, base, current)...)
}

// declaredVersion busca la versión en los manifiestos habituales de la raíz.
func declaredVersion(cwd string) (string, string) {
	if v := packageJSONVersion(filepath.Join(cwd, "package.json")); v != "" {
		return "package.json", v
	}
	tomlSources := map[string]string{"pyproject.toml": `(?m)^\[(project|tool\.poetry)\]`, "Cargo.toml": `(?m)^\[package\]`}
	for _, name := range []string{"pyproject.toml", "Cargo.toml"} {
		if v := tomlVersion(filepath.Join(cwd, name), tomlSources[name]); v != "" {
			return name, v
		}
	}
	for _, name := range []string{"VERSION", "version.txt"} {
		data, err := os.ReadFile(filepath.Join(cwd, name))
		if err == nil && strings.TrimSpace(string(data)) != "" {
			return name, strings.TrimSpace(string(data))
		}
	}
	return "", ""
}

func packageJSONVersion(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var pkg struct {
		Version string `json:"version"`
	}
	if json.Unmarshal(data, &pkg) != nil {
		return ""
	}
	return pkg.Version
}

var tomlVersionPattern = regexp.MustCompile(`(?m)^version\s*=\s*"([^"]*)"`)

// tomlVersion toma el primer 'version = "..."' de la sección indicada.
func tomlVersion(path, sectionPattern string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	text := string(data)
	loc := regexp.MustCompile(sectionPattern).FindStringIndex(text)
	if loc == nil {
		return ""
	}
	section := text[loc[1]:]
	if next := regexp.MustCompile(`(?m)^\[`).FindStringIndex(section); next != nil {
		section = section[:next[0]]
	}
	m := tomlVersionPattern.FindStringSubmatch(section)
	if m == nil {
		return ""
	}
	return m[1]
}

func gitOutput(cwd string, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}

// mergedSemverTags: tags SemVer alcanzables desde HEAD (ignora ramas paralelas).
func mergedSemverTags(cwd string) []semver {
	var tags []semver
	for _, line := range strings.Split(gitOutput(cwd, "tag", "--merged", "HEAD"), "\n") {
		if v, ok := parseSemver(line); ok {
			tags = append(tags, v)
		}
	}
	return tags
}

func highestTag(tags []semver) semver {
	best := tags[0]
	for _, t := range tags[1:] {
		if cmpCore(t, best) > 0 {
			best = t
		}
	}
	return best
}

// baseTag: el tag estable más alto con núcleo menor que la versión declarada.
func baseTag(tags []semver, current semver) (semver, bool) {
	var base semver
	found := false
	for _, t := range tags {
		if t.isPre || cmpCore(t, current) >= 0 {
			continue
		}
		if !found || cmpCore(t, base) > 0 {
			base, found = t, true
		}
	}
	return base, found
}

// releasePending: la versión aún no tiene tag (se está preparando) o su tag
// apunta a HEAD (CI de publicación). Si el tag quedó atrás, es trabajo en curso.
func releasePending(cwd string, current semver, tags []semver) bool {
	for _, t := range tags {
		if t.raw != current.raw {
			continue
		}
		head := strings.TrimSpace(gitOutput(cwd, "rev-parse", "HEAD"))
		tagged := strings.TrimSpace(gitOutput(cwd, "rev-list", "-n", "1", "refs/tags/v"+current.raw))
		if tagged == "" {
			tagged = strings.TrimSpace(gitOutput(cwd, "rev-list", "-n", "1", "refs/tags/"+current.raw))
		}
		return head != "" && head == tagged
	}
	return true
}

func checkBumpSize(cwd, source string, base, current semver) []Violation {
	required, evidence := requiredBump(cwd, base)
	if base.core[0] == 0 && required == bumpMajor {
		// En 0.y.z un cambio incompatible sube MINOR (SemVer §4).
		required = bumpMinor
	}
	actual := actualBump(base, current)
	if actual >= required {
		return nil
	}
	return []Violation{versionViolation("VER-03-UNDERSIZED-BUMP", source,
		fmt.Sprintf("De v%s a %s es un salto %s, pero los cambios exigen %s (%s). Versión sugerida: %s.",
			base.raw, current.raw, bumpNames[actual], bumpNames[required], evidence, base.bumpTo(required)))}
}

// requiredBump lee los commits desde el tag base (Conventional Commits).
func requiredBump(cwd string, base semver) (int, string) {
	ref := "v" + base.raw
	if gitOutput(cwd, "rev-parse", "-q", "--verify", "refs/tags/"+ref) == "" {
		ref = base.raw
	}
	required, evidence := bumpNone, ""
	for _, msg := range strings.Split(gitOutput(cwd, "log", "--format=%B%x1e", ref+"..HEAD"), "\x1e") {
		kind := commitBump(strings.TrimSpace(msg))
		if kind > required {
			required, evidence = kind, firstLine(strings.TrimSpace(msg))
		}
	}
	return required, fmt.Sprintf("p. ej. '%s'", evidence)
}

func commitBump(msg string) int {
	m := conventionalPattern.FindStringSubmatch(msg)
	if m == nil {
		return bumpNone
	}
	if m[3] == "!" || breakingFooterPattern.MatchString(msg) {
		return bumpMajor
	}
	if m[1] == "feat" {
		return bumpMinor
	}
	return bumpPatch
}

func actualBump(base, current semver) int {
	if current.core[0] > base.core[0] {
		return bumpMajor
	}
	if current.core[1] > base.core[1] {
		return bumpMinor
	}
	if current.core[2] > base.core[2] {
		return bumpPatch
	}
	return bumpNone
}

// checkChangelogEntry solo aplica si el CHANGELOG ya documenta versiones
// (menciona la versión base): entonces debe documentar también la nueva.
func checkChangelogEntry(cwd string, base, current semver) []Violation {
	data, err := os.ReadFile(filepath.Join(cwd, "CHANGELOG.md"))
	if err != nil {
		return nil
	}
	text := string(data)
	if !strings.Contains(text, base.raw) || strings.Contains(text, current.raw) {
		return nil
	}
	return []Violation{versionViolation("VER-04-CHANGELOG-MISSING", "CHANGELOG.md",
		fmt.Sprintf("El CHANGELOG no tiene entrada para %s (la última documentada es %s).", current.raw, base.raw))}
}

func firstLine(s string) string {
	return strings.SplitN(s, "\n", 2)[0]
}

func versionViolation(rule, file, desc string) Violation {
	return Violation{Category: "VERSIONING", RuleID: rule, Description: desc, File: file}
}
