# Datos de prueba de pkg/remoto

- `esperas.json`: copia sin cambios de `test/fixtures/esperas.json` de qdd-movile (commit `c016e1b`, 2026-11-10).
  Es el contrato de lo que espera a la persona y qué push lo avisa, compartido por la app, el relay y qdd.
  Lo lee `TestEsperan_CumpleElContratoDeQDDMovil_FND208` (`esperas_contrato_test.go`). Si qdd-movile lo cambia, se
  vuelve a copiar entero; no se edita aquí.
