# QDD Framework — Quality-Driven Development

QDD no es un conjunto de prompts.
QDD tampoco es un agente.
QDD tampoco es un asistente de programación.

**QDD es un AI-Native Software Engineering Framework** cuyo propósito es gobernar el ciclo completo de desarrollo de software utilizando Inteligencia Artificial. 

Nuestro objetivo es crear un estándar abierto que pueda ser utilizado por cualquier modelo de IA (ChatGPT, Claude, Gemini, Copilot, Cursor, modelos locales, etc.) sin depender de uno en particular.

La IA será únicamente el motor de ejecución. Toda la inteligencia del proceso vive dentro del Framework.

---

## 🎯 Problema que buscamos resolver

Actualmente las herramientas de IA generan código, sin embargo:
- No preservan el conocimiento del sistema.
- No mantienen la documentación sincronizada.
- No convierten los bugs en conocimiento permanente.
- No generan trazabilidad.
- No gobiernan el ciclo de vida completo.
- No protegen automáticamente la compatibilidad de las APIs.
- No mantienen una certificación viva del software.

QDD pretende resolver ese problema, ayudando a incorporar IA de forma segura sobre proyectos existentes (Legacy Software).

---

## 🧠 Filosofía y Principios

- **La calidad es el punto de partida:** Toda funcionalidad comienza con una certificación. El código únicamente implementa la certificación.
- **Certification First:** Nunca se comienza escribiendo código.
- **Findings Become Knowledge:** Cada bug encontrado genera un Finding. Todo Finding genera nuevas pruebas y nueva documentación.
- **Evidence of Resolution:** Un bug nunca se considera resuelto únicamente porque el código cambió; debe existir evidencia objetiva (logs, pruebas, métricas, etc.).
- **Continuous Certification:** La certificación nunca termina. Evoluciona con el sistema.
- **Backward Compatibility First:** Nunca se rompen contratos automáticamente. Si hay que modificar APIs o contratos públicos, el framework se detiene y solicita autorización.

---

## 🏗️ Arquitectura General

QDD está compuesto por:
- Specification
- Runtime
- CLI
- Templates
- Schemas
- Plugins
- Documentation
- Website
- SDK
- AI Adapters

---

## 💻 CLI (Command Line Interface)

QDD incluye una CLI escrita en Go con comandos como:
- `qdd init`: Detecta el stack tecnológico y prepara el Runtime.
- `qdd learn`: Lee el proyecto, entiende la arquitectura y genera conocimiento inicial sin modificar código.
- `qdd discover`: Construye el mapa completo del sistema.
- `qdd baseline`: Genera la certificación, findings, ADR, roadmap y estado inicial.
- `qdd audit`: Realiza una auditoría completa descubriendo bugs, riesgos y deuda técnica.
- `qdd feature`: Genera una nueva funcionalidad creando primero la certificación.
- `qdd fix`: Resuelve findings (un commit por finding) generando nueva evidencia.
- `qdd sprint`: Ejecuta una iteración de mejora continua.
- `qdd release`
- `qdd doctor`: Entrega el estado general del proyecto (Quality Score, Coverage, etc.).
- `qdd ask`: Permite consultar el conocimiento utilizando únicamente la especificación y documentación (sin inventar).

---

## 📦 Repositorio y Estructura

El repositorio sigue un modelo de comunidad abierta inspirado en Rust, Kubernetes o Terraform:

- `specification/`: Definiciones independientes del modelo de IA (Lifecycle, Artifacts, Governance).
- `cli/`: Código de la interfaz de línea de comandos.
- `runtime/`: El núcleo del framework.
- `rfcs/`: Request for Comments. Cada decisión importante se documenta y discute aquí antes de implementarse.
- `plugins/`: Extensibilidad para lenguajes y plataformas (Go, Node, Java, AWS, etc.).

---

## 🛣️ Roadmap Inicial

**v0.1**
- Especificación inicial
- CLI mínima
- Runtime
- `qdd init` y `qdd learn`

**v0.5**
- Descubrimiento automático
- Baseline
- Auditoría
- Templates y plugins iniciales

**v1.0**
- Framework estable
- CLI completa
- Website y documentación
- AI Adapters
- Plugins oficiales

---

## 🤝 Contribuyendo y RFCs

QDD es un estándar abierto. Todas las propuestas de arquitectura y diseño importantes deben pasar por un proceso de Request for Comments (RFC) en la carpeta `rfcs/`. Lee `CONTRIBUTING.md` para más información.
