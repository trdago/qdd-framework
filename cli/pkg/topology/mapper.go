package topology

import (
	"os"
	"path/filepath"
	"strings"
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
				
				moduleNode := &TopologyNode{
					ID:            "mod-" + d.Name(),
					Name:          d.Name(),
					Type:          "Module",
					Path:          relPath,
					Certified:     false,
					RequiredCerts: []string{"CERT-005-CLEAN-CODE"},
					MissingCerts:  []string{},
					Children:      []*TopologyNode{},
				}

				// Verificar si el archivo tiene anotación de certificación o si está limpio (sin 'else', por ejemplo)
				hasElse := strings.Contains(code, " else ") || strings.Contains(code, "}else{") || strings.Contains(code, "} else {")
				hasCertAnnotation := strings.Contains(code, "@qdd:certify") || strings.Contains(code, "@certified")

				if !hasElse || hasCertAnnotation {
					moduleNode.Certified = true
				} else {
					moduleNode.MissingCerts = append(moduleNode.MissingCerts, "CERT-005-CLEAN-CODE")
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

	if totalNodes > 0 {
		topology.GlobalScore = (certifiedNodes * 100) / totalNodes
	} else {
		topology.GlobalScore = 100
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
