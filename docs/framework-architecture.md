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

#### 1. La Vía Cognitiva (Lenguaje Natural y Modo Consultivo)
Dado que QDD gobierna a la IA, puedes delegarle la creación del estándar directamente al CLI usando lenguaje natural:
```bash
qdd "Agrega un estándar estricto que prohíba usar la librería Axios a favor de fetch nativo en el componente UI"
```
QDD operará en **Modo Consultivo**. Si detecta que falta un estándar de la industria (ej. no estás usando Clean Architecture), te propondrá adoptarlo antes de generar el código. Si lo autorizas, creará automáticamente el artefacto de certificación YAML.

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
Para que tu IDE (Cursor, Claude, Copilot) respete las reglas automáticamente mientras programas, debes sincronizar los estándares:
```bash
qdd sync
```
Este comando compilará todos tus estándares, el Wisdom Registry y el Manifiesto en un archivo `.cursorrules` o `.clauderc`. Así, la filosofía "Production-First" se inyecta directamente en el "cerebro" del modelo de lenguaje en tu IDE.

---
**Resumen del Flujo**: 
Tú dictas la regla (o la IA la propone en Modo Consultivo) ➔ QDD la registra como una Certificación ➔ `qdd sync` la inyecta en la IA ➔ `qdd certify` valida que el código la cumpla.

## Specification

La capa de **Specification** es el conjunto de reglas agnósticas que definen la estructura y comportamiento de QDD. Al estar documentada de forma abierta (QDD Schema y QDD Protocol), asegura que el framework no dependa de ninguna herramienta comercial específica.

- **Formatos Universales**: Dicta cómo se estructuran las *Certifications*, *Findings*, *ADRs* y *Sprints* utilizando estándares YAML y JSON rígidos.
- **Protocolo de Interacción**: Define los contratos que deben respetar los IDEs y los LLMs para leer el conocimiento y proponer cambios. Esto garantiza que una regla escrita hoy será entendida por los modelos de IA del mañana.

## Engine (QDD Runtime & QCL)

El motor de QDD, compilado en Go para máximo rendimiento y distribución, opera sobre dos rieles (Paths) independientes:

1. **Fast Path (Determinista)**: 
   Un motor rápido y local que ejecuta validaciones estrictas y seguras sin necesidad de invocar Inteligencia Artificial. Comandos como `qdd certify`, `qdd audit`, y `qdd score` viven aquí. Son inmediatos, reproducibles offline y actúan como el verdadero "juez" de calidad.

2. **Cognitive Path (QCL - QDD Cognitive Layer)**: 
   Cuando se usa lenguaje natural (ej. `qdd "refactoriza la base de datos"`), la orden entra en un Pipeline Inteligente compuesto por múltiples Nodos:
   - **Gatekeeper**: Aborta la misión si falta entendimiento crítico (`config.yaml`).
   - **Context Analyzer**: Extrae el *Intelligence Report* (`understanding.json`) y las certificaciones.
   - **Risk Analyzer**: Evalúa si la orden puede romper contratos públicos.
   - **Strategy Planner**: Diseña los artefactos (*Findings*, *ADRs*) a crear antes de tocar código.
   - **Consultative Node**: Propone al usuario el estándar aplicable (ej. OpenAPI) antes de implementar.

## Artifacts (Gestión del Conocimiento)

QDD trata el conocimiento como el activo más valioso. Los artefactos son los "documentos vivos" que persisten este conocimiento dentro de `.qdd/`:

- **Findings (`.qdd/project/findings/`)**: Todo bug encontrado se documenta aquí. Describe la causa raíz, la evidencia y la prueba asociada. Evita que un bug resuelto vuelva a ocurrir.
- **ADRs (`.qdd/project/adr/`)**: Architecture Decision Records. Congelan en el tiempo el "por qué" se tomó una decisión (ej. "Por qué usamos gRPC en lugar de REST").
- **Sprints (`.qdd/project/sprints/`)**: Archivos Markdown que orquestan el trabajo iterativo, definiendo el alcance, los criterios de aceptación y las certificaciones requeridas para el ciclo.

## Plugins (Extensibilidad)

La arquitectura de **Plugins** permite a QDD integrarse nativamente con cualquier lenguaje o framework, sin alterar el binario principal.

- Mediante inyección de dependencias o subprocesos (RPC), los plugins proveen lógicas específicas de escaneo.
- Ejemplo: Un plugin de "Go" enseñará a `qdd audit` a buscar goroutines mal gestionadas, mientras que un plugin de "Node" auditará el árbol de dependencias de npm.

## AI Adapters

Para cumplir la directiva de **Independencia Tecnológica**, QDD utiliza **AI Adapters**.

- La interfaz del adaptador aísla completamente al modelo de IA (Gemini, Claude, ChatGPT, modelos locales) del resto del framework.
- Si hoy usas Claude y mañana decides usar Gemini, el *Engine* sigue aplicando la misma Gobernanza y el mismo Wisdom Registry; únicamente cambia el conector que traduce las instrucciones hacia el LLM.
- Esta capa es la encargada de inyectar el *Modo Consultivo* en las peticiones enviadas al modelo.
