package cmd

import (
	"fmt"
	"time"

	"github.com/qdd-framework/qdd/pkg/baul"
)

// Rule: Prohibido usar 'else'
// Grupo del baúl en doctor (RFC 0023). Solo aparece con tokens guardados o un .env con secretos. --fix solo aparta un
// registro roto; un token vencido o rechazado lo cambia la persona en su terminal.

const grupoBaul = "Baúl de accesos"

func buildBaulChecks(projectPath string) *CheckGroup {
	g := &CheckGroup{Name: grupoBaul}
	for _, d := range baul.Diagnosticar(projectPath, time.Now()) {
		g.Items = append(g.Items, &CheckItem{Name: d.Nombre, Success: d.OK, Error: d.Detalle})
	}
	return g
}

func repairBaul(g *CheckGroup) {
	for _, item := range g.Items {
		if item.Success || item.Name != baul.ItemRegistro {
			continue
		}
		apartado, err := baul.ApartarRegistro(time.Now())
		if err != nil {
			fmt.Printf("    [!] No se pudo apartar el registro del baúl: %v\n", err)
			continue
		}
		fmt.Printf("    [✓] Registro del baúl roto apartado en %s; empieza otro\n", apartado)
	}
}
