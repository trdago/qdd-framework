# Architecture

QDD (Quality-Driven Development) es una Plataforma de Ingeniería de Software basada en Mejora Continua. Se compone de múltiples capas para asegurar la calidad y preservar el conocimiento del sistema desde el día cero.

## The Wisdom Registry (El Cerebro)

El núcleo absoluto de QDD es el **Wisdom Registry** ubicado en `.qdd/core/wisdom/`. Esta es la mente del proyecto.
En lugar de depender del conocimiento efímero de los desarrolladores o de un LLM aislado, QDD almacena su Constitución aquí:
- **`manifesto.md`**: Define las reglas inquebrantables del framework (Production-First, Quality-First).
- **`principles.md`**: Contiene las reglas de ingeniería (No-Else, Timeouts, Contratos).

Todo Motor Cognitivo que opere sobre el repositorio **debe** asimilar el Wisdom Registry antes de interactuar con el código.

## Governance (Gobernanza)

La Gobernanza en QDD es el mecanismo que asegura que el proyecto cumpla con los estándares arquitectónicos, apoyado en el principio de **"Rules-as-Code"**.

### ¿Cómo agregar estándares a un componente o proyecto?

El framework te ofrece un enfoque de 3 capas para definir y aplicar estándares:

#### 1. La Vía Cognitiva (A través del IDE y MCP)
Dado que QDD gobierna a la IA, puedes delegarle la creación del estándar directamente a tu IDE (Claude/Cursor) usando lenguaje natural:
```bash
/qdd "Agrega un estándar estricto que prohíba usar la librería Axios a favor de fetch nativo en el componente UI"
```
QDD operará como Gatekeeper. Si detecta que falta un estándar de la industria, el servidor MCP bloqueará el código libre y exigirá la adopción de un estándar antes de generar el código. Si lo autorizas, creará automáticamente el artefacto de certificación YAML.

#### 2. La Vía Estructural (Archivos de Certificación)
Si prefieres el control manual, los estándares se guardan como archivos YAML en la carpeta `.qdd/core/certification/` (framework) o `.qdd/project/certification/` (proyecto). 
Para crear un nuevo estándar, simplemente creas un archivo (ej: `CERT-010-UI-PERFORMANCE.yaml`):

```yaml
id: CERT-010
title: "Performance Base del Dashboard UI"
status: pending  # Empezará como pendiente hasta que qdd certify lo valide
level: strict
description: "Todos los componentes de Vue deben compilar en menos de 500ms y el bundle no debe superar los 100KB."
criteria:
  - "No usar librerías pesadas como lodash o moment."
  - "Implementar Lazy Loading en las rutas principales."
```
El *Gatekeeper* de QDD no permitirá que el `QDD Score` llegue a 100 hasta que este estándar pase a estado `certified`.

#### 3. Sincronización con Asistentes IA (Protocolo QCP)
Para que tu IDE respete las reglas automáticamente mientras programas, QDD escribe tus estándares, el Wisdom Registry y el Manifiesto en los archivos de reglas de cada asistente: `.clauderc` (Claude Code), `.cursorrules` (Cursor), `.antigravityrules` (Antigravity) y `AGENTS.md` (OpenCode y agentes genéricos). Lo hace `qdd init` y, después, la tool MCP `qdd_sync`, que puedes pedirle a tu IA cuando cambies un estándar. Así, la filosofía "Production-First" se inyecta directamente en el "cerebro" del modelo de lenguaje en tu IDE.

---
**Resumen del Flujo**: 
Tú dictas la regla (o la IA la propone en Modo Consultivo) ➔ QDD la registra como una Certificación ➔ `qdd_sync` la inyecta en la IA ➔ `qdd certify` valida que el código la cumpla.

## Fichas de módulo (El Corazón de cada módulo)

Las certificaciones dicen *cómo* se construye; las **fichas de módulo** dicen *para qué* existe cada parte del sistema. Una ficha (`.qdd/project/modules/<id>.yaml`) guarda:

