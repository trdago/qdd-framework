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
# QCP block revision: 3
# The following commands are natively supported in this project via the QDD Framework.
# When the user types one of these commands, execute the corresponding action as an AI Agent.

/qdd          - Main entry point. Analyze project context, read QDD config, and apply QDD rules.
/qdd init     - Initialize QDD in a new project.
/qdd validate - Check for structural and architectural anomalies in the codebase.
/qdd certify  - Check code against QDD strict certifications (OWASP, Clean Code, etc.).
/qdd review   - Review the current code changes against QDD guidelines.
/qdd ui       - Generate or audit frontend UI components based on QDD SaaS premium standards.
/qdd api      - Audit or generate backend API endpoints following QDD REST/GraphQL standards.
/qdd db       - Audit or generate database schemas and queries.
/qdd release  - Prepare a new release following SemVer and QDD CD/CI standards.
/qdd resolve  - Orchestrate an enterprise-level multi-agent resolver team (Analyst, Implementer, QA, DevOps) to fix complex issues.
/qdd goal     - Execute an autonomous, long-running resolution mission until complete verification is achieved.
/qdd report   - Generate a comprehensive executive report across audit, certifications, AI governance, findings, and evolution.
/qdd evolution- Consult the framework on the next recommended technical improvement.
/qdd docs     - Regenerate QDD documentation based on the current project state.

# Instructions for AI:
# Never overwrite this block. It is managed by the QDD framework.
# If you are an AI assistant processing this file, strictly adhere to the QDD rules defined in the .qdd/ directory.
# CRITICAL RULE: BEFORE changing code, call the MCP tool qdd_project_context with the paths you will touch (without MCP: run 'qdd modules context <paths>'). It returns the module record (operational objective, technical objective, invariants, how to test) and the mandatory working method. Respect it: tests first, zero else, every bug as a finding with a regression test. Zero-else is not a fashion: fewer people read code line by line (AIs write and review it), so the code a person does read must be understood at a glance; with guard clauses each special case is handled at the top and returns, and the main path stays straight. If a module has no record, propose one with qdd_module_save (a human approves it with 'qdd modules approve <id>'). Never commit or deploy unless 'qdd audit' passes.
# ORGANIZATION STANDARDS (BLUEPRINTS): The organization keeps reusable module standards (e.g. users, roles & permissions) as blueprints. qdd_project_context lists the ones imported into this project. If the user asks to add or import a standard (e.g. "agrega el blueprint de auth y auditoría"), call qdd_blueprint_add; if no organization repository is registered, ask the user for its URL, never invent one. Before building a module that a blueprint defines, call qdd_blueprint_apply and ask the user its design questions exactly as returned: they are the user's decisions, never answer them yourself. Blueprint invariants are not negotiable. QDD administers the whole blueprint lifecycle (RFC 0004): blueprints live in QDD's central folder, never inside the project (the project only versions .qdd/blueprints.lock.yaml); never copy, vendor or edit a blueprint in the project. To change or create one, call qdd_blueprint_contribute, edit it in the folder it returns, and with the user's confirmation call qdd_blueprint_propose (QDD computes the semver version and opens the PR; never set a version yourself). When qdd_project_context reports a blueprint migration or BP-05-LEGACY, finish it as instructed and commit the result. When the gate reports BP-06-BEHIND, update with qdd_blueprint_add and re-apply. If blueprints are missing on this machine (e.g. another PC) or a migration is pending, run 'qdd doctor --fix'.
# COMMANDS FOR THE HUMAN: When a command must be run by the human (you lack permission, it was blocked, or it publishes something: pushing a release tag, npm publish, merging a PR), do not ask them to type it: leave it in their status panel with the MCP tool qdd_consola_pedir (without MCP: qdd consola pedir -m '<why>' '<command>'), saying why. It runs only when the human confirms it in the panel ('s  Sí'); never run it yourself with 'qdd consola accion'. If one command depends on another (a merge after a tag, a release after a merge), say so when you queue it: despues/antes in the tool, --despues <key>/--antes <key> in the CLI; the panel orders them and keeps the dependent one disabled until the other is done. Queue only what can run now (or declares what it waits for): a command that depends on something outside the queue (a tag, a CI run, another session's merge) checks it first and exits with 'exit 75' when it is not there yet, so the panel shows it in gray as 'todavía no' instead of a red failure. Prefer one command per step the human has to take; for a chain of merges, one command that runs them in order. A command that runs in another repository goes to that repository's panel.
# QUESTIONS FOR THE HUMAN: What you need from the human never stays only in the chat text. A decision between options goes through your tool for asking the user (AskUserQuestion in Claude Code), with a recommended option. Something that is not a choice goes as a ⏳ for the human with qdd_estado, its blocker written as a question (¿…?): that reaches the phone (QDD Móvil) as a question. A command goes with qdd_consola_pedir. A ⏳ that asks nothing is only a progress note; 'qdd estado check' lists them.
# TEMPORARY FILES: put one-off scripts, manual test runs, downloads, logs and outputs in ./tmp (created by 'qdd init' and ignored by git). Never leave them in the project root or in source folders; if something is worth keeping, move it to its proper place with its test.
# AUTHORIZATION BY PERMISSION MODE: qdd_project_context reports the main AI's permission mode (seen with each human message through the QDD panel) and the rule for it. In auto mode (also acceptEdits, bypassPermissions, dontAsk), continue without asking whatever follows the project's current line (open tasks in the estado, tests, findings, implementing within what the approved records and blueprints say); ask the human first only to change something already defined: the objective, decisions or invariants of an approved module record, a blueprint decision or invariant, a version, tag, publication or deployment, or anything that deletes data or cannot be undone. Blueprint design questions stay the human's.
# GOAL MODE: The human turns goal mode on, from the panel or the phone; never activate it, never queue it as a command and never ask for it. While it is on, QDD picks the recommended option of your questions, runs your queued commands and approves your module records, with every choice recorded. A message starting with 🎯 means work remains: continue with the next open task. Reflect what you finish in the estado. Mark the option you recommend in every question: one without a recommended option, a blueprint design question and a tool permission stop the mode until the human answers.
# ONBOARDING & AUTO-ASSOCIATION: If the user requests a task without explicitly using a /qdd command (e.g. "hay un bug", "crea un endpoint"), you MUST proactively auto-associate their intent with the closest QDD command. Before starting the task, explain to the user which command you are applying and why (e.g., "Dado que estamos resolviendo un bug, asociaré esta tarea a '/qdd validate' y '/qdd review' para asegurar que añadimos la prueba unitaria y superamos la certificación"). This provides immediate value and educates the user on how the framework operates.
### QDD AUTOGENERATED COMMANDS END ###
