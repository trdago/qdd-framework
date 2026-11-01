package candado

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestTomar_UnoALaVez(t *testing.T) {
	lock := filepath.Join(t.TempDir(), "x.lock")
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		dentro    int
		maxDentro int
	)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			soltar, err := Tomar(lock, 5*time.Second, "la prueba")
			if err != nil {
				t.Error(err)
				return
			}
			mu.Lock()
			dentro++
			if dentro > maxDentro {
				maxDentro = dentro
			}
			mu.Unlock()
			time.Sleep(time.Millisecond)
			mu.Lock()
			dentro--
			mu.Unlock()
			soltar()
		}()
	}
	wg.Wait()
	if maxDentro != 1 {
		t.Fatalf("nunca hay dos dentro: %d", maxDentro)
	}
}

func TestTomar_UnBloqueoHuerfanoSeDescartaYUnoOcupadoLoDice(t *testing.T) {
	lock := filepath.Join(t.TempDir(), "x.lock")
	if err := os.WriteFile(lock, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	viejo := time.Now().Add(-2 * Abandonado)
	if err := os.Chtimes(lock, viejo, viejo); err != nil {
		t.Fatal(err)
	}
	soltar, err := Tomar(lock, time.Second, "la prueba")
	if err != nil {
		t.Fatalf("el de un proceso que murió no bloquea para siempre: %v", err)
	}
	if _, err := Tomar(lock, 20*time.Millisecond, "la cola"); err == nil || !strings.Contains(err.Error(), "la cola: lo usa otro proceso") {
		t.Fatalf("uno vigente no se pisa y lo dice: %v", err)
	}
	soltar()
}