- **Objetivo operativo**: el problema que resuelve, para quién, cómo se mide el éxito y el caso real que lo motivó.
- **Objetivo tecnológico**: la arquitectura, las decisiones vigentes y las **invariantes** (lo que ningún cambio puede romper).
- **Cómo se prueba**: los comandos exactos y los goldensets del módulo.
- Sus ADRs, findings y las fuentes de donde salió.

Es el contexto que una IA necesita para tocar un módulo sin perder su intención:

1. Antes de cambiar código, la IA llama a la tool MCP `qdd_project_context` con las rutas que va a tocar y recibe la ficha, la forma de trabajo obligatoria y cómo se prueba (sin MCP: `qdd modules context <rutas>`).
2. Si un módulo no tiene ficha, la IA la propone con `qdd_module_save`. Siempre queda en **borrador**.
3. Un humano la aprueba con `qdd modules approve <id>`, que registra el commit revisado.
4. El gate vigila el desfase: `HEART-02` avisa cuando el código cambió después de la aprobación y `HEART-03` señala zonas de código sin ficha (política `module_heart: true`).

## Blueprints (Los Estándares de la Organización)

Una organización repite los mismos módulos en todos sus productos (usuarios y roles, notificaciones, auditoría). Un **blueprint** es el esqueleto conceptual de uno de esos módulos: vocabulario, preguntas de diseño, decisiones, invariantes y escenarios de conformidad.

La separación es estricta:

- **El core de QDD** (público) trae solo el mecanismo: importar (`qdd blueprint add`), aplicar (`qdd blueprint apply`) y vigilar (reglas `BP-01` a `BP-04`).
- **El contenido** vive en el repositorio de cada organización, privado si quiere, versionado con tags. El proyecto lo registra una vez en `.qdd/config.yaml` y fija lo importado en `.qdd/blueprints.lock.yaml`.

Aplicar un blueprint genera la **ficha del módulo** en borrador: así, el estándar de la organización entra al mismo ciclo de aprobación humana y de vigilancia que cualquier otra ficha. Las preguntas de diseño las responde siempre un humano; la IA las hace, no las contesta.

Guía completa: [Blueprints](blueprints.md). Diseño: [RFC 0003](../rfcs/0003-blueprints.md).

## Specification

La capa de **Specification** es el conjunto de reglas agnósticas que definen la estructura y comportamiento de QDD. Al estar documentada de forma abierta (QDD Schema y QDD Protocol), asegura que el framework no dependa de ninguna herramienta comercial específica.

- **Formatos Universales**: Dicta cómo se estructuran las *Certifications*, *Findings*, *ADRs* y *Sprints* utilizando estándares YAML y JSON rígidos.
- **Protocolo de Interacción**: Define los contratos que deben respetar los IDEs y los LLMs para leer el conocimiento y proponer cambios. Esto garantiza que una regla escrita hoy será entendida por los modelos de IA del mañana.

## Engine (QDD Runtime & MCP Server)

El motor de QDD, compilado en Go para máximo rendimiento y distribución, opera como el cerebro certificador y Servidor MCP:

1. **Fast Path (Determinista)**: 
   Un motor rápido y local que ejecuta validaciones estrictas y seguras sin necesidad de invocar Inteligencia Artificial. Comandos como `qdd audit`, `qdd certify` y `qdd modules` viven aquí. Son inmediatos, reproducibles offline y actúan como el verdadero "juez" de calidad: `qdd audit` corre las pruebas del proyecto y bloquea si fallan, si no hay pruebas o si un pipeline despliega sin pasarlas.

