package remoto

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// RFC 0013, inv. 16: nada que parezca un secreto sale del PC. Pruebas con los patrones conocidos y con lo que no es
// un secreto y no debe taparse.

func TestOcultar_PatronesConocidos(t *testing.T) {
	casos := []struct {
		nombre, entrada, secreto string
	}{
		{"openai", "usa sk-proj-AbCdEfGhIjKlMnOpQrStUvWx12 para llamar", "sk-proj-AbCdEfGhIjKlMnOpQrStUvWx12"},
		{"openai clásico", "OPENAI sk-AbCdEfGhIjKlMnOpQrStUvWxYz0123456789", "sk-AbCdEfGhIjKlMnOpQrStUvWxYz0123456789"},
		{"anthropic", "clave sk-ant-api03-AbCdEfGhIjKl_mnop-QRST", "sk-ant-api03-AbCdEfGhIjKl_mnop-QRST"},
		{"stripe", "sk_live_51HAbCdEfGhIjKlMn", "sk_live_51HAbCdEfGhIjKlMn"},
		{"github", "token ghp_0123456789abcdefghijABCDEFGHIJ012345", "ghp_0123456789abcdefghijABCDEFGHIJ012345"},
		{"github oauth", "gho_0123456789abcdefghijABCDEFGHIJ", "gho_0123456789abcdefghijABCDEFGHIJ"},
		{"github pat", "github_pat_11ABCDEFG0123456789_abcdefghijklmnop", "github_pat_11ABCDEFG0123456789_abcdefghijklmnop"},
		{"aws", "aws AKIAIOSFODNN7EXAMPLE listo", "AKIAIOSFODNN7EXAMPLE"},
		{"slack bot", "xoxb-1234567890-abcdefghij", "xoxb-1234567890-abcdefghij"},
		{"slack user", "xoxp-1234567890-abcdefghij", "xoxp-1234567890-abcdefghij"},
		{"supabase secreta", "SUPA sb_secret_AbCdEfGhIjKlMnOp", "sb_secret_AbCdEfGhIjKlMnOp"},
		{"google", "AIzaSyA1234567890abcdefghijklmnopqrstuv", "AIzaSyA1234567890abcdefghijklmnopqrstuv"},
		{"jwt", "Token: eyJhbGciOiJIUzI1NiJ9.eyJyb2xlIjoic2VydmljZV9yb2xlIn0.c2lnbmF0dXJl fin", "eyJhbGciOiJIUzI1NiJ9.eyJyb2xlIjoic2VydmljZV9yb2xlIn0.c2lnbmF0dXJl"},
		{"bearer", "curl -H 'Authorization: Bearer abcdef0123456789xyz' https://api", "abcdef0123456789xyz"},
		{"postgres", "conectando a postgres://admin:SuperClave123@db.ejemplo.com:5432/app", "SuperClave123"},
		{"postgresql", "postgresql://u:p4ss@host/db", "p4ss"},
		{"mysql", "mysql://root:toor!@localhost/x", "toor!"},
		{"mongodb+srv", "mongodb+srv://user:Mongo%40Pass@cluster0.mongodb.net/test", "Mongo%40Pass"},
		{"redis sin usuario", "redis://:RedisPass99@cache:6379/0", "RedisPass99"},
		{"llave privada", "-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXktdjEAAAA\nAAAAB3NzaC1yc2E\n-----END OPENSSH PRIVATE KEY-----", "b3BlbnNzaC1rZXktdjEAAAA"},
		{"llave rsa", "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA\n-----END RSA PRIVATE KEY-----", "MIIEpAIBAAKCAQEA"},
		{"llave cortada", "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkq sin final", "MIIEvQIBADANBgkq"},
		{"env API_KEY", "API_KEY=abc123def", "abc123def"},
		{"env export", "export GH_TOKEN=\"tok-1234\"", "tok-1234"},
		{"env password", "DB_PASSWORD='hunter2'", "hunter2"},
		{"env pwd", "PWD_ADMIN=xyz", "xyz"},
		{"env secret en medio", "corriendo con JWT_SECRET=s3cr3t y listo", "s3cr3t"},
		{"env credential", "GOOGLE_CREDENTIALS=/home/u/creds.json", "/home/u/creds.json"},
		{"env dsn", "SENTRY_DSN=https://abc@o1.ingest.sentry.io/1", "https://abc@o1.ingest.sentry.io/1"},
		{"env database url", "DATABASE_URL=postgres://u:pw@h/db", "postgres://u:"},
		{"yaml", "api_key: valor-secreto-1", "valor-secreto-1"},
		{"json", `{"access_token": "tok_abcdef"}`, "tok_abcdef"},
		{"pass", "SMTP_PASS=correo123", "correo123"},
	}
	for _, c := range casos {
		got := Ocultar(c.entrada)
		if strings.Contains(got, c.secreto) || !strings.Contains(got, Oculto) {
			t.Errorf("%s: %q → %q", c.nombre, c.entrada, got)
		}
	}
}

