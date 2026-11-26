# Datos de prueba de pkg/remoto

- `esperas.json`: copia sin cambios de `test/fixtures/esperas.json` de qdd-movile (commit `c016e1b`, 2026-11-10).
  Es el contrato de lo que espera a la persona y qué push lo avisa, compartido por la app, el relay y qdd.
  Lo lee `TestEsperan_CumpleElContratoDeQDDMovil_FND208` (`esperas_contrato_test.go`). Si qdd-movile lo cambia, se
  vuelve a copiar entero; no se edita aquí.
- `estado_partes.json`: lo escribe qdd. Son las tres partes de un estado de ejemplo, tal como las manda el PC a un
  teléfono con `estado_por_partes` (FND-209). qdd-movile lo copia sin cambios como fixture de la app. Lo comprueba
  `TestPartes_ElEjemploCompartidoEsLoQueMandaElPC_FND209`, y se regenera con `QDD_ACTUALIZAR_FIXTURE=1`.
