package remoto

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/modules"
)

// Rule: Prohibido usar 'else'

// Quién aprobó (el teléfono y su nombre), qué, cuándo y cómo terminó queda en <QDD_HOME>/remoto/log.jsonl con 0600; lo
// que sale por el canal y por el relay no lleva quién lo hizo (inv. 5).
// contrato: remoto#549ed912
func TestAccion_QuienAproboQuedaEnElLogLocalYNuncaSaleAlRelay(t *testing.T) {
	homeTemporal(t)
	_, clave := proyectoConFicha(t, "pagos")
	_, tr, rl := bucleConFirma(t)
	tr.eventos <- telefono("d-7f3a", "")
	tr.esperar(t, time.Second, "estado")
	antes := time.Now().UTC().Add(-time.Second)
	a := firmada("1c6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, "aprobar:pagos", "d-7f3a", ahoraEn())
	a.DispositivoNombre = "Tablet de Rocío"
	mandarAccion(tr, a)
	for _, want := range []string{EstadoRecibida, EstadoEjecutando, EstadoHecha} {
		if got := estadoDe(t, tr, 5*time.Second); got.Estado != want {
			t.Fatalf("esperaba %s: %+v", want, got)
		}
	}
	time.Sleep(150 * time.Millisecond)

	log := lineasLog(t)
	if len(log) != 1 {
		t.Fatalf("una línea por la aprobación: %+v", log)
	}
	l := log[0]
	if l.Dispositivo != "d-7f3a" || l.Nombre != "Tablet de Rocío" || l.AccionID != "aprobar:pagos" || l.Proyecto != clave || l.Estado != EstadoHecha {
		t.Fatalf("quién, qué y cómo terminó: %+v", l)
	}
	if l.En.Before(antes) || l.En.After(time.Now().Add(time.Second)) {
		t.Fatalf("y cuándo: %v", l.En)
	}
	ruta, _ := rutaRemoto("log.jsonl")
	info, err := os.Stat(ruta)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("el log es 0600: %v", err)
	}

	for _, e := range tr.Enviados() {
		if strings.Contains(string(e.Payload), "Tablet de Rocío") || strings.Contains(string(e.Payload), "d-7f3a") {
			t.Fatalf("%s lleva quién aprobó: %s", e.Evento, e.Payload)
		}
	}
	tr.mu.Lock()
	defer tr.mu.Unlock()
	for _, b := range tr.tracks {
		if strings.Contains(string(b), "d-7f3a") {
			t.Fatalf("la presence del PC no lleva al teléfono: %s", b)
		}
	}
	for len(rl.avisos) > 0 {
		av := <-rl.avisos
		if strings.Contains(av.Tipo+av.Proyecto+av.Clave, "d-7f3a") {
			t.Fatalf("un aviso no lleva al teléfono: %+v", av)
		}
	}
}

// Sin firma válida, vieja, de un teléfono revocado o de un proyecto que el PC no publica: nada se ejecuta y cada una
// queda en el log local con su motivo (inv. 18).
// contrato: remoto#389ef195
func TestAccion_LoQueNoValeNoSeEjecutaYQuedaEnElLog(t *testing.T) {
	homeTemporal(t)
	root, clave := proyectoConFicha(t, "pagos")
	_, tr, _ := bucleConFirma(t)
	tr.eventos <- telefono("d1", "")
	tr.esperar(t, time.Second, "estado")
	if err := Revocar("d2"); err != nil {
		t.Fatal(err)
	}
	falsa := firmada("s-falsa", clave, "aprobar:pagos", "d1", ahoraEn())
	falsa.Firma = strings.Repeat("0", 64)
	casos := []AccionFirmada{
		falsa,
		firmada("s-vieja", clave, "aprobar:pagos", "d1", time.Now().Add(-6*time.Minute).UTC().Format(formatoEn)),
		firmada("s-revocado", clave, "aprobar:pagos", "d2", ahoraEn()),
		firmada("s-proyecto", "no-existe-0123", "aprobar:pagos", "d1", ahoraEn()),
	}
	for _, a := range casos {
		mandarAccion(tr, a)
	}
	limite := time.Now().Add(3 * time.Second)
	for time.Now().Before(limite) && len(lineasLog(t)) < len(casos) {
		time.Sleep(20 * time.Millisecond)
	}
	motivos := map[string]EntradaLog{}
	for _, l := range lineasLog(t) {
		motivos[l.Solicitud] = l
	}
	for sol, dice := range map[string]string{"s-falsa": "firma", "s-vieja": "vieja", "s-revocado": "revocado", "s-proyecto": "no publica"} {
		l, ok := motivos[sol]
		if !ok || (l.Estado != EstadoDescartada && l.Estado != EstadoRechazada) || !strings.Contains(l.Resultado, dice) {
			t.Errorf("%s queda en el log diciendo %q: %+v", sol, dice, l)
		}
	}
	mods, _ := modules.Load(root)
	if mods[0].Estado != modules.EstadoBorrador {
		t.Fatalf("ninguna aprobó la ficha: %s", mods[0].Estado)
	}
}
