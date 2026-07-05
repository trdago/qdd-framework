package graph

import (
	"database/sql"
	"encoding/json"
	"fmt"
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
	db, err := sql.Open("sqlite", dbPath)
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
				_, err = g.db.Exec(query, id, nodeType, name, content, metadataStr)
				if err != nil {
					fmt.Printf("Error upserting node %s: %v\n", id, err)
				}

				// Extract edges
				g.db.Exec("DELETE FROM edges WHERE source_id = ?", id)
				
				if parent, ok := raw["parent"].(string); ok && parent != "" {
					targetID := fmt.Sprintf("rule:%s", parent)
					if !strings.HasSuffix(parent, ".yaml") {
						targetID = fmt.Sprintf("rule:%s.yaml", parent)
					}
					g.db.Exec("INSERT OR IGNORE INTO edges (source_id, target_id, relation) VALUES (?, ?, ?)", id, targetID, "CHILD_OF")
				}

				if dependsOn, ok := raw["depends_on"]; ok {
					var deps []string
					if list, isList := dependsOn.([]interface{}); isList {
						for _, item := range list {
							deps = append(deps, fmt.Sprintf("%v", item))
						}
					} else if str, isStr := dependsOn.(string); isStr {
						deps = append(deps, str)
					}

					for _, d := range deps {
						targetID := fmt.Sprintf("rule:%s", d)
						if !strings.HasSuffix(d, ".yaml") {
							targetID = fmt.Sprintf("rule:%s.yaml", d)
						}
						g.db.Exec("INSERT OR IGNORE INTO edges (source_id, target_id, relation) VALUES (?, ?, ?)", id, targetID, "DEPENDS_ON")
					}
				}
			}
		}
	}
	return nil
}

func (g *GraphDB) Close() error {
	if g.db != nil {
		return g.db.Close()
	}
	return nil
}
