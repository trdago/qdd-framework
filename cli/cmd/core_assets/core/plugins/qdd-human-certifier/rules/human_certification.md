# PROTOCOLO DE CERTIFICACIÓN DE COMPORTAMIENTO HUMANO (QDD HUMAN QA)

## 1. PRINCIPIO DE CERTIFICACIÓN HUMANA
Todo flujo crítico de usuario (autenticación, formularios, navegación, dashboards, pasarelas y portales públicos) debe ser certificado mediante simulación de interacción humana real, auditoría en base de datos y evaluación multimodal antes de ser liberado a producción.

---

## 2. REGLAS DE EJECUCIÓN OBLIGATORIAS

### 1. Stack Primario Playwright en Golang (`playwright-go`):
- El estándar predeterminado para la construcción y ejecución de suites de interacción humana es **Golang** con `playwright-go`.
- Los scripts deben estar fuertemente tipados, estructurados en módulos reutilizables y con gestión explícita de recursos (`defer`).

### 2. Carpeta de Evidencia Aislada por Ejecución (1 Run = 1 Carpeta):
- Toda ejecución genera un directorio aislado (`.qdd/evidence/run_<TIMESTAMP>_<UUID>/`).
- Debe contener obligatoriamente: `manifest.json`, `screenshots/`, `traces/`, `db_snapshots/`, `network_logs/` y `backend_logs/`.

### 3. El LLM es el Juez y Certificador Final (LLM-as-a-Judge):
- Las aserciones mecánicas de código solo recopilan datos; **el LLM que invoca la prueba es la autoridad exclusiva que decide si el flujo pasa (`PASS`) o falla (`FAIL`)**.
- El LLM evalúa capturas visuales, integridad en base de datos, códigos de respuesta HTTP y ergonomía general antes de emitir el certificado.

### 4. Verificación Obligatoria de Persistencia en Base de Datos:
- Ninguna prueba se aprueba solo por la apariencia en la UI.
- Se debe consultar directamente la Base de Datos antes, durante y después de cada mutación para constatar que los datos fueron efectivamente persistidos y que no existen excepciones registradas en tablas de errores del backend.

### 5. Trazabilidad Continua de Datos (Single-Entity Lineage):
- En flujos completos, se debe generar un token único (`TRACE_ID`) e inyectarlo en la entidad de prueba.
- Se debe validar paso a paso (creación, listado, detalle, edición, eliminación) que en todo instante se está operando sobre **exactamente el mismo dato**, impidiendo el desvío hacia datos precargados o ficticios.

### 6. Zero-Else y Early Return Estricto:
- Queda terminantemente prohibido el uso de sentencias `else` o `else if` en los scripts de prueba.
- Toda función debe resolver errores y condiciones de salida en las primeras líneas con `return`.

### 7. Doble Viewport Obligatorio (Desktop & Mobile):
- Toda suite debe validar Viewport Desktop (`1440x900`) y Viewport Mobile (`390x844`).
- Ningún elemento interactivo clave debe quedar oculto, cortado o inaccesible en pantallas táctiles.

### 8. Cero Esperas Ciegas (Zero Flakiness):
- Prohibido el uso de `waitForTimeout` estáticos.
- Se debe sincronizar con el estado reactivo del DOM (`waitForSelector`, `waitForURL`, `waitForFunction`).

### 9. Documentación Obligatoria de Bugs (Findings):
- Cualquier anomalía detectada por el LLM o los scripts debe documentarse de inmediato como un `Finding` estructurado y acompañarse de un test unitario para prevenir regresiones.
