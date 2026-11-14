package consola

import (
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// MEJ-026 (issue #150): lo que necesita a la persona llega siempre al panel y al teléfono; qdd lo exige en vez de confiar
// en la IA. La persona: «siempre debería salir en el tablero […] qdd debería obligarte, ya que si estoy desde el celular
// debería poder continuar». Al terminar un turno, si la IA cierra con una pregunta o le pide correr un comando sin
// haberlo dejado en el panel, qdd le devuelve el turno una vez con lo que falta.

func TestFinDeTurno_UnaPreguntaSinPanelVuelveALaIA(t *testing.T) {
	for _, mensaje := range []string{
		"Recomiendo empezar por la 1. ¿Parto con esa?",
		"Elegí tres mejoras.\n\n¿Las implemento todas o solo la primera?\n",
		"Listo el PR. **¿Lo mergeo?**",
		"Should I merge it?",
	} {
		falta := QueFaltaAlTerminar(TurnoTerminado{Mensaje: mensaje})
		if !strings.Contains(falta, "qdd_estado") || !strings.Contains(falta, "para qué") {
			t.Errorf("%q: vuelve con qué dejar (un ⏳ con su para qué): %q", mensaje, falta)
		}
	}
}

func TestFinDeTurno_UnComandoParaLaPersonaSinPanelVuelveALaIA(t *testing.T) {
	for _, mensaje := range []string{
		"Falta el login. Corre `gcloud auth login` en tu terminal.",
		"Para seguir:\n- ejecuta `npm publish --tag preview`",
		"Escribe `! gh auth login` y sigo.",
	} {
		falta := QueFaltaAlTerminar(TurnoTerminado{Mensaje: mensaje})
		if !strings.Contains(falta, "qdd_consola_pedir") || !strings.Contains(falta, "de_persona") {
			t.Errorf("%q: vuelve pidiendo el pedido en el panel: %q", mensaje, falta)
		}
	}
}

func TestFinDeTurno_LoQueNoPideNadaNoVuelve(t *testing.T) {
	for _, mensaje := range []string{
		"",
		"Listo: rc.69 está en npm preview.",
		"¿Por qué falló? Porque el índice tenía la dirección vieja. Ya está arreglado.",
		"Corrí `go test ./...` y pasó todo.",
		"Ejecuté `qdd audit`: 100/100.",
		"Escribe `/reload-plugins` para cargar el panel nuevo.",
		"El texto del ⏳ dice «¿Sigo?» en el panel.",
		"Así se pide:\n```\n¿Sigo?\n```",
	} {
		if falta := QueFaltaAlTerminar(TurnoTerminado{Mensaje: mensaje}); falta != "" {
			t.Errorf("%q: un resumen, una pregunta retórica o un comando ya corrido no vuelven: %q", mensaje, falta)
		}
	}
}

func TestFinDeTurno_SiQuedoEnElPanelNoVuelve(t *testing.T) {
	pregunta := "¿Parto con esa?"
	for nombre, llamada := range map[string]LlamadaDelTurno{
		"⏳ por la tool":      {Tool: "mcp__qdd__qdd_estado", Entrada: map[string]any{"accion": "set", "estado": "espera", "bloqueo": "¿Parto?"}},
		"⏳ con el emoji":     {Tool: "mcp__qdd__qdd_estado", Entrada: map[string]any{"accion": "set", "estado": "⏳"}},
		"pedido por la tool": {Tool: "mcp__qdd__qdd_consola_pedir", Entrada: map[string]any{"comando": "gh pr merge 7", "motivo": "Integrar"}},
		"AskUserQuestion":    {Tool: "AskUserQuestion", Entrada: map[string]any{"questions": []any{}}},
		"ficha en borrador":  {Tool: "mcp__qdd__qdd_module_save", Entrada: map[string]any{"ficha_yaml": "id: x"}},
		"pedido por la CLI":  {Tool: "Bash", Entrada: map[string]any{"command": "qdd consola pedir -m 'Integrar' gh pr merge 7"}},
		"⏳ por la CLI":       {Tool: "Bash", Entrada: map[string]any{"command": "qdd estado set x pr --estado espera --bloqueo '¿Sigo?' --de-quien persona"}},
	} {
		if falta := QueFaltaAlTerminar(TurnoTerminado{Mensaje: pregunta, Herramientas: []LlamadaDelTurno{llamada}}); falta != "" {
			t.Errorf("%s: ya quedó en el panel, no vuelve: %q", nombre, falta)
		}
	}
	for nombre, llamada := range map[string]LlamadaDelTurno{
		"un ✅":             {Tool: "mcp__qdd__qdd_estado", Entrada: map[string]any{"accion": "set", "estado": "ok"}},
		"quitar un pedido": {Tool: "mcp__qdd__qdd_consola_pedir", Entrada: map[string]any{"resuelto": "a"}},
		"otra herramienta": {Tool: "Read", Entrada: map[string]any{"file_path": "x"}},
		"un ✅ por la CLI":  {Tool: "Bash", Entrada: map[string]any{"command": "qdd estado set x pr --estado ok --evidencia y"}},
		"leer los pedidos": {Tool: "Bash", Entrada: map[string]any{"command": "qdd consola pedidos"}},
	} {
		if falta := QueFaltaAlTerminar(TurnoTerminado{Mensaje: pregunta, Herramientas: []LlamadaDelTurno{llamada}}); falta == "" {
			t.Errorf("%s: no deja nada para la persona, vuelve", nombre)
		}
	}
}

func TestFinDeTurno_VuelveUnaSolaVez(t *testing.T) {
	if falta := QueFaltaAlTerminar(TurnoTerminado{Mensaje: "¿Parto con esa?", YaDevuelto: true}); falta != "" {
		t.Fatalf("el turno que ya volvió una vez termina, sin bucle: %q", falta)
	}
}

func TestFinDeTurno_ElMensajeCitaLaPregunta(t *testing.T) {
	falta := QueFaltaAlTerminar(TurnoTerminado{Mensaje: "Hay dos caminos. ¿Prefieres A o B?"})
	if !strings.Contains(falta, "«¿Prefieres A o B?»") {
		t.Fatalf("dice cuál pregunta falta: %q", falta)
	}
	larga := "¿" + strings.Repeat("palabra ", 40) + "?"
	if falta := QueFaltaAlTerminar(TurnoTerminado{Mensaje: larga}); len([]rune(falta)) > 600 {
		t.Fatalf("una pregunta larga se recorta: %d", len([]rune(falta)))
	}
}
