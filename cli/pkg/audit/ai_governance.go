package audit

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	secretInPromptRegex   = regexp.MustCompile(`(?i)(api_key|sk-[a-zA-Z0-9]{20,}|ghp_[a-zA-Z0-9]{20,}|password|secret)\s*[:=]\s*["'][^"']+["']`)
	promptInjectionRegex  = regexp.MustCompile(`(?i)(prompt|system_prompt|user_message)\s*\+=\s*.*\b(userInput|req\.Body|params\[|args\[)`)
	rawLLMExecutionRegex  = regexp.MustCompile(`(?i)(exec\.Command|os\.System|db\.Exec|db\.Query)\s*\(\s*.*(llmResponse|aiOutput|modelResult|completion)`)
)

// RunAIGovernanceCheck ejecuta la auditoría de los 4 estándares de IA:
// ISO/IEC 42001 (AIMS), ISO/IEC 29119-11 (AI Testing), SOC 2 (Processing Integrity) e ISO/IEC 23894 (AI Risk).
func RunAIGovernanceCheck(cwd string) []Violation {
	var violations []Violation

	aiDetected := detectAIComponents(cwd)
	if !aiDetected {
		return violations
	}

	checkISO42001Governance(cwd, &violations)
	checkISO29119Testing(cwd, &violations)
	checkSOC2ProcessingIntegrity(cwd, &violations)
	checkISO23894RiskMitigation(cwd, &violations)

	return violations
}

func detectAIComponents(cwd string) bool {
	hasAI := false
	aiKeywords := []string{"openai", "anthropic", "gemini", "langchain", "ollama", "llm", "claude", "gpt-4", "evaluator"}

	filepath.WalkDir(cwd, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			if d != nil && isIgnoredDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		if !hasCodeExtension(d.Name()) {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		contentLower := strings.ToLower(string(content))
		for _, kw := range aiKeywords {
			if strings.Contains(contentLower, kw) {
				hasAI = true
				return filepath.SkipAll
			}
		}
		return nil
	})

	return hasAI
}

func checkISO42001Governance(cwd string, violations *[]Violation) {
	configPath := filepath.Join(cwd, ".qdd", "config.yaml")
	if !fileExists(configPath) {
		*violations = append(*violations, Violation{
			Category:    "ISO/IEC 42001 (AIMS)",
			RuleID:      "ISO42001-GOV-01-MISSING-POLICY",
			Description: "Se detectaron componentes de IA pero falta el archivo de gobernanza de ciclo de vida (.qdd/config.yaml).",
		})
		return
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		return
	}

	if !strings.Contains(string(content), "governance") {
		*violations = append(*violations, Violation{
			Category:    "ISO/IEC 42001 (AIMS)",
			RuleID:      "ISO42001-GOV-01-MISSING-POLICY",
			Description: "La configuración de QDD (.qdd/config.yaml) no define la sección de gobernanza ('governance').",
			File:        ".qdd/config.yaml",
		})
	}
}

func checkISO29119Testing(cwd string, violations *[]Violation) {
	goldensetsDir := filepath.Join(cwd, ".qdd", "project", "goldensets")
	if !dirExists(goldensetsDir) {
		*violations = append(*violations, Violation{
			Category:    "ISO/IEC 29119-11 (AI Testing)",
			RuleID:      "ISO29119-TEST-01-NO-GOLDEN-SETS",
			Description: "Los componentes de IA requieren una suite de Golden Sets deterministas en .qdd/project/goldensets/ para validar precisión y ground truth.",
		})
		return
	}

	entries, err := os.ReadDir(goldensetsDir)
	if err != nil || len(entries) == 0 {
		*violations = append(*violations, Violation{
			Category:    "ISO/IEC 29119-11 (AI Testing)",
			RuleID:      "ISO29119-TEST-01-NO-GOLDEN-SETS",
			Description: "La carpeta .qdd/project/goldensets/ está vacía. Se requiere al menos un Golden Set de calibración para la IA.",
		})
	}
}

func checkSOC2ProcessingIntegrity(cwd string, violations *[]Violation) {
	filepath.WalkDir(cwd, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			if d != nil && isIgnoredDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		if !hasCodeExtension(d.Name()) {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		relPath, _ := filepath.Rel(cwd, path)
		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			if secretInPromptRegex.MatchString(line) && !strings.Contains(path, "_test") {
				*violations = append(*violations, Violation{
					Category:    "SOC 2 (AI Processing Integrity)",
					RuleID:      "SOC2-AI-03-LEAKED-SECRETS-IN-PROMPTS",
					Description: "Detección de posibles credenciales o secretos embebidos en el código fuente de IA.",
					File:        relPath,
					Line:        i + 1,
				})
			}

			if rawLLMExecutionRegex.MatchString(line) {
				*violations = append(*violations, Violation{
					Category:    "SOC 2 (AI Processing Integrity)",
					RuleID:      "SOC2-AI-02-UNVALIDATED-OUTPUT-EXECUTION",
					Description: "Ejecución directa de respuestas generativas de IA en comandos de sistema o BD sin validación de esquema previa.",
					File:        relPath,
					Line:        i + 1,
				})
			}
		}
		return nil
	})
}

func checkISO23894RiskMitigation(cwd string, violations *[]Violation) {
	filepath.WalkDir(cwd, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			if d != nil && isIgnoredDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		if !hasCodeExtension(d.Name()) {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		relPath, _ := filepath.Rel(cwd, path)
		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			if promptInjectionRegex.MatchString(line) {
				*violations = append(*violations, Violation{
					Category:    "ISO/IEC 23894 (AI Risk Management)",
					RuleID:      "ISO23894-RISK-01-UNSANITIZED-PROMPT-INJECTION",
					Description: "Concatenación directa de input de usuario en prompt sin delimitadores o sanitización contra Prompt Injection.",
					File:        relPath,
					Line:        i + 1,
				})
			}
		}
		return nil
	})
}

func hasCodeExtension(filename string) bool {
	ext := filepath.Ext(filename)
	valid := map[string]bool{
		".go": true, ".ts": true, ".js": true, ".py": true, ".java": true,
	}
	return valid[ext]
}

func isIgnoredDir(name string) bool {
	ignored := map[string]bool{
		".git": true, "node_modules": true, "vendor": true, "dist": true,
		".venv": true, "venv": true, "myenv": true, ".qdd": true,
	}
	return ignored[name]
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

