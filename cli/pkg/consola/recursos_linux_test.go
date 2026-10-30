package consola

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGpusDRM_NvidiaLaMideNvidiaSmiIntegradaSinLecturaYOrdenPorNumero(t *testing.T) {
	raiz := t.TempDir()
	tarjeta := func(nombre, vendor, ocupada string) {
		dev := filepath.Join(raiz, nombre, "device")
		if err := os.MkdirAll(dev, 0755); err != nil {
			t.Fatal(err)
		}
		if vendor != "" {
			_ = os.WriteFile(filepath.Join(dev, "vendor"), []byte(vendor+"\n"), 0644)
		}
		if ocupada != "" {
			_ = os.WriteFile(filepath.Join(dev, "gpu_busy_percent"), []byte(ocupada+"\n"), 0644)
		}
	}
	tarjeta("card1", "0x10de", "")
	tarjeta("card10", "0x1002", "42")
	tarjeta("card2", "0x8086", "")
	tarjeta("card1-eDP-1", "0x8086", "99")
	got := gpusDRM(raiz)
	if len(got) != 2 || got[0] != nil || got[1] == nil || *got[1] != 42 {
		t.Fatalf("card2 (Intel, sin contador) «—» y card10 (AMD) 42 %%: %v", got)
	}
}
