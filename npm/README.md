# QDD Framework — Quality-Driven Development

[![NPM Version](https://img.shields.io/npm/v/qdd-framework.svg)](https://www.npmjs.com/package/qdd-framework)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

**QDD (Quality-Driven Development)** es una Plataforma de Ingeniería de Software basada en Mejora Continua y Gobernanza de Código desde el Día Cero. Asegura la calidad de producción eliminando prompts manuales mediante un Framework (Wisdom Registry), certificaciones 'Rules-as-Code', auditoría técnica automatizada (9 pilares de calidad), equipo resolutor multi-agente nivel FAANG (`/qdd resolve`), soporte completo para estándares internacionales de Inteligencia Artificial (ISO 42001, ISO 29119-11, SOC 2, ISO 23894), detección de APIs locales de LLM (Antigravity, Ollama, LM Studio) y un sistema unificado de reportes y dashboard en tiempo real.

---

## 🚀 Instalación Rápida

```bash
npm install -g qdd-framework
```

> **Conflicto con el paquete `qdd` de npm:** existe un paquete ajeno llamado `qdd` (no pertenece a QDD Framework) que también instala un comando `qdd`. Si lo tienes instalado globalmente, desinstálalo antes (`npm uninstall -g qdd`) para que `qdd` apunte a QDD Framework. Desde v1.9.22 su presencia ya no rompe la instalación (antes fallaba con `EISDIR`).

---

## ⚡ Comandos Principales

```bash
cd mi-proyecto
qdd init      # detecta el stack (también monorepos) y conecta tu IA
qdd audit     # corre las pruebas y las reglas de QDD; funciona desde cualquier subcarpeta
```

- `qdd init`: Inicializa QDD en el proyecto, detecta el stack (Node, Go, Java/Kotlin, Python, Dart/Flutter, Rust, PHP, Ruby, .NET) y conecta tu IA (Claude Code, Cursor, Antigravity, OpenCode) por MCP. En un proyecto sin código: `qdd init --lenguaje Go --arquitectura "API REST"`.
- `qdd audit`: Corre las pruebas del proyecto y revisa las reglas de QDD (Zero-Else, OWASP, Clean Code, despliegues sin pruebas, fichas y blueprints). Separa lo que bloquea de las recomendaciones (adopción gradual).
- `qdd certify`: Audita, calcula el puntaje y lo guarda en el historial de calidad. `qdd certify human` certifica usabilidad y accesibilidad (WCAG 2.2).
- `qdd modules`: Fichas de módulo (objetivo, invariantes y pruebas de cada módulo). `qdd modules approve <id>` las aprueba; `qdd modules context <rutas>` muestra lo que recibe la IA.
- `qdd blueprint add | apply | list | show`: Importa los estándares de módulo de tu organización desde su propio repositorio y genera la ficha de cada módulo.
- `qdd doctor`: Diagnóstico determinista de salud del entorno.
- `qdd report`: Reporte ejecutivo consolidado en formato Texto, Markdown o JSON para CI/CD.
- `qdd evolution`: Recomienda la siguiente mejora del proyecto.
- `qdd dashboard`: Centro de Comando Web interactivo con streaming en tiempo real (SSE) y ejecución de agentes.
- `qdd run`: Pipeline secuencial de comandos con rollback, o modo supervisor con auto-reparación (`qdd run --keep-alive`).

---

## 🤖 Integración con Inteligencia Artificial (MCP Server)

`qdd init` registra el servidor MCP de QDD y escribe sus reglas para **Claude Code**, **Cursor**, **Google Antigravity** y **OpenCode** (`AGENTS.md`). Con eso, tu IA:

- llama a `qdd_project_context` antes de cambiar código y respeta la ficha y las invariantes del módulo;
- aplica los blueprints de tu organización y te hace a ti sus preguntas de diseño;
- no commitea ni despliega sin pasar `qdd_audit`.

Pídeselo en lenguaje natural:

```text
/qdd "agrega el blueprint de notificaciones y construye el módulo"
/qdd "resuelve el bug de conexión y genera el finding"
```

Todos los comandos y tools: [Command Reference](https://github.com/trdago/qdd-framework/blob/main/docs/command-reference.md) · [Blueprints](https://github.com/trdago/qdd-framework/blob/main/docs/blueprints.md) · [Getting Started](https://github.com/trdago/qdd-framework/blob/main/docs/getting-started.md)

---

## 🛡️ Reglas Globales de Codificación
1. **Cero-Else:** QDD prohíbe el uso de `else`, exigiendo cláusulas de guarda y retornos tempranos (*Early Returns*).
2. **Salida Más Rápida Primero:** Los errores y validaciones se manejan al inicio de cada función.
3. **Aprendizaje Perpetuo:** Todo bug encontrado se documenta como Finding y requiere un test unitario obligatorio.

---

## 📄 Licencia
MIT © QDD Framework Team
