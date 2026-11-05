package audit

import (
	_ "embed"
	"os"
	"path/filepath"

	"github.com/qdd-framework/qdd/pkg/contexto"
	"gopkg.in/yaml.v3"
)

// Alcance de un principio (ver principios.yaml).
const (
	AlcanceUnidad      = "unidad"
	AlcanceRepositorio = "repositorio"
	AlcancePruebas     = "pruebas"
)

// Principio es lo que una regla protege y por qué (RFC 0005 §3.1).
type Principio struct {
	ID        string   `yaml:"id"`
	Nombre    string   `yaml:"nombre"`
	Categoria string   `yaml:"categoria"`
	Alcance   string   `yaml:"alcance"`
	Politicas []string `yaml:"politicas"`
	Reglas    []string `yaml:"reglas"`
	PorQue    string   `yaml:"por_que"`
}

//go:embed principios.yaml
var principiosYAML []byte

var catalogo = cargarPrincipios()

func cargarPrincipios() []Principio {
	var ps []Principio
	if err := yaml.Unmarshal(principiosYAML, &ps); err != nil {
		return nil
	}
	return ps
}

// Principios devuelve el catálogo de principios que verifica el motor.
func Principios() []Principio {
	return catalogo
}

var principioPorRegla = indexarReglas(catalogo)

func indexarReglas(ps []Principio) map[string]Principio {
	out := map[string]Principio{}
	for _, p := range ps {
		for _, r := range p.Reglas {
			out[r] = p
		}
	}
	return out
}

// PrincipioDeRegla devuelve el principio al que pertenece una regla del motor.
func PrincipioDeRegla(ruleID string) (Principio, bool) {
	p, ok := principioPorRegla[ruleID]
	return p, ok
}

// conPrincipio completa cada violación con su principio, su razón y su
// categoría, para que nadie tenga que adivinar por qué existe la regla.
func conPrincipio(violations []Violation) {
	for i := range violations {
		p, ok := PrincipioDeRegla(violations[i].RuleID)
		if !ok {
			continue
		}
		violations[i].Principio = p.ID
		violations[i].PorQue = p.PorQue
		violations[i].Categoria = p.Categoria
	}
}

// detector dice qué sabe revisar el código que verifica un principio. Lo
// declara el código, junto a los chequeos, porque es un hecho del código: el
// informe nunca promete revisar algo que el detector no lee.
type detector struct {
	// stacks que el detector lee (principios de alcance unidad).
	stacks []string
	// limite explica lo que el detector no revisa, aunque evalúe.
	limite string
	// evaluable dice si el detector puede revisar este repositorio
	// (principios de alcance repositorio). nil: siempre puede.
	evaluable func(cwd string) (bool, string)
}

var (
	soloGo         = []string{contexto.Go}
	lenguajesDeIA  = []string{contexto.Go, contexto.Node, contexto.Python, contexto.JVM}
	limiteDeIA     = "lee archivos .go, .js, .ts, .py y .java"
	limiteDeExiste = "solo revisa que exista, no su contenido"
)

var detectores = map[string]detector{
	"P-NO-INJECTION":          {stacks: soloGo},
	"P-NO-PLAIN-SECRETS":      {stacks: soloGo},
	"P-TESTS-PER-UNIT":        {},
	"P-E2E-EVIDENCE":          {limite: "solo revisa que exista e2e/, tests/e2e/, cypress/ o playwright/ en la raíz, no su contenido"},
	"P-READABLE-FLOW":         {stacks: soloGo},
	"P-NO-TEST-CODE-IN-PROD":  {stacks: soloGo},
	"P-VERSION-CONTROL":       {},
	"P-DEPLOY-GATED":          {limite: "revisa GitHub Actions, Azure Pipelines, GitLab CI y Bitbucket Pipelines", evaluable: pipelinesReconocidos},
	"P-MODULE-RECORDS":        {},
	"P-BLUEPRINTS-INTACT":     {},
	"P-DB-QUERY-COST":         {stacks: soloGo, limite: "lee archivos .go y .sql"},
	"P-DATA-UNIQUENESS":       {limite: "lee esquemas .sql y .ddl y modelos Go (*_schema.go, *_models.go)"},
	"P-TRACEABLE-COMMITS":     {limite: "revisa el último commit", evaluable: esRepositorioGit},
	"P-NO-PANIC":              {stacks: soloGo},
	"P-BOUNDED-CALLS":         {stacks: soloGo},
	"P-DECISIONS-RECORDED":    {limite: "solo revisa que exista docs/adr/, no su contenido"},
	"P-DOMAIN-ISOLATION":      {stacks: soloGo},
	"P-SMALL-FUNCTIONS":       {stacks: soloGo},
	"P-AI-POLICY":             {limite: limiteDeExiste},
	"P-AI-GOLDEN-SETS":        {limite: limiteDeExiste},
	"P-NO-PROMPT-INJECTION":   {stacks: lenguajesDeIA, limite: limiteDeIA},
	"P-AI-OUTPUT-VALIDATED":   {stacks: lenguajesDeIA, limite: limiteDeIA},
	"P-NO-SECRETS-IN-PROMPTS": {stacks: lenguajesDeIA, limite: limiteDeIA},
	"P-FINDINGS-COMPLETE":     {limite: "revisa que los campos existan, no corre la prueba que nombran"},
	"P-TASK-STATE-VALID":      {},
	"P-TOOLS-DEFINED":         {limite: "no corre las herramientas ni exige su ejecutable"},
	"P-SEMVER-RELEASES":       {},
}

// esRepositorioGit: sin git, el chequeo de commits no tiene qué revisar.
func esRepositorioGit(cwd string) (bool, string) {
	if pathExists(filepath.Join(cwd, ".git")) {
		return true, ""
	}
	return false, "no es un repositorio git"
}

// pipelinesReconocidos: un proyecto que despliega con un CI que QDD no lee
// queda no evaluado, en vez de aprobar porque no encontró pipelines.
func pipelinesReconocidos(cwd string) (bool, string) {
	for _, otro := range []string{"Jenkinsfile", ".circleci", ".travis.yml", ".drone.yml"} {
		if _, err := os.Stat(filepath.Join(cwd, otro)); err == nil && !hayPipelinesReconocidos(cwd) {
			return false, "usa " + otro + ", que el detector no lee"
		}
	}
	return true, ""
}

func hayPipelinesReconocidos(cwd string) bool {
	for _, patron := range []string{".github/workflows/*.yml", ".github/workflows/*.yaml", "*azure-pipelines*.yml", ".gitlab-ci.yml", "bitbucket-pipelines.yml"} {
		if m, _ := filepath.Glob(filepath.Join(cwd, patron)); len(m) > 0 {
			return true
		}
	}
	return false
}
