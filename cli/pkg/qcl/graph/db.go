package graph

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"go/ast"
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

	_, err := db.ExecContext(context.Background(), schema)
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
		syncGraphDir(qddDir, dirName, nodeType, tx)
	}

	if err := g.syncCodebase(projectPath, tx); err != nil {
		fmt.Printf("Error al mapear código fuente: %v\n", err)
	}

	return tx.Commit()
}

func syncGraphDir(qddDir, dirName, nodeType string, tx *sql.Tx) {
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
			processGraphFile(p, f, nodeType, tx)
		}
	}
}

func processGraphFile(p string, f os.DirEntry, nodeType string, tx *sql.Tx) {
	if !isValidGraphFile(f) {
		return
	}

	filePath := filepath.Join(p, f.Name())
	data, err := os.ReadFile(filePath)
	if err != nil {
		return
	}

	var raw map[string]interface{}
	yaml.Unmarshal(data, &raw)

	id, name := extractGraphNodeInfo(f.Name(), nodeType, raw)
	metadataBytes, _ := json.Marshal(raw)

	query := `
	INSERT INTO nodes (id, type, name, content, metadata) 
	VALUES (?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET 
		type=excluded.type, 
		name=excluded.name, 
		content=excluded.content, 
		metadata=excluded.metadata;
	`
	_, err = tx.ExecContext(context.Background(), query, id, nodeType, name, string(data), string(metadataBytes))
	if err != nil {
		fmt.Printf("Error upserting node %s: %v\n", id, err)
	}

	tx.ExecContext(context.Background(), "DELETE FROM edges WHERE source_id = ?", id)

	processGraphParent(raw, id, tx)
	processGraphDeps(raw, id, tx)
	processGraphFeature(raw, id, nodeType, tx, query)
	processGraphResolves(raw, id, tx)
}

func isValidGraphFile(f os.DirEntry) bool {
	if f.IsDir() {
		return false
	}
	if !strings.HasSuffix(f.Name(), ".yaml") && !strings.HasSuffix(f.Name(), ".md") {
		return false
	}
	return true
}

func extractGraphNodeInfo(fileName, nodeType string, raw map[string]interface{}) (string, string) {
	id := fmt.Sprintf("%s:%s", nodeType, fileName)
	name := fileName
	if title, ok := raw["title"].(string); ok {
		name = title
	}
	return id, name
}

func processGraphParent(raw map[string]interface{}, id string, tx *sql.Tx) {
	if parent, ok := raw["parent"].(string); ok && parent != "" {
		targetID := fmt.Sprintf("rule:%s", parent)
		if !strings.HasSuffix(parent, ".yaml") {
			targetID = fmt.Sprintf("rule:%s.yaml", parent)
		}
		tx.ExecContext(context.Background(), "INSERT OR IGNORE INTO edges (source_id, target_id, relation) VALUES (?, ?, ?)", id, targetID, "CHILD_OF")
	}
}

func processGraphDeps(raw map[string]interface{}, id string, tx *sql.Tx) {
	dependsOn, ok := raw["depends_on"]
	if !ok {
		return
	}

	var deps []string
	if list, isList := dependsOn.([]interface{}); isList {
		for _, item := range list {
			deps = append(deps, fmt.Sprintf("%v", item))
		}
	}
	if str, isStr := dependsOn.(string); isStr {
		deps = append(deps, str)
	}

	insertGraphDeps(id, deps, tx)
}

func insertGraphDeps(id string, deps []string, tx *sql.Tx) {
	for _, d := range deps {
		targetID := fmt.Sprintf("rule:%s", d)
		if !strings.HasSuffix(d, ".yaml") {
			targetID = fmt.Sprintf("rule:%s.yaml", d)
		}
		tx.ExecContext(context.Background(), "INSERT OR IGNORE INTO edges (source_id, target_id, relation) VALUES (?, ?, ?)", id, targetID, "DEPENDS_ON")
	}
}

func processGraphFeature(raw map[string]interface{}, id, nodeType string, tx *sql.Tx, query string) {
	feat, ok := raw["feature"].(string)
	if !ok || feat == "" {
		return
	}

	featID := fmt.Sprintf("feature:%s", feat)
	tx.ExecContext(context.Background(), query, featID, "feature", feat, "", "{}")

	relation := determineFeatureRelation(nodeType)
	tx.ExecContext(context.Background(), "INSERT OR IGNORE INTO edges (source_id, target_id, relation) VALUES (?, ?, ?)", id, featID, relation)
}

