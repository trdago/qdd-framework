package topology

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"go/ast"
	"go/parser"
	"go/token"

	"gopkg.in/yaml.v3"
)

// MapProject scans the project directory and generates a native structural topology.
// For the initial version, we perform static scanning searching for common patterns and annotations.
func MapProject(cwd string) (*ProjectTopology, error) {
	if _, err := os.Stat(cwd); err != nil {
		return nil, err
	}

	rootNode := &TopologyNode{
		ID:            "app-root",
		Name:          filepath.Base(cwd),
		Type:          "App",
		Path:          cwd,
		Certified:     true,
		RequiredCerts: []string{"OWASP", "Clean Code"},
		Children:      []*TopologyNode{},
	}

	topology := &ProjectTopology{
		Application: rootNode,
	}

	var availableCerts []CertYAML
	loadAvailableCerts(cwd, &availableCerts)

	tagsCfg := loadDynamicTagsConfig(cwd)

	err := filepath.WalkDir(cwd, func(path string, d os.DirEntry, err error) error {
		return processMapPath(cwd, path, d, err, rootNode, availableCerts, tagsCfg)
	})

	if err != nil {
		return nil, err
	}

	// Calcular Global Score basado en nodos
	totalNodes := 0
	certifiedNodes := 0
	calculateScore(rootNode, &totalNodes, &certifiedNodes)

	topology.GlobalScore = 100
	if totalNodes > 0 {
		topology.GlobalScore = (certifiedNodes * 100) / totalNodes
	}

	return topology, nil
}

type TagsConfig struct {
	Extensions map[string][]string `yaml:"extensions"`
	Paths      map[string][]string `yaml:"paths"`
}

func loadDynamicTagsConfig(cwd string) TagsConfig {
	path := filepath.Join(cwd, ".qdd", "tags.yaml")
	var cfg TagsConfig
	if data, err := os.ReadFile(path); err == nil {
		yaml.Unmarshal(data, &cfg)
	}
	
	if len(cfg.Extensions) == 0 && len(cfg.Paths) == 0 {
		cfg.Extensions = map[string][]string{
			".vue": {"frontend", "vue"},
			".css": {"frontend"},
			".html": {"frontend"},
			".go": {"backend", "go"},
		}
		cfg.Paths = map[string][]string{
			"ui/": {"ui"},
			"pkg/": {"core"},
		}
	}
	return cfg
}

type CertYAML struct {
	ID     string   `yaml:"id"`
	Active *bool    `yaml:"active"`
	Tags   []string `yaml:"tags"`
	IsCore bool
}

func loadAvailableCerts(cwd string, availableCerts *[]CertYAML) {
	coreCertDir := filepath.Join(cwd, ".qdd", "core", "certification")
	loadCertsFromDir(coreCertDir, true, availableCerts)

	projCertDir := filepath.Join(cwd, ".qdd", "project", "certification")
	os.MkdirAll(projCertDir, 0755) // Ensure exists
	loadCertsFromDir(projCertDir, false, availableCerts)
}

func loadCertsFromDir(dir string, isCore bool, availableCerts *[]CertYAML) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, f := range files {
		loadSingleCert(dir, f, isCore, availableCerts)
	}
}

func loadSingleCert(dir string, f os.DirEntry, isCore bool, availableCerts *[]CertYAML) {
	if !isValidCertFile(f) {
		return
	}
	
	data, err := os.ReadFile(filepath.Join(dir, f.Name()))
	if err != nil {
		return
	}
	
	parseAndAppendCert(data, isCore, availableCerts)
}

func isValidCertFile(f os.DirEntry) bool {
	return !f.IsDir() && strings.HasSuffix(f.Name(), ".yaml")
}

func parseAndAppendCert(data []byte, isCore bool, availableCerts *[]CertYAML) {
	var cy CertYAML
	if yaml.Unmarshal(data, &cy) != nil {
		return
	}
	if cy.Active == nil {
		active := true
		cy.Active = &active
	}
	cy.IsCore = isCore
	*availableCerts = append(*availableCerts, cy)
}

func processMapPath(cwd string, path string, d os.DirEntry, err error, rootNode *TopologyNode, availableCerts []CertYAML, tagsCfg TagsConfig) error {
	if err != nil {
		return handleMapPathError(cwd, path, err)
	}

	if d.IsDir() {
		return handleMapDir(d)
	}

	processMapFile(cwd, path, d, rootNode, availableCerts, tagsCfg)
	return nil
}

func handleMapPathError(cwd, path string, err error) error {
	if path == cwd {
		return err
	}
	return filepath.SkipDir
}

func handleMapDir(d os.DirEntry) error {
	if isIgnoredMapDir(d.Name()) {
		return filepath.SkipDir
	}
	return nil
}

func isIgnoredMapDir(name string) bool {
	return name == ".git" || name == ".qdd" || name == "node_modules" || name == "vendor" || name == "dist"
}

func processMapFile(cwd, path string, d os.DirEntry, rootNode *TopologyNode, availableCerts []CertYAML, tagsCfg TagsConfig) {
	if !isValidModuleExt(filepath.Ext(d.Name())) {
		return
	}

	code, ok := readModuleCode(path)
	if !ok {
		return
	}

	relPath, _ := filepath.Rel(cwd, path)
	fileTags := determineFileTags(filepath.Ext(d.Name()), relPath, tagsCfg)
	
	moduleNode := createModuleNode(d.Name(), relPath, fileTags)
	applyRequiredCerts(moduleNode, availableCerts, fileTags)
	evaluateCertification(moduleNode, code, rootNode, filepath.Ext(d.Name()))
	addEndpoints(moduleNode, code, relPath)

	rootNode.Children = append(rootNode.Children, moduleNode)
}

