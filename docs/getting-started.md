# Getting Started con QDD

Bienvenido a Quality-Driven Development. QDD no es un asistente de programación, es una **Plataforma de Ingeniería de Software basada en Mejora Continua**. Esta guía te enseñará paso a paso cómo inicializar tu proyecto y gobernar su ciclo de vida bajo los estándares de producción más altos.

## Paso 1: Inicialización
Ejecuta el comando base para que QDD instale sus estructuras de gobernanza (`.qdd/config.yaml`, `state.json`) y el **Wisdom Registry** (el Manifiesto QDD y los principios de ingeniería base) en tu repositorio.
```bash
qdd init
```

## Paso 2: Aprendizaje y Contexto (Intelligence Report)
QDD no opera a ciegas. `qdd_learn` es una *tool* MCP (no un comando de terminal): conecta tu IDE con IA (Claude Code, Cursor, Antigravity) por MCP y pídele que la ejecute para que escanee tu arquitectura, lenguajes y documentación existente, y genere/refine el **Intelligence Report** en `.qdd/understanding.json` — la memoria central del proyecto, que se actualiza incrementalmente en cada corrida.
```text
Pídele a tu IDE: "ejecuta qdd_learn" (o equivalente en tu asistente)
```

## Paso 3: Análisis y Calidad Inicial
Mide el estado actual de tu deuda técnica. `dashboard`, `audit` y `evolution` son comandos reales de terminal; `status` y `score` son *tools* MCP (pídeselas a tu IA).
```bash
qdd dashboard   # Inicia el Centro de Comando Web para visualizar el Intelligence Report y métricas
qdd audit       # Ejecuta el Linter de reglas estructurales
qdd evolution   # Recomienda la siguiente mejora estudiando findings, certs e historial
```

## Paso 4: Programación Guiada con MCP
Cuando estés listo para desarrollar, delega el trabajo a tu IDE asistido por IA (Cursor, Claude Code, Antigravity). QDD opera como un **Servidor MCP** que inyecta todo el contexto, reglas y herramientas de certificación directamente en el cerebro de la IA.

Si tu IDE intenta desarrollar una característica que requiere seguridad, QDD (actuando como Gatekeeper a través de MCP) *no permitirá un prototipo rápido*. Rechazará el código deficiente y propondrá adoptar un estándar (como OWASP o OpenAPI), exigiendo autorización para garantizar una ingeniería "Production-First".

```bash
# Ejemplo: Pide a tu IDE (Claude/Cursor) que ejecute herramientas QDD:
/qdd "agrega un endpoint de login y certifícalo"
/qdd "resuelve el bug de conexión y genera el finding"
```

## Paso 5: Versionado y Calidad Evolutiva
Una vez completado tu ciclo iterativo (Sprint) sin haber roto certificaciones o contratos públicos, empaqueta una versión segura. `release` también es una *tool* MCP (`qdd_release`): compila, actualiza `state.json` y crea un Git Tag local — no hace push ni publica por sí sola.
```text
Pídele a tu IDE: "ejecuta qdd_release con la versión v1.0.0"
```
