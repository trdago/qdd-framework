# Changelog

Todos los cambios notables de este proyecto serán documentados en este archivo.

El formato está basado en [Keep a Changelog](https://keepachangelog.com/es-ES/1.0.0/), y este proyecto adhiere a [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v1.7.6] - 2026-07-07

### 🐛 Bug Fixes
- **Dashboard Data Binding:** Se ha corregido un bug crítico donde el frontend del Dashboard renderizaba gráficas con datos iniciales (mock data) en lugar de refrescar con los valores provenientes de `cognitive_history.json`.
- **SSE Connection Resilience:** Se ha mitigado un fallo en ambientes Docker/WSL y configuraciones de NGINX Proxy donde la conexión SSE se quedaba colgada sin enviar estado inicial. Se ha introducido envío manual pre-listener de `state` y la cabecera `X-Accel-Buffering: no` para prevenir congelamiento por buffering de proxies.
- **Terminal output fix:** Se corrigió un mensaje estático que indicaba erróneamente que el servidor inicializaba en `localhost:8080`, lo que desorientaba a desarrolladores en despliegues remotos.

## [v1.7.5] - 2026-07-07

### ✨ Nuevas Características
- **QDD Doctor (`qdd doctor`)**: Nuevo comando de diagnóstico introducido para asegurar mediante pruebas deterministas que el entorno del QDD Framework y sus integraciones estén completamente configurados y operativos. Genera automáticamente un reporte/checklist de evidencia en `.qdd/project/evidence/doctor/`.
- **`qdd init` Autocorrectivo**: El comando de inicialización ha sido refactorizado para ejecutarse en un loop de hasta 3 iteraciones validando cada pasada contra el comando `qdd doctor`. Esto garantiza la correcta creación de toda la estructura de directorios, integraciones (`.cursor`, `.clauderc`) e inicialización de estados antes de entregar un éxito o de abortar, mejorando la resiliencia en todo el framework.

## [v1.7.4] - 2026-07-07

### 🐛 Bug Fixes
- **AI Integrations (MCP)**: Se solucionó un error crítico donde `qdd init` configuraba el servidor MCP en `.cursor/mcp.json` usando el string estático `"qdd"`. Ahora la inicialización inyecta dinámicamente la ruta absoluta del binario (ej. `/home/user/.local/bin/qdd`) permitiendo a los IDEs como Cursor y Antigravity conectarse correctamente sin depender de configuraciones complejas de `$PATH`.

## [v1.7.3] - 2026-07-07

### 🐛 Bug Fixes (Hotfix)
- **NPM Publish Sync (Real)**: Se corrigió la versión del archivo `npm/package.json` para reflejar la versión `v1.7.3` (anteriormente estancado en 1.5.1), desbloqueando de forma definitiva la publicación en el registro de NPM.

## [v1.7.2] - 2026-07-07

### 🐛 Bug Fixes (Hotfix)
- **NPM Publish Sync**: Sincronización de las versiones en `package.json` y `cli/ui/package.json` con la versión del CLI `v1.7.2` para permitir la correcta ejecución del pipeline de publicación en NPM.

## [v1.7.1] - 2026-07-07

### 🐛 Bug Fixes (Hotfix)
- **Zero-Else Violations**: Corregidas varias violaciones a la política "Zero-Else" en `App.vue` introducidas en v1.7.0. Se reemplazaron por `v-if` negados y *Guard Clauses*.
- **Database Audit Panic**: Corregido un `panic: not implemented` en los tests y la ejecución del motor de auditoría `database.go` al utilizar erróneamente `fs.ReadFile(osFS{})`. Reemplazado de manera segura por `os.ReadFile`.
- **Limpieza de Archivos de Prueba**: Eliminación de archivos temporales (`test_sqlite.go`, `cli/test.db`, etc.) que causaban fallos de build (`main redeclared in this block`) durante el testing nativo.

## [v1.7.0] - 2026-07-07

### 🗺️ Project Map: Mega-Jerarquía Conceptual (D3 Ecosystem Graph)
- **Mega-Árbol D3 Unificado**: El "Project Map" ha sido rediseñado arquitecturalmente desde cero. En lugar de aplanar los archivos en un único nivel, ahora construye un **árbol maestro del ecosistema** con 5 ramas conceptuales: Módulos (Código), Deuda Técnica, Sprints Agile, Certificaciones y Documentación. El Nivel 1 muestra únicamente los "planetas" de abstracción, con profundidad expandible hasta Nivel 10.
- **Filtros de Conceptos Interactivos**: Nueva fila de botones-píldora (`Código`, `Bugs`, `Sprints`, `Certs`, `Docs`) posicionada junto al slider de abstracción. Permiten activar/desactivar ramas completas del árbol D3 en tiempo real, re-renderizando el `d3.pack` dinámicamente sin recargar la página.
- **Colorimetría por Categoría**: Cada familia conceptual tiene ahora una paleta de colores propia (Azul=Código, Rojo=Bugs, Naranja=Sprints, Verde=Certs, Violeta=Docs) aplicada tanto al fill/stroke de los círculos SVG como al texto, usando las funciones `getNodeFill`, `getNodeStroke` y `getNodeTextFill`.
- **Slider de Abstracción Extendido**: El control "DETALLE" se amplió de `max=6` a `max=10` para permitir exploraciones más profundas de la jerarquía de carpetas y módulos.

### 🛠️ Backend: Especialista de Tuning PostgreSQL (MCP)
- **Herramienta MCP `qdd_postgres_tuner`**: Se agregó un especialista de rendimiento de base de datos al servidor MCP, invocable desde cualquier agente externo. Provee recomendaciones anti-patrones (ej. `SELECT COUNT(*)` sin filtros → `pg_class`), análisis de queries y guías de optimización con certificación de base de datos.
- **Regla de Auditoría `DB-PERF-01`**: Nueva regla en el motor de auditoría (`cli/pkg/audit/database.go`) que detecta el anti-patrón `COUNT(*) sin filtros` en el código fuente y lo reporta como finding de alta severidad, generando evidencia trazable en el dashboard.
- **Registro en `mcp.go`**: La herramienta `registerPostgresTunerTool` fue registrada correctamente en el arranque del servidor MCP.

### 🧹 Mantenimiento
- **Actualización `.gitignore`**: Se añadieron exclusiones para archivos temporales de desarrollo (`test.db`, `test_sqlite.go`, `test_sync.go`, `testyaml.go`, `run_sync.go`, `_tmp_old_app.vue`, `.qdd/working/`), manteniendo el repositorio limpio.

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
