# QDD — Quality-Driven Development

[![npm](https://img.shields.io/npm/v/qdd-framework?label=latest)](https://www.npmjs.com/package/qdd-framework)
[![preview](https://img.shields.io/npm/v/qdd-framework/preview?label=preview)](https://www.npmjs.com/package/qdd-framework?activeTab=versions)
[![Licencia: MIT](https://img.shields.io/badge/licencia-MIT-blue.svg)](https://opensource.org/licenses/MIT)

**QDD hace que una IA que programa en tu proyecto trabaje con tu método, y te deja ver y decidir lo que importa.** Es un CLI
en Go, un servidor MCP y un panel, para Claude Code, Cursor, Antigravity y OpenCode.

## Instalar

```bash
npm install -g qdd-framework --allow-scripts=qdd-framework
```

npm 12 no corre los scripts de instalación sin esa bandera. Si instalas sin ella, la primera vez que uses `qdd` se baja su
binario desde el release de GitHub de esa versión. Lo más nuevo está en el canal preview:
`npm install -g qdd-framework@preview --allow-scripts=qdd-framework`.

Hay binarios para Linux y macOS (amd64 y arm64) y para Windows (amd64).

> **No lo confundas con el paquete `qdd`.** Es ajeno y también instala un comando `qdd`. Si lo tienes instalado globalmente,
> desinstálalo antes: `npm uninstall -g qdd`.

## Empezar

```bash
cd mi-proyecto
git init        # si todavía no es un repositorio
qdd init        # detecta el stack, escribe las reglas de tu IA y activa el gate
qdd audit       # pruebas + reglas
qdd consola activar   # el panel, en Claude Code (init ya conectó el servidor MCP)
```

## Qué trae

- **Gate:** `qdd audit` corre las pruebas y las reglas, sin llamar a un modelo. Sale con 1 si algo bloquea.
- **Puertas:** frenan un commit sin auditar, un `fix` sin finding o una publicación hecha por la IA.
- **Fichas y blueprints:** el porqué de cada módulo y los estándares de tu organización, que la IA lee antes de tocar código.
- **Panel y tablero:** qué está hecho con evidencia, qué espera a quién y lo que necesita tu «Sí».
- **Flujos, modo goal, QDD Móvil, baúl de accesos, herramientas y cola de mejoras.**

Las reglas de código leen solo Go; en otros lenguajes el gate corre las pruebas, el formato y los despliegues, y dice qué no
pudo evaluar. El soporte por IA, los límites y lo que usa la red están en el
[README](https://github.com/trdago/qdd-framework#readme).

## Documentación

[Getting Started](https://github.com/trdago/qdd-framework/blob/main/docs/getting-started.md) ·
[Command Reference](https://github.com/trdago/qdd-framework/blob/main/docs/command-reference.md) ·
[Blueprints](https://github.com/trdago/qdd-framework/blob/main/docs/blueprints.md) ·
[CHANGELOG](https://github.com/trdago/qdd-framework/blob/main/CHANGELOG.md)

Licencia MIT.
