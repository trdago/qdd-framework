# Getting Started con QDD

Bienvenido a Quality-Driven Development. QDD no es un asistente de programación, es una **Plataforma de Ingeniería de Software basada en Mejora Continua**. Esta guía te lleva desde la instalación hasta tu primer módulo construido con IA bajo las reglas de QDD.

> En cada paso se indica si es un **comando de terminal** o una ***tool* MCP** (algo que le pides a tu IDE con IA). La lista completa está en la [Command Reference](command-reference.md).

## Paso 0: Instalación

```bash
npm install -g qdd-framework
```

Sin Node, descarga el binario de tu sistema desde [Releases](https://github.com/trdago/qdd-framework/releases/latest). Más opciones en el [README](../README.md#instalación).

## Paso 1: Inicialización

En la carpeta de tu proyecto:

```bash
qdd init
```

`qdd init`:

- detecta el stack (Node, Go, Java/Kotlin, Python, Dart/Flutter, Rust, PHP, Ruby, .NET), también en subcarpetas de un monorepo;
- crea `.qdd/` con `config.yaml`, `policies.yaml`, `state.json` y el **Wisdom Registry** (el Manifiesto QDD y los principios de ingeniería);
- conecta tu IA: escribe las reglas y registra el servidor MCP para Claude Code, Cursor, Antigravity y OpenCode (`AGENTS.md`).

En un proyecto que todavía no tiene código, declara el stack:

```bash
qdd init --lenguaje Go --arquitectura "API REST"
```

Es seguro repetirlo: completa lo vacío y no pisa lo que ya configuraste.

## Paso 2: Primera auditoría

```bash
qdd audit
```

`qdd audit` corre las pruebas de tu proyecto y revisa las reglas de QDD. Funciona desde cualquier subcarpeta. Separa dos cosas:

- **Lo que bloquea** (exit 1): pruebas que fallan, no tener pruebas, OWASP, código con `else`, un pipeline que despliega sin pasar las pruebas.
- **Recomendaciones**: en un proyecto nuevo QDD parte en **adopción gradual**, así que E2E, ADR y Conventional Commits se muestran sin bloquear.

Si QDD no encuentra cómo correr tus pruebas, decláralas en `.qdd/config.yaml`:

```yaml
test_commands: ["go test ./..."]
```

Para exigirlo todo desde el día uno: `adopcion: estricta` en `.qdd/policies.yaml`.

```bash
qdd evolution   # te dice cuál es la siguiente mejora que más conviene
qdd dashboard   # Centro de Comando Web con el puntaje y las métricas
```

## Paso 3: Los estándares de tu organización (blueprints)

Si tu organización mantiene un repositorio de blueprints (estándares de módulo como usuarios y roles, notificaciones o auditoría), impórtalos. La primera vez indicas el repositorio; QDD lo recuerda:

```bash
qdd blueprint add git@github.com:org/qdd-blueprints.git --ref v1.0.0 notificaciones
qdd blueprint add --paquete base     # después, solo nombres o paquetes
```

O pídeselo a tu IA: *"agrega el blueprint de notificaciones"*.

Si tu organización aún no tiene uno, sáltate este paso. Cómo crearlo: [Blueprints](blueprints.md).

## Paso 4: Las fichas de módulo

Cada módulo importante tiene una **ficha** (`.qdd/project/modules/<id>.yaml`): para qué existe, sus invariantes y cómo se prueba. Es lo que la IA lee antes de tocarlo.

```bash
qdd modules        # qué fichas hay y qué código todavía no tiene ficha
```

Una ficha nace de dos formas:

- **Desde un blueprint**: `qdd blueprint apply notificaciones --ruta internal/notificaciones`. QDD te hace las preguntas de diseño del blueprint y genera la ficha.
- **Propuesta por la IA**: la IA la escribe con la *tool* `qdd_module_save` basándose en el código y la documentación existentes.

En los dos casos la ficha queda en **borrador** hasta que un humano la revisa:

```bash
qdd modules approve notificaciones
```

## Paso 5: Programar con tu IA

Delega el trabajo a tu IDE con IA (Claude Code, Cursor, Antigravity, OpenCode). QDD opera como un **Servidor MCP** y las reglas que dejó `qdd init` le fijan a la IA una forma de trabajo:

1. Antes de cambiar código, llama a `qdd_project_context` con las rutas que va a tocar y respeta la ficha y sus invariantes.
2. Pruebas primero; cero `else`; cada bug queda como finding con su prueba de regresión.
3. Si el módulo lo define un blueprint, lo aplica y **te hace a ti** sus preguntas de diseño.
4. No commitea ni despliega sin pasar `qdd_audit`.

```bash
# Ejemplo: pídele a tu IDE
/qdd "agrega el envío de notificaciones por correo"
/qdd "resuelve el bug de conexión y genera el finding"
```

Si pides algo que requiere un estándar de la industria (por ejemplo, autenticación), QDD entra en **Modo Consultivo**: te propone el estándar (OWASP, OpenAPI…) y pide tu autorización antes de implementar.

¿Tu IA no tiene MCP? `qdd modules context <rutas>` imprime el mismo contexto para pegárselo.

## Paso 6: Certificar y versionar

```bash
qdd certify     # audita, calcula el puntaje y lo guarda en el historial
qdd report --format md --output reporte.md
```

Cuando el ciclo esté listo, empaqueta la versión con la *tool* MCP `qdd_release`: compila, actualiza `state.json` y crea un Git Tag local. No hace push ni publica por sí sola.

```text
Pídele a tu IDE: "ejecuta qdd_release con la versión v1.0.0"
```
