package estado

import (
	"os"
	"path/filepath"
	"strings"
)

// Rule: Prohibido usar 'else'
// RFC 0006 §3.9: lo local de un proyecto (la cola de pedidos y la actividad) es uno por repositorio. Un worktree se
// guarda como su checkout principal: la sesión que trabaja en un worktree y la del checkout principal ven el mismo
// panel. Se resuelve leyendo los archivos de git (.git, commondir), sin ejecutar git: la barra lo usa a cada rato.

// ClaveLocal es la carpeta que identifica al proyecto fuera del repositorio. En un worktree es la misma carpeta dentro
// del checkout principal; fuera de git, o si git no se puede leer, es la carpeta tal cual.
func ClaveLocal(root string) string {
	abs, err := filepath.Abs(root)
	if err != nil {
		return root
	}
	top := ancestroCon(abs, ".git")
	if top == "" {
		return abs
	}
	principal := checkoutPrincipal(top)
	rel, err := filepath.Rel(top, abs)
	if err != nil {
		return abs
	}
	return filepath.Join(principal, rel)
}

// ProyectoDe es el proyecto al que pertenece una carpeta: el ancestro más cercano con .qdd o, si no hay, la raíz de
// su repositorio. Fuera de los dos, ninguno ("").
func ProyectoDe(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	if p := ancestroCon(abs, ".qdd"); p != "" {
		return p
	}
	return ancestroCon(abs, ".git")
}

// ancestroCon es la carpeta más cercana (dir o un ancestro) que contiene nombre; "" si ninguna.
func ancestroCon(dir, nombre string) string {
	for d := dir; ; d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, nombre)); err == nil {
			return d
		}
		if filepath.Dir(d) == d {
			return ""
		}
	}
}

// checkoutPrincipal resuelve un worktree (.git es un archivo «gitdir: <repo>/.git/worktrees/<nombre>», con un
// commondir que apunta a <repo>/.git) a la carpeta del checkout principal. Un checkout normal es él mismo.
func checkoutPrincipal(top string) string {
	b, err := os.ReadFile(filepath.Join(top, ".git"))
	if err != nil {
		return top
	}
	gitdir := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(string(b)), "gitdir:"))
	if !filepath.IsAbs(gitdir) {
		gitdir = filepath.Join(top, gitdir)
	}
	common, err := os.ReadFile(filepath.Join(gitdir, "commondir"))
	if err != nil {
		return top
	}
	dir := strings.TrimSpace(string(common))
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(gitdir, dir)
	}
	dir = filepath.Clean(dir)
	if filepath.Base(dir) != ".git" {
		return top
	}
	return filepath.Dir(dir)
}
