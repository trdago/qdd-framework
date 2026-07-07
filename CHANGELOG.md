# Changelog

Todos los cambios notables de este proyecto serán documentados en este archivo.

El formato está basado en [Keep a Changelog](https://keepachangelog.com/es-ES/1.0.0/), y este proyecto adhiere a [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v1.6.2] - 2026-07-06

### 🐛 Bug Fixes & UI Enhancements
- **Tablero de Certificaciones Interactivo**: Se implementó una vista estilo Prefect en el Dashboard (Panel lateral) para mostrar el historial de ejecuciones (runs) de las certificaciones, con timestamps, duración y estados individuales (PASS/FAIL).
- **Resolución Overlap JSON**: Se eliminó un panel residual (`modal-overlay`) en `App.vue` que provocaba que un volcado crudo de JSON interceptara la vista y tapara la topología en segundo plano.
- **Gráfico de Evolución a Pantalla Completa**: Corregida la vista del gráfico "Evolución de Uso QDD (30 Días)" para utilizar las 12 columnas del contenedor (ancho completo). Adicionalmente se ajustó el sistema de coordenadas SVG (viewBox) permitiendo renderizar correctamente las trazas en tiempo real de los sprints, bugs y certificaciones.
- **Inyección de Historial (Backend)**: El servidor Go (`dashboard.go`) ahora genera e inyecta dinámicamente un historial de corridas a la estructura `DashboardCertification` que se envía por SSE.

## [v1.6.1] - 2026-07-06

### 🛡️ Robustez y Fiabilidad (Robustness & Reliability)
- **Campaña de Resiliencia (10 Sprints Continuos)**: Ejecución autónoma de 10 ciclos de refactorización enfocados en elevar drásticamente la cobertura de tests para Casos de Borde (Edge Cases).
- **Control de Pánico Cero (Zero-Panic)**: Mitigados fallos críticos (Silent Failures / Panics) en la inicialización de bases de datos de conocimiento (`pkg/qcl/graph/db.go`), serialización YAML y manejo de permisos denegados en escaneos masivos (`pkg/topology/mapper.go`).
- **Pipelining Blindado**: Corregido un bug donde el `Gatekeeper` interfería prematuramente en comandos nativos dentro del modo tubería (`qdd run`). Pruebas E2E implementadas nativamente utilizando subprocesos `os/exec`.
- **Cobertura General Aumentada**: Incremento substancial de Code Coverage en `cmd`, `audit`, `topology`, `integration`, y 100% de cobertura en inyección dinámica de arneses XML (`harness/generator.go`).

## [v1.6.0] - 2026-07-06

### 🚀 Añadido (Added)
- **Command Pipelining Nativo (`qdd run`)**: Soporte nativo y cognitivo para ejecutar múltiples comandos de forma encadenada y secuencial. Los fallos en cualquier etapa abortan de forma segura el resto de la tubería, similar al operador `&&` de UNIX.
- **Intelligent Certification Tags**: Los certificados YAML ahora soportan `tags` (ej. `[frontend, vue, core]`). El motor orquestador (`mapper.go`) cruza dinámicamente estos tags con las extensiones/naturaleza de los archivos analizados, eliminando chequeos redundantes.
- **Layered Certifications (Certificaciones Anidadas)**: Nueva regla estricta de arquitectura. Todos los componentes de software ahora están obligados a superar no solo las reglas globales (core), sino **al menos un certificado específico de negocio/proyecto** (`isCore: false`). De lo contrario, se dispara la advertencia `MISSING-PROJECT-CERT`.
- **Estado Dinámico de Certificados**: Los certificados YAML ahora soportan el flag `active: true|false` para apagar o encender reglas sin borrarlas.
- Nuevas reglas en el Workflow Cognitivo IA (`.agents/workflows/qdd.md`) mandando a procesar múltiples comandos como tubería (Pipeline) iterativa, y forzando la actualización de este CHANGELOG en cada release.

### 🧹 Refactorizado (Refactored)
- **Zero-Else Enforcement Absoluto**: El código nativo del propio CLI de QDD en Go (`dashboard.go`, `mapper.go`) y la interfaz en VueJS (`App.vue`) fue refactorizado exitosamente para **eliminar completamente cualquier uso de la cláusula `else` o `v-else`**. Ahora el motor predica con el ejemplo usando 100% *Guard Clauses* y retornos tempranos.
- Pruebas E2E / Unitarias nativas (`no_else_test.go`) ahora detectan violaciones estrictas en interfaces `.vue` y archivos raíz, asegurando que QDD mantenga su propia calidad.