func determineFeatureRelation(nodeType string) string {
	switch nodeType {
	case "rule":
		return "IMPLEMENTS"
	case "finding":
		return "AFFECTS"
	case "task":
		return "WORKS_ON"
	}
	return "RELATED_TO"
}

func processGraphResolves(raw map[string]interface{}, id string, tx *sql.Tx) {
	if resolves, ok := raw["resolves"].(string); ok && resolves != "" {
		findingID := fmt.Sprintf("finding:%s", resolves)
		if !strings.HasSuffix(resolves, ".yaml") {
			findingID = fmt.Sprintf("finding:%s.yaml", resolves)
		}
		tx.ExecContext(context.Background(), "INSERT OR IGNORE INTO edges (source_id, target_id, relation) VALUES (?, ?, ?)", id, findingID, "RESOLVES")
	}
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
		return walkCodebaseFile(projectPath, path, info, err, tx, fset)
	})

	if err != nil {
		return err
	}

	syncDocsFolder(projectPath, tx)
	return nil
}

func walkCodebaseFile(projectPath, path string, info os.FileInfo, err error, tx *sql.Tx, fset *token.FileSet) error {
	if err != nil {
		return nil
	}

	if info.IsDir() {
		return handleCodebaseDir(info)
	}

	if !strings.HasSuffix(info.Name(), ".go") {
		return nil
	}

	return syncCodeFile(projectPath, path, info, tx, fset)
}

func handleCodebaseDir(info os.FileInfo) error {
	name := info.Name()
	if name == ".git" || name == "vendor" || name == "node_modules" || name == ".qdd" {
		return filepath.SkipDir
	}
	return nil
}

func syncCodeFile(projectPath, path string, info os.FileInfo, tx *sql.Tx, fset *token.FileSet) error {
	relPath, err := filepath.Rel(projectPath, path)
	if err != nil {
		relPath = path
	}

	f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		return nil
	}

	fileID, isTest := determineFileIDAndType(info.Name(), relPath)
	insertCodeFileNode(tx, fileID, info.Name(), isTest, relPath)

	tx.ExecContext(context.Background(), "DELETE FROM edges WHERE source_id = ?", fileID)

	insertCodeFileImports(tx, f, fileID)

	return nil
}

func determineFileIDAndType(fileName, relPath string) (string, bool) {
	isTest := strings.HasSuffix(fileName, "_test.go") || strings.HasSuffix(fileName, ".spec.ts")
	if isTest {
		return fmt.Sprintf("test:%s", relPath), true
	}
	return fmt.Sprintf("file:%s", relPath), false
}

func insertCodeFileNode(tx *sql.Tx, fileID, fileName string, isTest bool, relPath string) {
	query := `
	INSERT INTO nodes (id, type, name, content, metadata) 
	VALUES (?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET 
		type=excluded.type, 
		name=excluded.name, 
		content=excluded.content, 
		metadata=excluded.metadata;
	`
	tx.ExecContext(context.Background(), query, fileID, "file", fileName, "", "{}")
	if isTest {
		tx.ExecContext(context.Background(), query, fileID, "test", fileName, "", "{}")

		baseFileName := strings.TrimSuffix(fileName, "_test.go") + ".go"
		baseFileID := fmt.Sprintf("file:%s", filepath.Join(filepath.Dir(relPath), baseFileName))
		tx.ExecContext(context.Background(), "INSERT OR IGNORE INTO edges (source_id, target_id, relation) VALUES (?, ?, ?)", fileID, baseFileID, "TESTS")
	}
}

func insertCodeFileImports(tx *sql.Tx, f *ast.File, fileID string) {
	query := `
	INSERT INTO nodes (id, type, name, content, metadata) 
	VALUES (?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET 
		type=excluded.type, 
		name=excluded.name, 
		content=excluded.content, 
		metadata=excluded.metadata;
	`
	for _, imp := range f.Imports {
		pkgPath := strings.Trim(imp.Path.Value, "\"")
		pkgID := fmt.Sprintf("package:%s", pkgPath)

		tx.ExecContext(context.Background(), query, pkgID, "package", pkgPath, "", "{}")
		tx.ExecContext(context.Background(), "INSERT OR IGNORE INTO edges (source_id, target_id, relation) VALUES (?, ?, ?)", fileID, pkgID, "IMPORTS")
	}
}

func syncDocsFolder(projectPath string, tx *sql.Tx) {
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
				tx.ExecContext(context.Background(), query, docID, "doc", d.Name(), "", "{}")
			}
			return nil
		})
	}
}
