package cmd

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"github.com/qdd-framework/qdd/pkg/foto"
)

// Rule: Prohibido usar 'else'

// Botón 📷 del panel (pedido de la persona, 2026-11-07): el panel pone en el chat la ruta corta de la foto, y un error se
// lo muestra a la persona.
func TestFotoJSON_RutaCortaParaElChatOElError(t *testing.T) {
	cwd := t.TempDir()
	f := foto.Foto{Ruta: filepath.Join(cwd, "tmp", "fotos", "a.png"), Fuente: "portapapeles (wl-paste)", Tipo: "image/png", Bytes: 2048}
	b, codigo := fotoJSON(cwd, f, nil)
	var r map[string]any
	if err := json.Unmarshal(b, &r); err != nil || codigo != 0 {
		t.Fatalf("%s %d %v", b, codigo, err)
	}
	if r["ruta_corta"] != filepath.Join("tmp", "fotos", "a.png") || r["ruta"] != f.Ruta || r["tipo"] != "image/png" || r["bytes"] != float64(2048) {
		t.Fatalf("%v", r)
	}
	b, codigo = fotoJSON(cwd, foto.Foto{}, errors.New("portapapeles: no hay imagen"))
	r = map[string]any{}
	json.Unmarshal(b, &r)
	if codigo != 1 || r["error"] != "portapapeles: no hay imagen" || r["ruta"] != nil {
		t.Fatalf("un error sale con 1 y solo dice el error: %s %d", b, codigo)
	}
}
