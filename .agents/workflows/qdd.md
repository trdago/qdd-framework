---
description: QDD Framework native AI commands and Cognitive Harness
---

# QDD AGENTIC HARNESS & COGNITIVE WORKFLOW
Este documento define la Máquina de Estados Finita (Círculo Virtuoso) que rige TODAS las interacciones de desarrollo dentro del QDD Framework. Cualquier Inteligencia Artificial (Agente) operando en este repositorio **DEBE** someterse a este ciclo. Está prohibido escribir código de producción sin haber recorrido las fases previas correspondientes.

## EL CÍRCULO VIRTUOSO (CORE STATE MACHINE)
El ciclo de desarrollo en QDD es determinista y consta de 5 Fases estandarizadas.

- **Fase 1: Contextualización (Certificaciones y Diseño)**
  El Agente recopila contexto leyendo archivos, explorando el código y preguntando al humano.
  *Criterio de salida:* Cero ambigüedades. Es lógico y claro lo que se va a construir.

- **Fase 2: Persistencia (Generación de Sprint)**
  El Agente documenta y formaliza qué se va a hacer (ej. actualizando o creando un documento de Sprint en `.qdd/project/sprints/`). Todo código debe tener trazabilidad.
  *Criterio de salida:* Sprint documentado.

- **Fase 3: TDD Determinista (Golden Sets)**
  El Agente escribe EXCLUSIVAMENTE los tests unitarios bajo la arquitectura **Data-Driven (Golden Set)**. 
  - *Nueva Funcionalidad:* El agente crea la carpeta en `.qdd/project/goldensets/<feature>/` y genera los JSON `happy_path`, `bad_path` y `edge_case`.
  - *Bugs:* El agente **genera un nuevo archivo JSON** (ej. `bug_<descripcion>.json`) en el goldenset para asegurar que la regresión quede atrapada en la base de datos de pruebas.
  - Luego, implementa/asegura que exista un runner en el código (`goldenset.RunSuite`) que lea e itere sobre esta carpeta (El test debe fallar inicialmente, `Red`).
  *Criterio de salida:* Archivos de Golden Set creados/actualizados y test unitario comprobado que falla.

- **Fase 4: Construcción Certificada (Code)**
  El Agente escribe la funcionalidad de producción, adhiriendo estrictamente a la Regla QDD: **"Cero Else" y Retornos Tempranos**, además de las certificaciones (Clean Code, OWASP, etc.).
  *Criterio de salida:* Implementación codificada.

- **Fase 5: Validación Continua (The Loop)**
  El Agente ejecuta `make test` y `qdd audit`. Si falla o hay errores, el Agente DEBE retornar iterativamente a la Fase 4 (o Fase 3 si el test estaba mal) hasta que todo esté en verde (Success).
  *Criterio de salida:* 100% Tests Pass + Cero Violaciones de Auditoría.

---

## QDD COMMANDS (PUNTOS DE ENTRADA)
El usuario invocará comandos (o intenciones auto-asociadas) que insertarán al Agente en un punto específico de la Máquina de Estados.

### Flujos Principales de Desarrollo
- **/qdd sprint** (Nueva Funcionalidad)
  - *Entrada:* Fase 1.
  - *Flujo:* F1 -> F2 -> F3 -> F4 -> F5.
- **/qdd bug** (Resolución de Bugs / Errores)
  - *Entrada:* Fase 3.
  - *Flujo:* F3 (Escribir test que reproduzca el bug) -> F4 (Fix) -> F5 (Validar).
- **/qdd validate** (Auditoría Continua y Certificación)
  - *Entrada:* Fase 5.
  - *Flujo:* F5 (Validar). Si falla, retroceder a F4 para auto-reparar -> F5.
- **/qdd learn** (Asimilación y Exploración)
  - *Entrada:* Fase 1.
  - *Flujo:* F1 -> F2 (Documentar hallazgos). *Termina aquí. No escribe código ni tests.*
- **/qdd release** (Ciclo de Despliegue)
  - Ejecuta Fase 5 para garantizar estabilidad. Si pasa, actualiza `CHANGELOG.md`, incrementa versiones (SemVer) sin hardcodear, hace commit, tag y push (o instruye al usuario a hacerlo).

### Comandos Auxiliares
- **/qdd init**     - Inicializa QDD en un nuevo proyecto.
- **/qdd certify**  - Equivalente a `/qdd validate` pero orientado a reglas estrictas (OWASP, Clean Code).
- **/qdd review**   - Revisa cambios actuales contra directrices QDD.
- **/qdd ui / api / db** - Ejecutan el Círculo Virtuoso (F1 a F5) enfocado específicamente en Frontend, Backend, o Base de Datos.

---

## REGLAS GLOBALES INQUEBRANTABLES (QDD PHILOSOPHY)
1. **Zero-Else & Guard Clauses:** Nunca uses la sentencia `else`. Usa `return` o `continue` tempranos para manejar flujos alternativos o de error.
2. **UI Predictiva:** Al requerir decisión del usuario, usa modales interactivos (`ask_question`), opciones estructuradas o comandos sugeridos antes que pedir input libre.
3. **Casos de Borde Obligatorios:** Todos los tests en Fase 3 deben evaluar timeouts, nulls y límites lógicos, no solo el "camino feliz". TODO bug genera un test.
4. **Paralelismo Contextual:** Si el agente lanza Sub-agentes (paralelismo), debe reportar al usuario qué rama o funcionalidad se está interviniendo.
5. **Pipelining (Cadenas):** Si recibes múltiples comandos (`/qdd bug sprint release`), se ejecutan iterativamente, pasando por la máquina de estados en el orden lógico.
6. **Auto-Asociación:** Si el usuario pide algo sin usar el comando `/qdd` (ej: "corrige la tabla"), el Agente DEBE auto-clasificar la intención en el comando más cercano (ej. `/qdd bug` o `/qdd sprint`) y verbalizarlo ("Asignando esta tarea a `/qdd bug`, iniciando en Fase 3...").