2. **Servidor MCP (Model Context Protocol)**: 
   Cuando pides a tu IA (ej. Antigravity) que trabaje en el repositorio, la orden entra en un ecosistema guiado por las herramientas MCP expuestas por QDD:
   - **Gatekeeper**: Las tools abortan si el código incumple el `config.yaml`.
   - **Context Provider**: `qdd_project_context` entrega las fichas de módulo, los blueprints de la organización, el *Intelligence Report* y las certificaciones a la IA.
   - **Risk Analyzer**: Impide que la IA proponga cambios que rompan contratos públicos.
   - **Consultative Node**: Impone el uso de estándares (ej. OpenAPI) exigiendo a la IA pedir aprobación antes de escribir código base.

## Artifacts (Gestión del Conocimiento)

QDD trata el conocimiento como el activo más valioso. Los artefactos son los "documentos vivos" que persisten este conocimiento dentro de `.qdd/`:

- **Findings (`.qdd/project/findings/`)**: Todo bug encontrado se documenta aquí. Describe la causa raíz, la evidencia y la prueba asociada. Evita que un bug resuelto vuelva a ocurrir.
- **ADRs (`.qdd/project/adr/`)**: Architecture Decision Records. Congelan en el tiempo el "por qué" se tomó una decisión (ej. "Por qué usamos gRPC en lugar de REST").
- **Sprints (`.qdd/project/sprints/`)**: Archivos Markdown que orquestan el trabajo iterativo, definiendo el alcance, los criterios de aceptación y las certificaciones requeridas para el ciclo.
- **Fichas de módulo (`.qdd/project/modules/`)**: El objetivo operativo, el objetivo tecnológico, las invariantes y las pruebas de cada módulo.
- **Blueprints (`.qdd/blueprints/`, `.qdd/blueprints.lock.yaml`, `.qdd/project/blueprints/`)**: La copia fijada de los estándares de la organización y las decisiones que el proyecto tomó al aplicarlos. Sus escenarios de conformidad van en `.qdd/project/goldensets/<modulo>/conformidad.yaml`.

> **Regla de oro (CERT-033):** `.qdd/project/` nunca contiene código de aplicación — solo el conocimiento que QDD acumula *sobre* ese código. El código real del proyecto vive siempre en la raíz del repo, en una carpeta de código explícita (`src/`, o la convención idiomática ya existente como `cmd/`+`pkg/` en Go). `qdd init` debe detectar esa convención o proponerla en Modo Consultivo — nunca improvisar.

## Plugins (Extensibilidad)

La arquitectura de **Plugins** permite a QDD integrarse nativamente con cualquier lenguaje o framework, sin alterar el binario principal.

- Mediante inyección de dependencias o subprocesos (RPC), los plugins proveen lógicas específicas de escaneo.
- Ejemplo: Un plugin de "Go" enseñará a `qdd audit` a buscar goroutines mal gestionadas, mientras que un plugin de "Node" auditará el árbol de dependencias de npm.
- Los plugins del core (`.qdd/core/plugins/`, protocolo en `PLUGIN_PROTOCOL.md`) se compilan dentro del binario público. Hoy son `qdd-human-certifier` (certificación de comportamiento humano y WCAG 2.2, `qdd certify human`) y `qdd-antigravity-tools` (ejecución autónoma de herramientas con un agente local).
- Un plugin trae **código** genérico. Los estándares propios de una organización no van en un plugin, sino en un [blueprint](blueprints.md), que vive en el repositorio de esa organización.

## Protocolo MCP (Model Context Protocol)

Para cumplir la directiva de **Independencia Tecnológica**, QDD utiliza el **Protocolo MCP**.

- Al usar MCP, el framework se aísla completamente de un modelo de IA específico (Gemini, Claude, ChatGPT).
- El *Engine* aplica la misma Gobernanza y el mismo Wisdom Registry; simplemente expone un conjunto de *Tools* estandarizadas que cualquier IDE moderno puede consumir.
- Esta capa es la encargada de obligar al modelo (ej. Claude Code, Cursor) a operar en *Modo Consultivo* en lugar de escribir código a ciegas.
