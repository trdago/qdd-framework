# Datos de prueba de pkg/remoto

- `esperas.json`: copia sin cambios de `test/fixtures/esperas.json` de qdd-movile (commit `c016e1b`, 2026-11-10).
  Es el contrato de lo que espera a la persona y qué push lo avisa, compartido por la app, el relay y qdd.
  Lo lee `TestEsperan_CumpleElContratoDeQDDMovil_FND208` (`esperas_contrato_test.go`). Si qdd-movile lo cambia, se
  vuelve a copiar entero; no se edita aquí.
- `estado_partes.json`: lo escribe qdd. Son las tres partes de un estado de ejemplo, tal como las manda el PC a un
  teléfono con `estado_por_partes` (FND-209). qdd-movile lo copia sin cambios como fixture de la app. Lo comprueba
  `TestPartes_ElEjemploCompartidoEsLoQueMandaElPC_FND209`, y se regenera con `QDD_ACTUALIZAR_FIXTURE=1`.
- `cifrado1.json`: lo escribe qdd (RFC 0027, fase 2). Son las claves de prueba del PC, su clave firmada y cada campo
  que sella el teléfono (texto de ejecutar, respuesta, accion_id y datos de una parte de foto2) con la efímera fija, más
  dos casos que el PC rechaza con el motivo `clave`. qdd-movile lo copia sin cambios: la app tiene que sellar esos
  mismos bytes. Lo comprueban `TestFixture_EsLoQueSellaYAbreQDD_RFC0027` y
  `TestFixture_ElPCAbreLosBuenosYRechazaLosMalos_RFC0027` (`pkg/remoto/cifrado`), y se regenera con
  `QDD_ACTUALIZAR_FIXTURE=1`. Sus claves son de prueba: nunca se usan fuera de las pruebas.
