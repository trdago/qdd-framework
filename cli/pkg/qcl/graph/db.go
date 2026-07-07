package graph

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	_ "modernc.org/sqlite"
)

type GraphDB struct {
	db *sql.DB
}

func InitDB() (*GraphDB, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	qddDir := filepath.Join(cwd, ".qdd")
	if err := os.MkdirAll(qddDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .qdd dir: %w", err)
	}

	dbPath := filepath.Join(qddDir, "knowledge.db")
	db, err := sql.Open("sqlite", dbPath+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_txlock=immediate")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return &GraphDB{db: db}, nil
}

func migrate(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS nodes (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		name TEXT NOT NULL,
		content TEXT,
		metadata JSON
	);

	CREATE TABLE IF NOT EXISTS edges (
		source_id TEXT NOT NULL,
		target_id TEXT NOT NULL,
		relation TEXT NOT NULL,
		PRIMARY KEY (source_id, target_id, relation),
		FOREIGN KEY (source_id) REFERENCES nodes(id),
		FOREIGN KEY (target_id) REFERENCES nodes(id)
	);
	`

	_, err := db.Exec(schema)
	return err
}

func (g *GraphDB) GetDB() *sql.DB {
	return g.db
}

func (g *GraphDB) SyncToGraph(projectPath string) error {
	tx, err := g.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	qddDir := filepath.Join(projectPath, ".qdd")
	
	dirsToSync := map[string]string{
		"findings":      "finding",
		"certification": "rule",
		"sprints":       "task",
	}

	for dirName, nodeType := range dirsToSync {
		paths := []string{
			filepath.Join(qddDir, "project", dirName),
			filepath.Join(qddDir, "core", dirName),
		}

		for _, p := range paths {
			files, err := os.ReadDir(p)
			if err != nil {
				continue
			}

			for _, f := range files {
				if f.IsDir() || (!strings.HasSuffix(f.Name(), ".yaml") && !strings.HasSuffix(f.Name(), ".md")) {
					continue
				}

				filePath := filepath.Join(p, f.Name())
				data, err := os.ReadFile(filePath)
				if err != nil {
					continue
				}

				var raw map[string]interface{}
				yaml.Unmarshal(data, &raw)

				metadataBytes, _ := json.Marshal(raw)
				metadataStr := string(metadataBytes)

				id := fmt.Sprintf("%s:%s", nodeType, f.Name())
				name := f.Name()
				if title, ok := raw["title"].(string); ok {
					name = title
				}
				content := string(data)

				query := `
				INSERT INTO nodes (id, type, name, content, metadata) 
				VALUES (?, ?, ?, ?, ?)
				ON CONFLICT(id) DO UPDATE SET 
					type=excluded.type, 
					name=excluded.name, 
					content=excluded.content, 
					metadata=excluded.metadata;
				`
				_, err = tx.Exec(query, id, nodeType, name, content, metadataStr)
				if err != nil {
					fmt.Printf("Error upserting node %s: %v\n", id, err)
				}

				// Extract edges
				tx.Exec("DELETE FROM edges WHERE source_id = ?", id)
				
				if parent, ok := raw["parent"].(string); ok && parent != "" {
					targetID := fmt.Sprintf("rule:%s", parent)
					if !strings.HasSuffix(parent, ".yaml") {
						targetID = fmt.Sprintf("rule:%s.yaml", parent)
					}
					tx.Exec("INSERT OR IGNORE INTO edges (source_id, target_id, relation) VALUES (?, ?, ?)", id, targetID, "CHILD_OF")
				}

				if dependsOn, ok := raw["depends_on"]; ok {
					var deps []string
					if list, isList := dependsOn.([]interface{}); isList {
						for _, item := range list {
							deps = append(deps, fmt.Sprintf("%v", item))
						}
					}
					if str, isStr := dependsOn.(string); isStr {
						deps = append(deps, str)
					}

					for _, d := range deps {
						targetID := fmt.Sprintf("rule:%s", d)
						if !strings.HasSuffix(d, ".yaml") {
							targetID = fmt.Sprintf("rule:%s.yaml", d)
						}
						tx.Exec("INSERT OR IGNORE INTO edges (source_id, target_id, relation) VALUES (?, ?, ?)", id, targetID, "DEPENDS_ON")
					}
				}

				// Map features (funcionalidad)
				if feat, ok := raw["feature"].(string); ok && feat != "" {
					featID := fmt.Sprintf("feature:%s", feat)
					tx.Exec(query, featID, "feature", feat, "", "{}")
					
					var relation string
					switch nodeType {
					case "rule":
						relation = "IMPLEMENTS"
					case "finding":
						relation = "AFFECTS"
					case "task":
						relation = "WORKS_ON"
					default:
						relation = "RELATED_TO"
					}
					tx.Exec("INSERT OR IGNORE INTO edges (source_id, target_id, relation) VALUES (?, ?, ?)", id, featID, relation)
				}
				
				// Map bug/finding resolutions for tasks
				if resolves, ok := raw["resolves"].(string); ok && resolves != "" {
					findingID := fmt.Sprintf("finding:%s", resolves)
					if !strings.HasSuffix(resolves, ".yaml") {
						findingID = fmt.Sprintf("finding:%s.yaml", resolves)
					}
					tx.Exec("INSERT OR IGNORE INTO edges (source_id, target_id, relation) VALUES (?, ?, ?)", id, findingID, "RESOLVES")
				}
			}
		}
	}
	
	if err := g.syncCodebase(projectPath, tx); err != nil {
		fmt.Printf("Error al mapear código fuente: %v\n", err)
	}
	
	return tx.Commit()
}

func (g *GraphDB) Close() error {
	if g.db != nil {
		return g.db.Close()
	}
	return nil
}

func (g *GraphDB) syncCodebase(projectPath string, tx *sql.Tx) error {
	fset := token.NewFileSet()
	
	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		
		if info.IsDir() {
			if info.Name() == ".git" || info.Name() == "vendor" || info.Name() == "node_modules" || info.Name() == ".qdd" {
				return filepath.SkipDir
			}
			return nil
		}
		
		if !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}
		
		relPath, err := filepath.Rel(projectPath, path)
		if err != nil {
			relPath = path
		}
		
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return nil
		}
		
		fileID := fmt.Sprintf("file:%s", relPath)
		fileName := info.Name()
		
		isTest := strings.HasSuffix(fileName, "_test.go") || strings.HasSuffix(fileName, ".spec.ts")
		if isTest {
			fileID = fmt.Sprintf("test:%s", relPath)
		}
		
		query := `
		INSERT INTO nodes (id, type, name, content, metadata) 
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET 
			type=excluded.type, 
			name=excluded.name, 
			content=excluded.content, 
			metadata=excluded.metadata;
		`
		_, err = tx.Exec(query, fileID, "file", fileName, "", "{}")
		if isTest {
			_, err = tx.Exec(query, fileID, "test", fileName, "", "{}")
			
			// Try to link test to the file it tests
			baseFileName := strings.TrimSuffix(fileName, "_test.go") + ".go"
			baseFileID := fmt.Sprintf("file:%s", filepath.Join(filepath.Dir(relPath), baseFileName))
			tx.Exec("INSERT OR IGNORE INTO edges (source_id, target_id, relation) VALUES (?, ?, ?)", fileID, baseFileID, "TESTS")
		}

		if err != nil {
			fmt.Printf("Error insertando file %s: %v\n", fileID, err)
		}
		
		_, err = tx.Exec("DELETE FROM edges WHERE source_id = ?", fileID)
		if err != nil {
			fmt.Printf("Error borrando edges para %s: %v\n", fileID, err)
		}
		
		for _, imp := range f.Imports {
			pkgPath := strings.Trim(imp.Path.Value, "\"")
			pkgID := fmt.Sprintf("package:%s", pkgPath)
			
			_, err = tx.Exec(query, pkgID, "package", pkgPath, "", "{}")
			if err != nil {
				fmt.Printf("Error insertando pkg %s: %v\n", pkgID, err)
			}
			
			_, err = tx.Exec("INSERT OR IGNORE INTO edges (source_id, target_id, relation) VALUES (?, ?, ?)", fileID, pkgID, "IMPORTS")
			if err != nil {
				fmt.Printf("Error insertando edge %s -> %s: %v\n", fileID, pkgID, err)
			}
		}
		
		return nil
	})
	
	if err != nil {
		return err
	}
	
	// Sincronizar documentación
	docFolders := []string{"docs", "rfcs", "specification"}
	for _, folder := range docFolders {
		folderPath := filepath.Join(projectPath, folder)
		filepath.WalkDir(folderPath, func(path string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() && strings.HasSuffix(d.Name(), ".md") {
				relPath, _ := filepath.Rel(projectPath, path)
				docID := fmt.Sprintf("doc:%s", relPath)
				
				query := `
				INSERT INTO nodes (id, type, name, content, metadata) 
				VALUES (?, ?, ?, ?, ?)
				ON CONFLICT(id) DO UPDATE SET 
					name=excluded.name, 
					content=excluded.content;
				`
				tx.Exec(query, docID, "doc", d.Name(), "", "{}")
			}
			return nil
		})
	}
	
	return nil
}
