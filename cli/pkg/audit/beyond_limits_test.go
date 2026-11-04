package audit

import (
	"os"
	"path/filepath"
	"testing"
)

// Rule: Prohibido usar 'else'
// FND-134: CERT-021 miraba solo el nombre del método y bloqueaba el gate por leer la query string de una URL.

func lineasCert021(t *testing.T, cuerpo string) int {
	t.Helper()
	dir := t.TempDir()
	fuente := "package p\n\nfunc f() {\n" + cuerpo + "\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte(fuente), 0o644); err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, v := range RunBeyondLimitsCheck(dir) {
		if v.RuleID == "CERT-021-CHAOS-TOLERANCE" {
			n++
		}
	}
	return n
}

func TestCert021_LeerLaQueryStringNoEsUnaConsulta_FND134(t *testing.T) {
	for _, cuerpo := range []string{
		`consulta := r.URL.Query()`,
		`formato := r.URL.Query().Get("formato")`,
		`v := u.Query()`,
	} {
		if n := lineasCert021(t, cuerpo); n != 0 {
			t.Errorf("%q no toca la base y se marcó %d veces", cuerpo, n)
		}
	}
}

func TestCert021_SiguenMarcadasLasConsultasSinContexto_FND134(t *testing.T) {
	for _, cuerpo := range []string{
		`filas, err := db.Query("select 1")`,
		`fila := db.QueryRow("select 1")`,
		`_, err := tx.Exec("delete from t")`,
		`_, err := stmt.Exec()`,
		`filas, err := stmt.Query(1)`,
	} {
		if n := lineasCert021(t, cuerpo); n != 1 {
			t.Errorf("%q es una consulta sin contexto y se marcó %d veces", cuerpo, n)
		}
	}
}
