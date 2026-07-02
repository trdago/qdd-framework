# Getting Started con QDD

Bienvenido a Quality-Driven Development. QDD es un AI-Native Software Engineering Framework. Esta guía te enseñará paso a paso cómo inicializar tu proyecto y gobernar su ciclo de vida.

## Paso 1: Inicialización
Ejecuta el comando base para que QDD instale sus estructuras de gobernanza (`.qdd/config.yaml`, `state.json`) en tu repositorio.
```bash
qdd init
```

## Paso 2: Aprendizaje y Contexto
QDD no opera a ciegas. Utiliza el comando seguro `learn` para que el framework escanee tu repositorio y extraiga tu arquitectura, lenguajes, y **toda la documentación existente** (Docs, RFCs, Especificaciones) hacia el `config.yaml`.
```bash
qdd learn
```

## Paso 3: Análisis y Calidad Inicial
Mide el estado actual de tu deuda técnica ejecutando los comandos de **Auditoría (Read-Only)**.
```bash
qdd status  # Ve el panel de control de certificaciones y bugs
qdd score   # Obtén tu calificación matemática (A, B, C, D)
qdd audit   # Ejecuta el Linter de reglas estructurales
```

## Paso 4: Programación Guiada por IA
Cuando estés listo para desarrollar, utiliza el **Cognitive Path**. QDD leerá el contexto, evaluará los riesgos de tu petición, trazará una estrategia y redactará el código por ti garantizando la calidad.
```bash
qdd "agrega un endpoint de login"
qdd "resuelve el bug de conexión a la base de datos"
```

## Paso 5: Versionado
Una vez completado tu ciclo (Sprint), empaqueta una versión segura.
```bash
qdd release v1.0.0
```
