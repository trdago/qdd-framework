# QDD Framework — Quality-Driven Development

QDD no es un generador de código.
QDD no es un asistente de programación.
QDD no es un agente conversacional.

**QDD es una Plataforma de Ingeniería de Software basada en Mejora Continua**, cuyo propósito es ayudar a construir y evolucionar productos de software preparados para producción siguiendo estándares de clase mundial.

La Inteligencia Artificial es solamente uno de sus motores. El verdadero propósito de QDD es preservar, incrementar y gobernar el conocimiento del proyecto durante todo su ciclo de vida.

---

## 🎯 Problema que buscamos resolver

Actualmente las herramientas de IA generan código, sin embargo:
- No preservan el conocimiento del sistema.
- Generan prototipos y maquetas desechables sin estándares de producción.
- No convierten los bugs en conocimiento permanente.
- No mantienen una certificación viva del software.
- Sacrifican la calidad y mantenibilidad a largo plazo por velocidad inmediata.

QDD resuelve este problema operando bajo el **Manifiesto QDD** y el principio fundamental de **Production-First Engineering**.

---

## 🧠 Filosofía y Principios Fundamentales

- **Production-First:** QDD no genera ejemplos desechables. Toda solución se diseña pensando en su operación a largo plazo y debe poder desplegarse en producción con calidad profesional.
- **El conocimiento es el activo principal:** El código no es la fuente de verdad, la fuente de verdad es el conocimiento acumulado (Certificaciones, ADRs, Findings).
- **Modo Consultivo y Certificación:** QDD actúa como un Arquitecto Principal. Nunca escribirá código deficiente ciegamente. Si detecta la oportunidad de implementar un estándar (ej. OpenAPI, OWASP, Clean Architecture), detendrá la ejecución, entrará en Modo Consultivo, explicará los beneficios y pedirá autorización.
- **Findings Become Knowledge:** Cada bug descubierto se transforma automáticamente en conocimiento permanente, generando nuevas pruebas, certificaciones y evidencias.
- **Backward Compatibility First:** Nunca se rompen contratos automáticamente. Si hay que modificar APIs o contratos públicos, el framework solicita autorización explícita.

---

## Instalación

### Opción 1: NPM (Recomendado para Web Devs)
```bash
npm install -g qdd-framework
```

### Opción 2: Compilación Manual (Go)
```bash
go install github.com/trdago/qdd-framework/cli@latest
```

---

## 🏗️ Arquitectura General

QDD está compuesto por:
- **Wisdom Registry:** La mente y constitución del proyecto (`.qdd/wisdom/`).
- **Specification:** Definiciones independientes del modelo de IA.
- **Runtime:** El núcleo de ejecución y certificaciones.
- **CLI:** Herramientas de análisis, auditoría y comandos IA.
- **Dashboard:** Centro de Comando Web que despliega el *Intelligence Report*.
- **AI Adapters:** Integraciones independientes (Gemini, Claude, ChatGPT, Cursor, etc.).

---

## 💻 CLI (Command Line Interface)

QDD incluye una CLI escrita en Go dividida en dos vías de ejecución:

**Fast Path Commands (Seguros y Deterministas):**
- `qdd init`: Detecta el stack tecnológico y prepara el Entorno Base y el Wisdom Registry.
- `qdd learn`: Inyecta la arquitectura al Motor Cognitivo generando el **Intelligence Report**.
- `qdd certify`: Ejecuta las certificaciones y garantiza la calidad del código.
- `qdd audit`: Un linter interno que escanea la base de código buscando violaciones de reglas.
- `qdd status`: Muestra un panel de control con el estado de certificaciones y bugs (Findings) abiertos.
- `qdd score`: Calcula automáticamente tu grado de calidad (World-Class, A, B, C, D) en base a la deuda técnica.
- `qdd sprint <n>`: Inicializa el ciclo iterativo de trabajo.
- `qdd release <version>`: Empaqueta una versión lista para producción.
- `qdd dashboard`: Inicia el Centro de Comando Web interactivo (Frontend embebido).
- `qdd sync`: Sincroniza las reglas de gobernanza con tus asistentes de IA.

**Cognitive Path Commands (Motor QCL):**
- `qdd "cualquier intención libre"`: El pipeline cognitivo evaluará la intención. Al estar regido por el *Manifiesto*, actuará en **Modo Consultivo** para proponer certificaciones industriales y mejoras arquitectónicas antes de alterar el código.

---

## 📦 Repositorio y Estructura

- `specification/`: Definiciones independientes del modelo de IA.
- `cli/`: Código de la interfaz de línea de comandos.
- `runtime/`: El núcleo del framework.
- `rfcs/`: Request for Comments.

---

## 🌍 Estándares Internacionales (Compliance)

QDD predica con el ejemplo. Nuestro Dashboard Web está desarrollado bajo:
- **WCAG Nivel AA (W3C)**: Interfaz 100% accesible (POUR).
- **ISO/IEC 25010**: Diseñado siguiendo los pilares de Usabilidad, Seguridad y Portabilidad.

---

## 🤝 Contribuyendo y RFCs

QDD es un estándar abierto enfocado en gobernar el desarrollo de software seguro y auditable. Lee `CONTRIBUTING.md` para más información.
