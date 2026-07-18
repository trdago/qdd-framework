// Package bugreport implements `qdd bug`: turning a captured crash/error into
// a permanent Finding, exactly as the framework's "Aprendizaje Perpetuo"
// principle requires — every detected problem becomes a Finding, evidence,
// and (eventually) a regression test, never just a transient log line.
//
// It deliberately does NOT try to write or apply a fix, nor synthesize a
// fake regression test: QDD's cognitive engine is deprecated in favor of an
// external AI (via MCP) doing the actual repair. Reporting a real, reproducible
// test for an arbitrary crashed process would require understanding that
// process's language/framework, which this package has no way to know.
package bugreport

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Finding is the subset of the real .qdd/project/findings/*.yaml schema this
// package writes. Other fields (parent, resolves) are intentionally left
// empty for a human/AI to fill in once the finding is triaged.
type Finding struct {
	ID          string         `yaml:"id"`
	Type        string         `yaml:"type"`
	Title       string         `yaml:"title"`
	Description string         `yaml:"description"`
	Impact      string         `yaml:"impact"`
	Status      string         `yaml:"status"`
	Feature     string         `yaml:"feature"`
	Author      string         `yaml:"author"`
	CreatedAt   string         `yaml:"created_at"`
	Parent      string         `yaml:"parent"`
	Resolves    string         `yaml:"resolves"`
	Metadata    map[string]any `yaml:"metadata"`
}

// Report describes the crash/problem being filed. Command+Args identify what
// was running; ExitCode and Output are the observed failure evidence.
type Report struct {
	Title    string
	Command  string
	Args     []string
	ExitCode int
	Output   string
}

// Filed is what File() produces: the created finding plus where its evidence
// was written, so the caller can point the user at both.
type Filed struct {
	Finding      Finding
	FindingPath  string
	EvidencePath string
}

// File writes a new OPEN Finding plus its evidence file under cwd's .qdd/
// tree, and returns what it created. It never marks anything RESOLVED and
// never touches the target process's source — that part is left to a human
// or an AI acting through the framework's normal Modo Consultivo flow.
func File(cwd string, r Report) (*Filed, error) {
	findingsDir := filepath.Join(cwd, ".qdd", "project", "findings")
	id, err := nextFindingID(findingsDir)
	if err != nil {
		return nil, err
	}

	evidencePath, err := writeEvidence(cwd, id, r)
	if err != nil {
		return nil, err
	}

	finding := buildFinding(id, r, evidencePath)
	findingPath, err := writeFinding(findingsDir, finding)
	if err != nil {
		return nil, err
	}

	return &Filed{Finding: finding, FindingPath: findingPath, EvidencePath: evidencePath}, nil
}

// MarkResolved updates an existing finding file after an auto-repair attempt
// confirms the fix. It never deletes the original crash evidence — only
// status and a summary of what changed are added, keeping the finding itself
// as the permanent record of what happened.
func MarkResolved(findingPath, summary string, testAdded bool) error {
	data, err := os.ReadFile(findingPath)
	if err != nil {
		return err
	}
	var f Finding
	if err := yaml.Unmarshal(data, &f); err != nil {
		return err
	}

	f.Status = "RESOLVED"
	if f.Metadata == nil {
		f.Metadata = map[string]any{}
	}
	f.Metadata["auto_repair_summary"] = summary
	f.Metadata["test_pending"] = !testAdded

	out, err := yaml.Marshal(f)
	if err != nil {
		return err
	}
	return os.WriteFile(findingPath, out, 0644)
}

func buildFinding(id string, r Report, evidencePath string) Finding {
	fullCommand := strings.TrimSpace(r.Command + " " + strings.Join(r.Args, " "))
	return Finding{
		ID:    id,
		Type:  "bug",
		Title: fmt.Sprintf("%s (detectado por qdd run --keep-alive)", displayTitle(r, fullCommand)),
		Description: fmt.Sprintf(
			"El proceso supervisado '%s' terminó con código de salida %d. Evidencia completa en %s.",
			fullCommand, r.ExitCode, evidencePath),
		Impact:    "HIGH - El proceso supervisado dejó de funcionar y la supervisión se detuvo hasta que esto se resuelva.",
		Status:    "OPEN",
		Feature:   "qdd run --keep-alive",
		Author:    "QDD Supervisor",
		CreatedAt: time.Now().Format(time.RFC3339),
		Metadata: map[string]any{
			"command":       fullCommand,
			"exit_code":     r.ExitCode,
			"evidence_path": evidencePath,
			"test_pending":  true,
			"test_note":     "Este finding no trae un test de regresión automático: QDD no puede sintetizar uno válido sin conocer el lenguaje/framework del proceso supervisado. Quien lo resuelva (humano o IA vía MCP) debe agregar un test que reproduzca el fallo antes de marcarlo RESOLVED (principio Auto-TDD ante Bugs).",
		},
	}
}

func displayTitle(r Report, fullCommand string) string {
	if r.Title != "" {
		return r.Title
	}
	return "Proceso '" + fullCommand + "' falló"
}

func writeFinding(findingsDir string, f Finding) (string, error) {
	if err := os.MkdirAll(findingsDir, 0755); err != nil {
		return "", err
	}
	data, err := yaml.Marshal(f)
	if err != nil {
		return "", err
	}
	path := filepath.Join(findingsDir, f.ID+".yaml")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", err
	}
	return path, nil
}

func writeEvidence(cwd, id string, r Report) (string, error) {
	evidenceDir := filepath.Join(cwd, ".qdd", "project", "evidence", "bugs")
	if err := os.MkdirAll(evidenceDir, 0755); err != nil {
		return "", err
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	path := filepath.Join(evidenceDir, fmt.Sprintf("%s_%s.log", id, timestamp))

	content := fmt.Sprintf("Comando: %s %s\nExit code: %d\nFecha: %s\n\n--- Output capturado (tail) ---\n%s\n",
		r.Command, strings.Join(r.Args, " "), r.ExitCode, time.Now().Format(time.RFC3339), r.Output)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", err
	}
	return path, nil
}

var findingIDPattern = regexp.MustCompile(`^FND-(\d+)`)

// nextFindingID scans dir for existing FND-NNN[-slug].yaml files and returns
// the next sequential ID, so bug reports slot into the same numbering the
// rest of the project's findings already use.
func nextFindingID(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return "FND-001", nil
	}
	if err != nil {
		return "", err
	}

	next := highestFindingNumber(entries) + 1
	return fmt.Sprintf("FND-%03d", next), nil
}

func highestFindingNumber(entries []os.DirEntry) int {
	highest := 0
	for _, entry := range entries {
		if n, ok := findingNumber(entry.Name()); ok && n > highest {
			highest = n
		}
	}
	return highest
}

func findingNumber(filename string) (int, bool) {
	match := findingIDPattern.FindStringSubmatch(filename)
	if match == nil {
		return 0, false
	}
	n, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, false
	}
	return n, true
}
