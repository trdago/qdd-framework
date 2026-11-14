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

- **Fase 2: Persistencia Fractal (Generación de Sprint)**
  El Agente documenta y formaliza qué se va a hacer. **REGLA FRACTAL:** Si la funcionalidad es un sub-componente (ej. `input_user` dentro de `login`), el sprint DEBE anidarse en la estructura de carpetas: `.qdd/project/sprints/login/input_user.md`. El motor GraphRAG leerá esto recursivamente e inferirá la relación matemática. Todo código debe tener trazabilidad.
  *Criterio de salida:* Sprint documentado en estructura jerárquica.

- **Fase 3: TDD Determinista (Golden Sets)**
  El Agente escribe EXCLUSIVAMENTE los tests unitarios bajo la arquitectura **Data-Driven (Golden Set)**, respetando la estructura anidada.
  - *Nueva Funcionalidad:* El agente crea la carpeta reflejando la jerarquía, ej: `.qdd/project/goldensets/login/input_user/` y genera los JSON `happy_path`, `bad_path` y `edge_case`.
  - *Bugs:* El agente **genera un nuevo archivo JSON** (ej. `bug_<descripcion>.json`) en el goldenset correspondiente.
  - Luego, implementa/asegura que exista un runner en el código (`goldenset.RunSuite(t, "login/input_user", handler)`) específico para esa micro-funcionalidad.
  *Criterio de salida:* Archivos de Golden Set anidados creados y test unitario comprobado que falla.

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
- **/qdd pull request** (o **/qdd pr**) (Asimilación de Aportes Externos)
  - *Entrada:* Fase 1 (Lectura del Certificado `.qdd/QDD_PR_CERTIFICATE.md` o enlace/archivo adjunto provisto por el usuario).
  - *Flujo:* F1 (Entender aporte y contexto del proyecto) -> F3 (Adaptar/Crear tests para el aporte) -> F4 (Integrar el código propuesto) -> F5 (Validar reglas QDD como Zero-Else).
- **/qdd resolve** (Equipo Resolutor FAANG)
  - *Entrada:* Fase 1.
  - *Flujo:* Orquestador lanza Analista Integral (F1-F2) -> Implementador (F3-F4) -> QA (F5) -> DevOps (F5+ Despliegue). Aborda problemas complejos con un escuadrón multi-agente de nivel empresarial garantizando 100% estabilidad.
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
1. **Zero-Else & Guard Clauses:** Nunca uses la sentencia `else`. Usa `return` o `continue` tempranos para manejar flujos alternativos o de error. Por qué: cada vez menos personas leen el código línea a línea, así que el que sí se lee tiene que entenderse de un vistazo.
2. **Zero-Mocks (Real Infrastructure):** Nunca hacemos mocks de dependencias. Si se requiere probar lógica de persistencia, el Agente debe levantar una base de datos SQLite real. Las pruebas deben ser deterministas contra infraestructuras reales.
3. **UI Predictiva:** Al requerir decisión del usuario, usa modales interactivos (`ask_question`), opciones estructuradas o comandos sugeridos antes que pedir input libre.
4. **Casos de Borde Obligatorios:** Todos los tests en Fase 3 deben evaluar timeouts, nulls y límites lógicos, no solo el "camino feliz". TODO bug genera un test.
5. **Paralelismo Contextual y Orquestación FAANG:** Si el agente lanza Sub-agentes (paralelismo) o invoca el Equipo Resolutor, el Orquestador DEBE garantizar el pase estricto del testigo de la Fase 1 a la 5 entre agentes (Analista, Implementador, QA, DevOps) reportando el estado en vivo al usuario.
6. **Pipelining (Cadenas):** Si recibes múltiples comandos (`/qdd bug sprint release`), se ejecutan iterativamente, pasando por la máquina de estados en el orden lógico.
7. **Auto-Asociación:** Si el usuario pide algo sin usar el comando `/qdd` (ej: "corrige la tabla"), el Agente DEBE auto-clasificar la intención en el comando más cercano (ej. `/qdd bug` o `/qdd sprint`) y verbalizarlo ("Asignando esta tarea a `/qdd bug`, iniciando en Fase 3...").
8. **No Fallos Silenciosos y Mocks en Tests Únicamente:** El Agente (Famas) tiene prohibido escribir fallos silenciosos; todo error debe manejarse explícitamente. Asimismo, no puede generar mocks para el flujo de ejecución del código, estos solo son permitidos en tests.
9. **Gestión de Tareas Largas y Modo Objetivo (/goal / Long-Running Tasks):** En tareas complejas o de larga duración, el Agente DEBE descomponer el trabajo en un checklist atómico e iterativo (`- [ ]`, `- [/]`, `- [x]`), generar un `implementation_plan.md` previo y NO detenerse hasta completar el 100% de los ítems con certificación en `walkthrough.md`.
10. **Principio Core de Unicidad del Dato (SSOT - CERT-034):** Todo dato debe tender a la unicidad en el modelo y base de datos; cualquier desnormalización o duplicación debe justificarse formalmente con un ADR en `docs/adr/`.