func TestOcultar_LoQueNoEsSecretoQueda(t *testing.T) {
	casos := []string{
		"commit 9f8c2a1b3e4d5f60718293a4b5c6d7e8f9012345 en main",
		"/home/dpailahueque/Documents/proyectos/qdd/cli/pkg/remoto/ocultar.go:42",
		"solicitud 0b6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f50 recibida",
		"go test ./pkg/remoto/... ok 2.4s",
		"task-list risk-free disk-usage",
		"KEYBOARD=es MONKEY=no PASSING_TESTS=12",
		"la tarea «Quitar claves de Gemini» está ✅",
		"https://github.com/trdago/qdd/pull/66",
		"postgres://db.ejemplo.com:5432/app sin credenciales",
		"sb_publishable_AbCdEfGhIjKl es la publicable",
		"eyJ solo, sin puntos",
		"qdd:pc:0b6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f50 canal",
	}
	for _, c := range casos {
		if got := Ocultar(c); got != c {
			t.Errorf("no es secreto y se tapó: %q → %q", c, got)
		}
	}
}

func TestOcultador_ValoresDeLosEnvDelProyecto(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "frontend"), 0o755)
	os.MkdirAll(filepath.Join(root, "node_modules", "x"), 0o755)
	os.WriteFile(filepath.Join(root, ".env"), []byte("# comentario\nSUPABASE_URL=https://abcdefgh.supabase.co\nNOMBRE_RARO=valorlargo99\nCORTO=abc\nexport OTRO=\"con comillas 123\"\nMODO=production # comentario\n"), 0o600)
	os.WriteFile(filepath.Join(root, "frontend", ".env.production"), []byte("VITE_COSA=frontsecreto42\n"), 0o600)
	os.WriteFile(filepath.Join(root, "node_modules", "x", ".env"), []byte("NO=noseleeesto99\n"), 0o600)
	o := OcultadorDe(root)
	texto := "llamando a https://abcdefgh.supabase.co/rest con valorlargo99, con comillas 123 y frontsecreto42; abc; modo production; noseleeesto99"
	got := o.Ocultar(texto)
	for _, s := range []string{"abcdefgh.supabase.co", "valorlargo99", "con comillas 123", "frontsecreto42"} {
		if strings.Contains(got, s) {
			t.Errorf("el valor %q del .env se tapa donde aparezca: %q", s, got)
		}
	}
	for _, s := range []string{"abc;", "production", "noseleeesto99"} {
		if !strings.Contains(got, s) {
			t.Errorf("%q no se tapa (corto, común o de node_modules): %q", s, got)
		}
	}
	var nada *Ocultador
	if nada.Ocultar("API_KEY=x") != "API_KEY="+Oculto {
		t.Fatal("sin proyecto, igual oculta los patrones")
	}
	if Recortar("abcdef", 3) != "abc…" || Recortar("ab", 3) != "ab" {
		t.Fatal("recortar")
	}
}