func isValidModuleExt(ext string) bool {
	return ext == ".go" || ext == ".js" || ext == ".ts" || ext == ".dart" || ext == ".vue"
}

func readModuleCode(path string) (string, bool) {
	content, readErr := os.ReadFile(path)
	if readErr != nil {
		return "", false
	}
	code := string(content)

	if !isModuleCode(code) {
		return "", false
	}
	return code, true
}

func isModuleCode(code string) bool {
	return strings.Contains(code, "func ") || strings.Contains(code, "function ") || strings.Contains(code, "class ") || strings.Contains(code, "<template>")
}

func determineFileTags(ext, relPath string, tagsCfg TagsConfig) []string {
	var fileTags []string
	
	if tags, ok := tagsCfg.Extensions[ext]; ok {
		fileTags = append(fileTags, tags...)
	}

	for p, tags := range tagsCfg.Paths {
		matched, _ := filepath.Match(p, relPath)
		if matched {
			fileTags = append(fileTags, tags...)
			continue
		}
		
		// Fallback estricto para directorios si no usaron glob (ej: "ui/")
		cleanP := filepath.Clean(p) + string(os.PathSeparator)
		if strings.HasPrefix(relPath, cleanP) {
			fileTags = append(fileTags, tags...)
		}
	}
	
	return fileTags
}

func createModuleNode(name, relPath string, tags []string) *TopologyNode {
	return &TopologyNode{
		ID:            "mod-" + name,
		Name:          name,
		Type:          "Module",
		Path:          relPath,
		Certified:     false,
		Tags:          tags,
		RequiredCerts: []string{},
		MissingCerts:  []string{},
		Children:      []*TopologyNode{},
	}
}

func applyRequiredCerts(moduleNode *TopologyNode, availableCerts []CertYAML, fileTags []string) {
	hasProjectCert := false
	for _, c := range availableCerts {
		if processSingleCertForModule(moduleNode, c, fileTags) {
			hasProjectCert = true
		}
	}

	if !hasProjectCert {
		moduleNode.RequiredCerts = append(moduleNode.RequiredCerts, "MISSING-PROJECT-CERT")
	}
}

func processSingleCertForModule(moduleNode *TopologyNode, c CertYAML, fileTags []string) bool {
	if !*c.Active {
		return false
	}
	
	if certApplies(c, fileTags) {
		moduleNode.RequiredCerts = append(moduleNode.RequiredCerts, c.ID)
		return !c.IsCore
	}
	return false
}

func certApplies(c CertYAML, fileTags []string) bool {
	for _, ct := range c.Tags {
		if checkCertTagMatch(ct, fileTags) {
			return true
		}
	}
	return false
}

func checkCertTagMatch(ct string, fileTags []string) bool {
	if ct == "core" || ct == "all" {
		return true
	}
	for _, ft := range fileTags {
		if ft == ct {
			return true
		}
	}
	return false
}

func evaluateCertification(moduleNode *TopologyNode, code string, rootNode *TopologyNode, ext string) {
	moduleNode.Certified = checkCertificationStatus(code, ext)
	if !moduleNode.Certified {
		markModuleUncertified(moduleNode, rootNode)
	}
}

func checkCertificationStatus(code string, ext string) bool {
	if ext == ".go" {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", code, 0)
		if err == nil {
			hasElse := false
			ast.Inspect(f, func(n ast.Node) bool {
				if ifStmt, ok := n.(*ast.IfStmt); ok {
					if ifStmt.Else != nil {
						hasElse = true
						return false
					}
				}
				return true
			})
			return !hasElse
		}
	}
	
	// Fallback para JS/TS/Vue/Dart etc usando Regex
	cleanCode := stripCommentsAndStrings(code)
	matched, _ := regexp.MatchString(`(?m)(?:^|[\s{}])else(?:[\s{]|$)`, cleanCode)
	return !matched
}

func stripCommentsAndStrings(code string) string {
	reStr := regexp.MustCompile(`"(.*?)"|'(.*?)'`)
	code = reStr.ReplaceAllString(code, "")
	reTpl := regexp.MustCompile("(?s)`(.*?)`")
	code = reTpl.ReplaceAllString(code, "")
	reMulti := regexp.MustCompile(`(?s)/\*.*?\*/`)
	code = reMulti.ReplaceAllString(code, "")
	reSingle := regexp.MustCompile(`//.*`)
	code = reSingle.ReplaceAllString(code, "")
	reHTML := regexp.MustCompile(`(?s)<!--.*?-->`)
	return reHTML.ReplaceAllString(code, "")
}

func markModuleUncertified(moduleNode *TopologyNode, rootNode *TopologyNode) {
	for _, req := range moduleNode.RequiredCerts {
		moduleNode.MissingCerts = append(moduleNode.MissingCerts, req)
	}
	rootNode.Certified = false
}

func addEndpoints(moduleNode *TopologyNode, code, relPath string) {
	if strings.Contains(code, "http.HandleFunc") || strings.Contains(code, "router.get") || strings.Contains(code, "app.get") {
		epNode := &TopologyNode{
			ID:            "ep-" + moduleNode.Name,
			Name:          "HTTP Endpoint Handler",
			Type:          "Endpoint",
			Path:          relPath,
			Certified:     moduleNode.Certified,
		}
		if !moduleNode.Certified {
			epNode.MissingCerts = append(epNode.MissingCerts, "OWASP-SECURITY")
		}
		moduleNode.Children = append(moduleNode.Children, epNode)
	}
}

func calculateScore(node *TopologyNode, total *int, certified *int) {
	*total++
	if node.Certified {
		*certified++
	}
	for _, child := range node.Children {
		calculateScore(child, total, certified)
	}
}
