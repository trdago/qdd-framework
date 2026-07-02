# QDD Framework — Quality-Driven Development

QDD no es un conjunto de prompts.
QDD tampoco es un agente.
QDD tampoco es un asistente de programación.

**QDD es un AI-Native Software Engineering Framework** cuyo propósito es gobernar el ciclo completo de desarrollo de software utilizando Inteligencia Artificial. 

Nuestro objetivo es crear un estándar abierto que pueda ser utilizado por cualquier modelo de IA (ChatGPT, Claude, Gemini, Copilot, Cursor, modelos locales, etc.) sin depender de uno en particular.

La IA será únicamente el motor de ejecución. Toda la inteligencia del proceso vive dentro del Framework.

## Instalación

### Opción 1: NPM (Recomendado para Web Devs)
```bash
npm install -g qdd-framework
```
> Esto descargará automáticamente el binario nativo súper rápido escrito en Go según tu sistema operativo.

### Opción 2: Compilación Manual (Go)
Si prefieres instalarlo desde el código fuente de Go:
```bash
go install github.com/trdago/qdd-framework/cli@latest
```
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

QDD incluye una CLI escrita en Go dividida en dos vías de ejecución:
- **Fast Path (Determinista)**: Comandos locales para operaciones estructurales y de gobernanza.
- **Cognitive Path (IA)**: Flujo conversacional e inteligente para crear o arreglar código.

**Fast Path Commands:**
- `qdd init`: Detecta el stack tecnológico y prepara el Runtime.
- `qdd learn`: Lee el proyecto, entiende la arquitectura y genera conocimiento inicial.
- `qdd certify`: Ejecuta las certificaciones y garantiza la calidad del código.
- `qdd audit`: Un linter interno que escanea la base de código buscando violaciones de reglas (ej. no usar `else`).
- `qdd status`: Muestra un panel de control con el estado de certificaciones y bugs (Findings) abiertos.
- `qdd score`: Calcula automáticamente tu grado de calidad (World-Class, A, B, C, D) en base a la deuda técnica.
- `qdd sprint <n>`: Prepara la documentación e inicializa el ciclo iterativo de trabajo.
- `qdd release <version>`: Empaqueta una versión (tag y state) lista para producción.

**Cognitive Path Commands (Motor QCL):**
- `qdd "cualquier intención libre"`: El pipeline cognitivo evaluará la intención (Feature, Fix, Ask), evaluará riesgos y generará un Plan de Ejecución Inteligente. Incluye **Resolución Interactiva de Ambigüedad** para guiarte si el prompt es muy genérico.

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

## 🌍 Estándares Internacionales (Compliance)

QDD no solo impone calidad en tu software, sino que predica con el ejemplo. Nuestro Dashboard Web (Centro de Comando) está desarrollado bajo el cumplimiento de normativas globales:
- **WCAG Nivel AA (W3C)**: Interfaz 100% accesible, inyectada con atributos semánticos y ARIA tags para garantizar que sea Perceptible, Operable, Comprensible y Robusta (POUR).
- **ISO/IEC 25010 (Calidad de Software)**: Diseñado siguiendo el pilar de **Usabilidad** (Reconocibilidad y Protección contra errores), así como el de **Seguridad y Portabilidad** (sirviendo un binario 100% embebido que no requiere internet).

---

## 🤝 Contribuyendo y RFCs

QDD es un estándar abierto. Todas las propuestas de arquitectura y diseño importantes deben pasar por un proceso de Request for Comments (RFC) en la carpeta `rfcs/`. Lee `CONTRIBUTING.md` para más información.
