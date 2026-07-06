# Getting Started con QDD

Bienvenido a Quality-Driven Development. QDD no es un asistente de programación, es una **Plataforma de Ingeniería de Software basada en Mejora Continua**. Esta guía te enseñará paso a paso cómo inicializar tu proyecto y gobernar su ciclo de vida bajo los estándares de producción más altos.

## Paso 1: Inicialización
Ejecuta el comando base para que QDD instale sus estructuras de gobernanza (`.qdd/config.yaml`, `state.json`) y el **Wisdom Registry** (el Manifiesto QDD y los principios de ingeniería base) en tu repositorio.
```bash
qdd init
```

## Paso 2: Aprendizaje y Contexto (Intelligence Report)
QDD no opera a ciegas. Utiliza el comando `learn` para que el framework escanee tu arquitectura, lenguajes y documentación existente. Este proceso invoca al Motor Cognitivo para asimilar el negocio y generar el **Intelligence Report**, el cual se utilizará como la memoria central del proyecto.
```bash
qdd learn
```

## Paso 3: Análisis y Calidad Inicial
Mide el estado actual de tu deuda técnica ejecutando los comandos de **Auditoría (Read-Only)**.
```bash
qdd dashboard # Inicia el Centro de Comando Web para visualizar el Intelligence Report y métricas
qdd status  # Ve el panel de control de certificaciones y bugs en la terminal
qdd score   # Obtén tu calificación matemática (A, B, C, D)
qdd audit   # Ejecuta el Linter de reglas estructurales
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
Una vez completado tu ciclo iterativo (Sprint) sin haber roto certificaciones o contratos públicos, empaqueta una versión segura.
```bash
qdd release v1.0.0
```
