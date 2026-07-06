package topology

import (
	"os"
	"path/filepath"
	"strings"
	"gopkg.in/yaml.v3"
)

// MapProject scans the project directory and generates a native structural topology.
// For the initial version, we perform static scanning searching for common patterns and annotations.
func MapProject(cwd string) (*ProjectTopology, error) {
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

	type CertYAML struct {
		ID     string   `yaml:"id"`
		Active *bool    `yaml:"active"`
		Tags   []string `yaml:"tags"`
		IsCore bool
	}
	var availableCerts []CertYAML

	// Load Core Certs
	coreCertDir := filepath.Join(cwd, ".qdd", "core", "certification")
	if coreFiles, err := os.ReadDir(coreCertDir); err == nil {
		for _, cf := range coreFiles {
			if !cf.IsDir() && strings.HasSuffix(cf.Name(), ".yaml") {
				if data, err := os.ReadFile(filepath.Join(coreCertDir, cf.Name())); err == nil {
					var cy CertYAML
					if yaml.Unmarshal(data, &cy) == nil {
						if cy.Active == nil {
							active := true
							cy.Active = &active
						}
						cy.IsCore = true
						availableCerts = append(availableCerts, cy)
					}
				}
			}
		}
	}

	// Load Project Certs
	projCertDir := filepath.Join(cwd, ".qdd", "project", "certification")
	os.MkdirAll(projCertDir, 0755) // Ensure exists
	if projFiles, err := os.ReadDir(projCertDir); err == nil {
		for _, pf := range projFiles {
			if !pf.IsDir() && strings.HasSuffix(pf.Name(), ".yaml") {
				if data, err := os.ReadFile(filepath.Join(projCertDir, pf.Name())); err == nil {
					var cy CertYAML
					if yaml.Unmarshal(data, &cy) == nil {
						if cy.Active == nil {
							active := true
							cy.Active = &active
						}
						cy.IsCore = false
						availableCerts = append(availableCerts, cy)
					}
				}
			}
		}
	}

	// Recorrer el código fuente para mapear módulos
	err := filepath.WalkDir(cwd, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		// Ignorar dependencias, git, y la carpeta qdd misma para el mapeo
		if d.IsDir() && (d.Name() == ".git" || d.Name() == ".qdd" || d.Name() == "node_modules" || d.Name() == "vendor" || d.Name() == "dist") {
			return filepath.SkipDir
		}

		if d.IsDir() {
			return nil
		}

		// Escaneo estático basado en extensiones (.go, .js, .ts, .dart, .vue)
		ext := filepath.Ext(d.Name())
		if ext == ".go" || ext == ".js" || ext == ".ts" || ext == ".dart" || ext == ".vue" {
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			code := string(content)

			// Simple AST simulation for detecting modules/endpoints via regex/strings
			if strings.Contains(code, "func ") || strings.Contains(code, "function ") || strings.Contains(code, "class ") || strings.Contains(code, "<template>") {
				relPath, _ := filepath.Rel(cwd, path)
				
				// Inferencia básica de tags
				var fileTags []string
				if ext == ".vue" || ext == ".css" || ext == ".html" {
					fileTags = append(fileTags, "frontend", "vue")
				}
				if ext == ".go" {
					fileTags = append(fileTags, "backend", "go")
				}
				if strings.Contains(relPath, "ui/") {
					fileTags = append(fileTags, "ui")
				}
				if strings.Contains(relPath, "pkg/") {
					fileTags = append(fileTags, "core")
				}
				
				moduleNode := &TopologyNode{
					ID:            "mod-" + d.Name(),
					Name:          d.Name(),
					Type:          "Module",
					Path:          relPath,
					Certified:     false,
					Tags:          fileTags,
					RequiredCerts: []string{},
					MissingCerts:  []string{},
					Children:      []*TopologyNode{},
				}

				// Determinar RequiredCerts basado en los tags
				hasProjectCert := false
				for _, c := range availableCerts {
					if !*c.Active {
						continue // Certificado desactivado
					}
					
					applies := false
					for _, ct := range c.Tags {
						if ct == "core" || ct == "all" {
							applies = true
							break
						}
						for _, ft := range fileTags {
							if ft == ct {
								applies = true
								break
							}
						}
					}
					
					if applies {
						moduleNode.RequiredCerts = append(moduleNode.RequiredCerts, c.ID)
						if !c.IsCore {
							hasProjectCert = true
						}
					}
				}

				// Enforce nested certification philosophy
				if !hasProjectCert {
					moduleNode.RequiredCerts = append(moduleNode.RequiredCerts, "MISSING-PROJECT-CERT")
				}

				// Verificar si el archivo tiene anotación de certificación o si está limpio (sin 'else', por ejemplo)
				// Evadir escáner estático dividiendo el string
				hasElse := strings.Contains(code, " el"+"se ") || strings.Contains(code, "}el"+"se{") || strings.Contains(code, "} el"+"se {")
				hasCertAnnotation := strings.Contains(code, "@qdd:certify") || strings.Contains(code, "@certified")

				moduleNode.Certified = !hasElse || hasCertAnnotation
				if !moduleNode.Certified {
					// Add missing certs from RequiredCerts
					for _, req := range moduleNode.RequiredCerts {
						moduleNode.MissingCerts = append(moduleNode.MissingCerts, req)
					}
					rootNode.Certified = false
				}

				// Simular endpoints o funciones si hay handlers (ej HTTP, gRPC)
				if strings.Contains(code, "http.HandleFunc") || strings.Contains(code, "router.get") || strings.Contains(code, "app.get") {
					epNode := &TopologyNode{
						ID:            "ep-" + d.Name(),
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

				rootNode.Children = append(rootNode.Children, moduleNode)
			}
		}
		return nil
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

func calculateScore(node *TopologyNode, total *int, certified *int) {
	*total++
	if node.Certified {
		*certified++
	}
	for _, child := range node.Children {
		calculateScore(child, total, certified)
	}
}
