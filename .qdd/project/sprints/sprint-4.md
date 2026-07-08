# Sprint 4

## Objetivos (Sprint Goal)
- [x] Garantizar que `qdd init` sea estrictamente no destructivo (adaptar y organizar, nunca borrar).
- [x] Implementar un loop iterativo dinámico que evalúe y auto-repare (`autoFix`) hasta alcanzar integridad del 100%.
- [x] Modificar la lógica de "Deep Merge" del adaptador de Cursor (y aplicarlo como regla a futuras IA) para prevenir borrados accidentales de configuración ajena al framework.

## Tareas (Backlog)
- [x] Modificar `runInit` y `runInitIteration` en `cli/cmd/init.go` para usar un `while` con contador de caídas (`failCount`) para prevenir cuelgues (anti-hangs) y evaluar éxito real.
- [x] Refactorizar la lógica `JSON Unmarshal/Marshal` en `cli/pkg/integration/cursor.go` para hacer *Deep Merge* y no truncar llaves primarias o de otros servidores ajenos a QDD.
- [x] Ajustar la firma de `RunDoctorCheck` en `cli/cmd/doctor.go` para retornar recuento matemático de anomalías.
- [x] Escribir tests de regresión para validar que el `mcp.json` no destruya llaves existentes.

## Métricas de Calidad Iniciales
- **Resiliencia de Configuración:** QDD ahora respeta otras integraciones ajenas a QDD que el usuario tenga configurado en sus herramientas.
- **Idempotencia:** Ejecutar `qdd init` $N$ veces consecutivas resulta exactamente en el mismo estado sin pérdida de datos.

---
*Gobernanza QDD: Todo código añadido en este sprint debe contar con evidencia (EV-FND) y pruebas unitarias.*
