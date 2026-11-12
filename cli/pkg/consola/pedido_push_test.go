package consola

import (
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// MEJ-053 (sfrc, 2026-11-09): cada push es el mismo comando, y el seguro de FND-154 lo rechazaba como «ya se hizo»
// aunque hubiera commits nuevos que subir; obligaba a pedir cada push con --de-nuevo.

func TestMEJ053_UnPushConCommitsNuevosNoEsRepetido(t *testing.T) {
	root := proyectoConCola(t)
	gitEn(t, root, "init", "-q", "-b", "main")
	gitEn(t, root, "-c", "commit.gpgsign=false", "commit", "-q", "--allow-empty", "-m", "uno")
	anotarHecho(root, Pedido{Comando: "git push origin main", Carpeta: root, Motivo: "Subir main"}, time.Now())

	_, err := Pedir(root, "git push origin main", "Subir main otra vez", "", time.Now())
	esYaHecho(t, err)

	gitEn(t, root, "-c", "commit.gpgsign=false", "commit", "-q", "--allow-empty", "-m", "dos")
	if _, err := Pedir(root, "git push origin main", "Subir main con el commit dos", "", time.Now()); err != nil {
		t.Fatalf("con un commit nuevo, el mismo push se puede pedir: %v", err)
	}
}
