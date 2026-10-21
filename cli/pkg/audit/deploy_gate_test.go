package audit

import (
	"path/filepath"
	"testing"
)

func rulesFor(t *testing.T, files map[string]string) []Violation {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		writeFile(t, filepath.Join(dir, rel), content)
	}
	return RunDeployGateCheck(dir)
}

// Caso real (notificador): "Test & Deploy" que solo construye y sube a S3.
func TestDeployGate_GitHubDeployWithoutTests(t *testing.T) {
	v := rulesFor(t, map[string]string{".github/workflows/deploy.yml": `
name: Test & Deploy
on: push
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - run: npm ci && npm run build
      - run: aws s3 sync dist/ s3://bucket
`})
	if !hasRule(v, "CI-01-UNGATED-DEPLOY") {
		t.Fatalf("se esperaba CI-01, se obtuvo %v", v)
	}
}

// Jobs paralelos: tener pruebas en el archivo no basta si el deploy no las espera.
func TestDeployGate_TestsInParallelJobDoNotGate(t *testing.T) {
	v := rulesFor(t, map[string]string{".github/workflows/ci.yml": `
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: go test ./...
  deploy:
    runs-on: ubuntu-latest
    steps:
      - run: serverless deploy --stage prod
`})
	if !hasRule(v, "CI-01-UNGATED-DEPLOY") {
		t.Fatalf("un deploy sin needs: corre en paralelo a las pruebas; se esperaba CI-01, se obtuvo %v", v)
	}
}

func TestDeployGate_TransitiveNeedsGate(t *testing.T) {
	v := rulesFor(t, map[string]string{".github/workflows/publish.yml": `
on: push
jobs:
  quality:
    runs-on: ubuntu-latest
    steps:
      - run: make test
  build:
    needs: quality
    runs-on: ubuntu-latest
    steps:
      - run: go build
  publish:
    needs: [build]
    runs-on: ubuntu-latest
    steps:
      - run: npm publish
`})
	if len(v) != 0 {
		t.Fatalf("publish depende (vía build) de quality, que corre pruebas; no debía haber violaciones: %v", v)
	}
}

func TestDeployGate_SilencedTests(t *testing.T) {
	v := rulesFor(t, map[string]string{".github/workflows/bot.yml": `
on: push
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - run: go test ./... || true
      - run: gcloud run deploy bot
`})
	if !hasRule(v, "CI-02-DEPLOY-IGNORES-FAILURES") {
		t.Fatalf("se esperaba CI-02, se obtuvo %v", v)
	}
}

// Caso real (gdatos): Azure despliega con condition: succeededOrFailed().
func TestDeployGate_AzureSucceededOrFailed(t *testing.T) {
	v := rulesFor(t, map[string]string{"azure-pipelines.yml": `
trigger: [master]
steps:
- script: npm ci && npm run deploy
  condition: succeededOrFailed()
`})
	if !hasRule(v, "CI-01-UNGATED-DEPLOY") || !hasRule(v, "CI-02-DEPLOY-IGNORES-FAILURES") {
		t.Fatalf("se esperaban CI-01 y CI-02, se obtuvo %v", v)
	}
}

func TestDeployGate_NoDeployNoViolation(t *testing.T) {
	v := rulesFor(t, map[string]string{".github/workflows/ci.yml": `
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: go vet ./... && go build ./...
`})
	if len(v) != 0 {
		t.Fatalf("un workflow sin despliegue no debe generar violaciones: %v", v)
	}
}

func TestPolicies_DeployGateDefaultsOnForLegacyPolicyFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".qdd", "policies.yaml"), "owasp: true\nclean_code: true\n")
	if !LoadPolicies(dir).DeployGateEnabled() {
		t.Error("un policies.yaml sin la clave deploy_gate debe dejarla activa")
	}
	writeFile(t, filepath.Join(dir, ".qdd", "policies.yaml"), "deploy_gate: false\n")
	if LoadPolicies(dir).DeployGateEnabled() {
		t.Error("deploy_gate: false debe desactivarla explícitamente")
	}
}

// Casos reales que la primera versión no veía: serverless con versión fijada,
// despliegues por SSH y tareas de Azure.
func TestDeployGate_RecognizesRealDeployForms(t *testing.T) {
	cases := map[string]string{
		".github/workflows/api.yml": "on: push\njobs:\n  api:\n    runs-on: x\n    steps:\n      - run: npx serverless@3 deploy --stage prod\n",
		".github/workflows/bot.yml": "on: push\njobs:\n  bot:\n    runs-on: x\n    steps:\n      - uses: appleboy/ssh-action@v1.0.3\n",
		"azure-pipelines.yml":       "steps:\n- script: npm run build\n- task: S3Upload@1\n",
	}
	for file, content := range cases {
		if v := rulesFor(t, map[string]string{file: content}); !hasRule(v, "CI-01-UNGATED-DEPLOY") {
			t.Errorf("%s: se esperaba CI-01, se obtuvo %v", file, v)
		}
	}
}
