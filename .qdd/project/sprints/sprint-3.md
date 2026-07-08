# Sprint 3

## Objetivos (Sprint Goal)
- [x] Restringir la capacidad de modificar el core del framework únicamente al comando `qdd doctor`.
- [x] Establecer un módulo de Control de Acceso basado en la jerarquía del comando de origen.
- [x] Dotar a `qdd doctor` de capacidades de reparación (`--fix`).

## Tareas (Backlog)
- [x] Implementar `GuardCoreWriteAccess` en `cli/pkg/qcl/auth/core_guard.go`.
- [x] Extender la API de `RunDoctorCheck` en `cli/cmd/doctor.go` para añadir parametrización `autoFix`.
- [x] Modificar `cli/cmd/init.go` para autorizar bootstrap de `core_assets` pero bloquear sobrescrituras indiscriminadas.
- [x] Incluir tests unitarios que aseguren que simulaciones de comandos no autorizados (ej. `validate`) reboten con Access Denied.

## Métricas de Calidad Iniciales
- **Seguridad Framework:** El core ahora posee inmutabilidad nativa ante comandos erráticos o ejecuciones automatizadas del workflow que no sean de reparación controlada.

---
*Gobernanza QDD: Todo código añadido en este sprint debe contar con evidencia (EV-FND) y pruebas unitarias.*
