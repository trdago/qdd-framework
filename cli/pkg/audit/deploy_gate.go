package audit

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Reglas "no se despliega sin pasar las pruebas":
//   - CI-01-UNGATED-DEPLOY: un job/pipeline despliega y ni él ni los jobs de
//     los que depende (needs:) ejecutan pruebas.
//   - CI-02-DEPLOY-IGNORES-FAILURES: el pipeline despliega aunque fallen pasos
//     previos (succeededOrFailed) o silencia el resultado de las pruebas.

var deployPattern = regexp.MustCompile(`(?i)(serverless(@[\w.]+)? (deploy|remove)|\bsls deploy|npm run deploy|aws s3 sync|aws lambda update-function-code|cdk deploy|wrangler (pages )?deploy|wrangler-action|gcloud (run|app|functions) deploy|kubectl (apply|rollout)|helm (upgrade|install)|firebase deploy|vercel[^\n]*--prod|vercel-action|npm publish|docker push|docker-compose up|docker compose up|terraform apply|fly deploy|github-pages-deploy-action|amazon-ecs-deploy|appleboy/(ssh|scp)-action|\bscp |rsync [^\n]*@|task: (S3Upload|LambdaDeployFunction|AzureWebApp|AzureFunctionApp|AzureStaticWebApp|AzureRmWebAppDeployment|AzureFileCopy)@)`)

var testPattern = regexp.MustCompile(`(?i)(go test|(npm|yarn|pnpm)( run)? test|vitest|jest|pytest|flutter test|cargo test|mvn [^\n]*(test|verify)|gradle[^\n]* test|make test|qdd (gate|audit|certify)|qdd_gate)`)

var silencedTestPattern = regexp.MustCompile(`(?i)(go test|(npm|yarn|pnpm)( run)? test|vitest|jest|pytest|flutter test)[^\n]*\|\|\s*(true|echo|exit 0)`)

// RunDeployGateCheck revisa los pipelines de CI/CD del proyecto.
func RunDeployGateCheck(cwd string) []Violation {
	var violations []Violation
	for _, file := range ciFiles(cwd) {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		rel, _ := filepath.Rel(cwd, file)
		if strings.Contains(filepath.ToSlash(rel), ".github/workflows/") {
			violations = append(violations, checkGitHubWorkflow(rel, data)...)
			continue
		}
		violations = append(violations, checkGenericPipeline(rel, string(data))...)
	}
	return violations
}

func ciFiles(cwd string) []string {
	var files []string
	for _, pattern := range []string{".github/workflows/*.yml", ".github/workflows/*.yaml", "*azure-pipelines*.yml", "*azure-pipelines*.yaml", ".gitlab-ci.yml", "bitbucket-pipelines.yml"} {
		matches, _ := filepath.Glob(filepath.Join(cwd, pattern))
		files = append(files, matches...)
	}
	sort.Strings(files)
	return files
}

type ghWorkflow struct {
	Jobs map[string]ghJob `yaml:"jobs"`
}

type ghJob struct {
	Needs yaml.Node `yaml:"needs"`
	Uses  string    `yaml:"uses"`
	Steps []struct {
		Run  string `yaml:"run"`
		Uses string `yaml:"uses"`
		Name string `yaml:"name"`
	} `yaml:"steps"`
}

func (j ghJob) text() string {
	var b strings.Builder
	b.WriteString(j.Uses + "\n")
	for _, s := range j.Steps {
		b.WriteString(s.Name + "\n" + s.Uses + "\n" + s.Run + "\n")
	}
	return b.String()
}

func (j ghJob) needs() []string {
	if j.Needs.Kind == yaml.ScalarNode && j.Needs.Value != "" {
		return []string{j.Needs.Value}
	}
	var out []string
	for _, n := range j.Needs.Content {
		out = append(out, n.Value)
	}
	return out
}

func checkGitHubWorkflow(rel string, data []byte) []Violation {
	var wf ghWorkflow
	if yaml.Unmarshal(data, &wf) != nil || len(wf.Jobs) == 0 {
		return checkGenericPipeline(rel, string(data))
	}
	// Un workflow disparado por la finalización exitosa de otro (workflow_run +
	// conclusion == 'success') hereda el gate de ese otro workflow.
	if strings.Contains(string(data), "workflow_run") && strings.Contains(string(data), "'success'") {
		return nil
	}

	var violations []Violation
	names := make([]string, 0, len(wf.Jobs))
	for name := range wf.Jobs {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		job := wf.Jobs[name]
		text := job.text()
		if !deployPattern.MatchString(text) {
			continue
		}
		if silencedTestPattern.MatchString(text) {
			violations = append(violations, deployViolation("CI-02-DEPLOY-IGNORES-FAILURES", rel,
				fmt.Sprintf("El job '%s' despliega pero silencia el resultado de las pruebas (|| true / || echo).", name)))
		}
		if jobIsGated(name, wf.Jobs, map[string]bool{}) {
			continue
		}
		violations = append(violations, deployViolation("CI-01-UNGATED-DEPLOY", rel,
			fmt.Sprintf("El job '%s' despliega sin que él ni sus 'needs:' ejecuten pruebas. Agrega un job de pruebas (o 'qdd audit') y haz que el deploy dependa de él con 'needs:'.", name)))
	}
	return violations
}

// jobIsGated: el job corre pruebas, o depende (transitivamente) de uno que las corre.
func jobIsGated(name string, jobs map[string]ghJob, visited map[string]bool) bool {
	if visited[name] {
		return false
	}
	visited[name] = true
	job, ok := jobs[name]
	if !ok {
		return false
	}
	if testPattern.MatchString(job.text()) {
		return true
	}
	for _, dep := range job.needs() {
		if jobIsGated(dep, jobs, visited) {
			return true
		}
	}
	return false
}

// checkGenericPipeline cubre Azure Pipelines, GitLab y Bitbucket con una
// heurística por archivo: si despliega, debe ejecutar pruebas en el mismo
// pipeline y no debe forzar el deploy cuando fallan pasos previos.
func checkGenericPipeline(rel, text string) []Violation {
	if !deployPattern.MatchString(text) {
		return nil
	}
	var violations []Violation
	if !testPattern.MatchString(text) {
		violations = append(violations, deployViolation("CI-01-UNGATED-DEPLOY", rel,
			"El pipeline despliega sin ejecutar ninguna prueba. Agrega un paso de pruebas (o 'qdd audit') antes del despliegue."))
	}
	if strings.Contains(text, "succeededOrFailed()") || silencedTestPattern.MatchString(text) {
		violations = append(violations, deployViolation("CI-02-DEPLOY-IGNORES-FAILURES", rel,
			"El pipeline despliega aunque fallen pasos previos (succeededOrFailed / || true). Un fallo de pruebas debe detener el despliegue."))
	}
	return violations
}

func deployViolation(rule, file, desc string) Violation {
	return Violation{Category: "CI-CD", RuleID: rule, Description: desc, File: file}
}
