package baul

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Rule: Prohibido usar 'else'
// FND-202 (2026-11-13, prueba de punta a punta con un op falso): modificar un token guardado en 1Password creaba un
// segundo ítem con el mismo título. 1Password los deja convivir, y después «item get» falla con «more than one item».
// Este op falso se comporta como el real: varios ítems con el mismo título, y get, delete y edit exigen uno solo.

const opComoElReal = `#!/bin/sh
D="$OP_ITEMS"
mkdir -p "$D"
uno() {
  m=$(grep -l "\"title\":\"$1\"" "$D"/* 2>/dev/null)
  c=$(printf '%s' "$m" | grep -c .)
  [ "$c" = 1 ] || { echo "\"$1\": $c items" >&2; exit 1; }
  echo "$m"
}
case "$1 $2" in
"item create") cat > "$D/$(date +%s%N)"; echo '{}' ;;
"item get") grep -o 'valor-[a-z0-9-]*' "$(uno "$3")" ;;
"item delete") rm "$(uno "$3")" ;;
"item edit") f=$(uno "$3"); [ "$4" = "--title" ] || exit 2; sed -i "s/\"title\":\"$3\"/\"title\":\"$5\"/" "$f" ;;
*) exit 2 ;;
esac
`

func opReal(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("op falso en sh")
	}
	dir := t.TempDir()
	op := filepath.Join(dir, "op")
	if err := os.WriteFile(op, []byte(opComoElReal), 0o755); err != nil {
		t.Fatal(err)
	}
	items := filepath.Join(dir, "items")
	t.Setenv("OP_ITEMS", items)
	t.Cleanup(func(v string) func() { return func() { binOp = v } }(binOp))
	binOp = op
	return items
}

func TestFND202_ModificarEnUnoPasswordDejaUnSoloItem(t *testing.T) {
	entorno(t)
	items := opReal(t)
	e := Entrada{Nombre: "pg-app", Tipo: "otro", Variable: "PGPASSWORD", Donde: EnUnoPass}
	for _, valor := range []string{"valor-uno-123456", "valor-dos-123456", "valor-tres-123456"} {
		if err := Guardar(e, valor, ahora); err != nil {
			t.Fatalf("guardar %s: %v", valor, err)
		}
	}
	quedan, _ := os.ReadDir(items)
	if len(quedan) != 1 {
		t.Fatalf("modificar reemplaza el ítem: quedan %d", len(quedan))
	}
	guardada, _ := Buscar("pg-app")
	v, err := leerValor(guardada)
	if err != nil || v != "valor-tres-123456" {
		t.Fatalf("se lee el último valor: %q %v", v, err)
	}
}

func TestFND202_SiNoSePuedeBorrarElAnteriorNoQuedaElNuevo(t *testing.T) {
	entorno(t)
	items := opReal(t)
	e := Entrada{Nombre: "pg-app", Tipo: "otro", Variable: "PGPASSWORD", Donde: EnUnoPass}
	if err := Guardar(e, "valor-uno-123456", ahora); err != nil {
		t.Fatal(err)
	}
	// Dos ítems con el mismo título (lo que dejaba el bug): 1Password no sabe cuál borrar.
	if err := guardarEnOp(tituloOp("pg-app"), "valor-copia-123456"); err != nil {
		t.Fatal(err)
	}
	if err := Guardar(e, "valor-dos-123456", ahora); err == nil {
		t.Fatal("si no puede borrar el anterior, lo dice")
	}
	quedan, _ := os.ReadDir(items)
	if len(quedan) != 2 {
		t.Fatalf("no deja el ítem nuevo a medio camino: quedan %d", len(quedan))
	}
}
