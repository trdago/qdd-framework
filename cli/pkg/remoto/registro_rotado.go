package remoto

import (
	"os"
	"path/filepath"
	"sync"
)

// Rule: Prohibido usar 'else'
// Sprint 9 (FND-238): lanzado por qddw.exe, qdd remoto no tiene consola, y en Windows no hay un journal que recoja su
// salida: escribe su registro en <QDD_HOME>/remoto/correr.log. Al pasar TopeRegistroCorrer guarda una sola copia
// (correr.log.1) y empieza de nuevo: el registro nunca ocupa más de dos topes.

const (
	TopeRegistroCorrer = 1 << 20
	nombreRegistro     = "correr.log"
)

// RutaRegistroCorrer es el registro de qdd remoto cuando corre sin consola.
func RutaRegistroCorrer() (string, error) {
	return rutaRemoto(nombreRegistro)
}

// RegistroRotado es un archivo de registro con tope y una copia.
type RegistroRotado struct {
	mu     sync.Mutex
	ruta   string
	tope   int64
	f      *os.File
	tamano int64
}

// AbrirRegistroRotado abre (o crea) el registro y sigue al final de lo que ya tenía.
func AbrirRegistroRotado(ruta string, tope int64) (*RegistroRotado, error) {
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(ruta, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	return &RegistroRotado{ruta: ruta, tope: tope, f: f, tamano: info.Size()}, nil
}

// Write escribe; si con esto pasaría el tope, antes guarda lo que había como la copia. Una escritura más grande que el
// tope se escribe entera.
func (r *RegistroRotado) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.tamano > 0 && r.tamano+int64(len(p)) > r.tope {
		if err := r.rotar(); err != nil {
			return 0, err
		}
	}
	n, err := r.f.Write(p)
	r.tamano += int64(n)
	return n, err
}

func (r *RegistroRotado) rotar() error {
	if err := r.f.Close(); err != nil {
		return err
	}
	if err := os.Rename(r.ruta, r.ruta+".1"); err != nil {
		return err
	}
	f, err := os.OpenFile(r.ruta, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	r.f, r.tamano = f, 0
	return nil
}

// Close cierra el archivo.
func (r *RegistroRotado) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.f.Close()
}
