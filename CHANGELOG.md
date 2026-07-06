# Changelog

Todos los cambios notables de este proyecto serán documentados en este archivo.

El formato está basado en [Keep a Changelog](https://keepachangelog.com/es-ES/1.0.0/), y este proyecto adhiere a [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
