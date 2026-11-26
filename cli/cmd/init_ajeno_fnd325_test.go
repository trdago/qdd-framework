package cmd

import (
	"strings"
	"testing"
)

// FND-325 (MEJ-102): qdd init salía con 1 cuando el doctor final encontraba algo que init no instala ni puede
// reparar, como el teléfono que desvinculó el PC (REM-15). init responde solo por lo que deja en el proyecto; lo demás
// lo muestra como aviso.
func TestInit_LoAjenoNoHaceFallarInit_FND325(t *testing.T) {
	c := &Checklist{Groups: []*CheckGroup{
		{Name: grupoEstructura, Items: []*CheckItem{{Name: "Directorio raíz .qdd/", Success: true}}},
		{Name: grupoPuertas, Items: []*CheckItem{{Name: itemPuertasActivas, Success: true}}},
		{Name: grupoMovil, Items: []*CheckItem{{Name: "Teléfono vinculado (REM-15)", Error: "el teléfono desvinculó este PC"}}},
	}}
	propias, ajenas := anomaliasDeInit(c)
	if propias != 0 {
		t.Fatalf("lo de QDD Móvil no es trabajo de init: %d propias", propias)
	}
	if len(ajenas) != 1 || !strings.Contains(ajenas[0], "REM-15") || !strings.Contains(ajenas[0], "desvinculó") {
		t.Fatalf("lo ajeno se muestra con su nombre y su causa: %q", ajenas)
	}
}

func TestInit_LoPropioSigueFallando_FND325(t *testing.T) {
	for _, grupo := range []string{grupoEstructura, grupoIA, grupoDashboard, grupoPuertas} {
		c := &Checklist{Groups: []*CheckGroup{{Name: grupo, Items: []*CheckItem{{Name: "algo de init", Error: "falta"}}}}}
		if propias, ajenas := anomaliasDeInit(c); propias != 1 || len(ajenas) != 0 {
			t.Errorf("%s es trabajo de init: %d propias, %q ajenas", grupo, propias, ajenas)
		}
	}
}
