package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/qdd-framework/qdd/pkg/gitenv"
)

// Rule: Prohibido usar 'else'

// creadoPorInit: lo que 'qdd init' deja en un proyecto nuevo. Cualquier otro archivo de .qdd es conocimiento del
// proyecto, y deshacer init no lo borra (FND-306).
var creadoPorInit = []string{
	".qdd/config.yaml", ".qdd/policies.yaml", ".qdd/state.json", ".qdd/core/", ".qdd/working/", ".qdd/cache/",
	".qdd/dashboard/", ".qdd/project/evidence/doctor/", ".qdd/project/metrics/cognitive_history.json",
	".qdd/project/metrics/mcp_eventos.json",
}

func imprimirUndoInit() {
	cwd, _ := os.Getwd()
	motivo, err := deshacerInit(cwd)
	if err != nil {
		fmt.Printf("[UNDO] No se pudo deshacer 'init': %v\n", err)
		return
	}
	if motivo != "" {
		fmt.Printf("[UNDO] No se borró nada: %s\n", motivo)
		return
	}
	fmt.Println("[UNDO] 'init' quedó deshecho.")
}

// deshacerInit borra .qdd solo si tiene únicamente lo que deja init y git no lo sigue; si no, dice por qué no.
func deshacerInit(dir string) (string, error) {
	qdd := filepath.Join(dir, ".qdd")
	if _, err := os.Stat(qdd); os.IsNotExist(err) {
		return "", nil
	}
	if gitSigueQdd(dir) {
		return "git ya sigue .qdd, así que no es de este init. Si de verdad quieres quitarlo, bórralo tú", nil
	}
	ajeno, err := conocimientoEn(dir, qdd)
	if err != nil {
		return "", err
	}
	if ajeno != "" {
		return ajeno + " es conocimiento del proyecto, no lo creó init. Si de verdad quieres quitar .qdd, bórralo tú", nil
	}
	return "", os.RemoveAll(qdd)
}

func gitSigueQdd(dir string) bool {
	c := exec.Command("git", "ls-files", "--", ".qdd")
	c.Dir = dir
	c.Env = gitenv.Limpio(os.Environ())
	salida, err := c.Output()
	return err == nil && len(strings.TrimSpace(string(salida))) > 0
}

// conocimientoEn devuelve el primer archivo de .qdd que init no crea, relativo a dir.
func conocimientoEn(dir, qdd string) (string, error) {
	ajeno := ""
	err := filepath.WalkDir(qdd, func(ruta string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, ruta)
		if err != nil || d.IsDir() || esDeInit(filepath.ToSlash(rel)) {
			return err
		}
		ajeno = rel
		return filepath.SkipAll
	})
	return ajeno, err
}

func esDeInit(rel string) bool {
	for _, p := range creadoPorInit {
		if rel == p || (strings.HasSuffix(p, "/") && strings.HasPrefix(rel, p)) {
			return true
		}
	}
	return false
}