### QDD AUTOGENERATED COMMANDS BEGIN ###
# QDD Command Protocol (QCP) v1.0
# QCP block revision: 5
# Managed by QDD: never edit this block; strictly follow the QDD rules in .qdd/.
# COMMANDS: when the user types /qdd or /qdd init, validate, certify, review, ui, api, db, release, resolve, goal, report, evolution, docs, run it as an AI agent ('qdd --help'). Work asked without a /qdd command: before starting, say in one line the closest QDD command you apply and why.
# BEFORE CHANGING CODE call qdd_project_context with the paths you will touch (no MCP: 'qdd modules context <paths>'): it gives the module record and the method. Tests first; zero else, because fewer people read code line by line (AIs write and review it) and what is read must be understood at a glance: guard clauses on top, straight main path. Every bug is a finding with a regression test. No record: propose one with qdd_module_save (a human approves: 'qdd modules approve <id>'). Never commit or deploy unless 'qdd audit' passes.
# BLUEPRINTS (organization standards): to add one, qdd_blueprint_add; no organization repository registered: ask the user its URL, never invent one. Before building a module a blueprint defines, qdd_blueprint_apply and ask its design questions exactly as returned: never answer them yourself. Its invariants are not negotiable. They live in QDD's central folder (the project versions only .qdd/blueprints.lock.yaml): never copy or edit one there. To change or create one: qdd_blueprint_contribute, edit where it says, then with the user's confirmation qdd_blueprint_propose (never set a version). Finish a migration or BP-05-LEGACY as told and commit; on BP-06-BEHIND, qdd_blueprint_add and re-apply; missing blueprints or pending migration: 'qdd doctor --fix'.
# CURRENT TASK: board and phone show as running only the task you declare: qdd_estado accion=activar (no MCP: qdd estado activa <task>). It goes out after 2 minutes idle: declare it again. Done or waiting for someone: accion=terminar (--fin).
# COMMANDS FOR THE HUMAN (no permission, blocked, or it publishes: release tag, npm publish, PR merge): never ask them to type it; queue it with qdd_consola_pedir (no MCP: qdd consola pedir -m '<why>' '<command>') saying why. It runs when they confirm it in the panel; never run it yourself with 'qdd consola accion'. Declare dependencies (despues/antes; --despues/--antes). Queue only what can run now: a command waiting on something outside the queue (tag, CI, another session's merge) checks it and does 'exit 75' (gray «todavía no»). One command per human step; a chain of merges in one command. Another repository's command goes to its panel.
# QUESTIONS FOR THE HUMAN never stay only in the chat: a choice goes through AskUserQuestion with a recommended option; what is not a choice is a ⏳ with qdd_estado whose blocker is a question (¿…?), which reaches the phone; a command goes with qdd_consola_pedir. A ⏳ that asks nothing is a progress note ('qdd estado check' lists them).
# TEMPORARY FILES go in ./tmp, never in the root or source folders; move what is worth keeping to its place with its test.
# PERMISSION MODE: in auto mode (also acceptEdits, bypassPermissions, dontAsk) continue without asking what follows the project's line (open tasks, tests, findings, work within approved records and blueprints). Ask first only to change something defined: an approved module record's objective, decisions or invariants; a blueprint decision or invariant; a version, tag, publication or deployment; anything that deletes data or cannot be undone. Blueprint design questions stay the human's.
# GOAL MODE: only the human turns it on (panel or phone): never activate it, queue it or ask for it. While on, QDD picks your questions' recommended option, runs your queued commands and approves your records, all recorded. A message starting with 🎯 means work remains: take the next open task and reflect what you finish in the estado. Always mark a recommended option: a question without one, a blueprint design question or a tool permission stops the mode until the human answers.
### QDD AUTOGENERATED COMMANDS END ###
