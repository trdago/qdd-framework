# QDD Framework — Quality-Driven Development

[![NPM Version](https://img.shields.io/npm/v/qdd-framework.svg)](https://www.npmjs.com/package/qdd-framework)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

**QDD (Quality-Driven Development)** es una Plataforma de Ingeniería de Software basada en Mejora Continua y Gobernanza de Código desde el Día Cero. Asegura la calidad de producción eliminando prompts manuales mediante un Framework (Wisdom Registry), certificaciones 'Rules-as-Code', auditoría técnica automatizada (9 pilares de calidad), equipo resolutor multi-agente nivel FAANG (`/qdd resolve`), soporte completo para estándares internacionales de Inteligencia Artificial (ISO 42001, ISO 29119-11, SOC 2, ISO 23894), detección de APIs locales de LLM (Antigravity, Ollama, LM Studio) y un sistema unificado de reportes y dashboard en tiempo real.

---

## 🚀 Instalación Rápida

```bash
npm install -g qdd-framework
```

---

## ⚡ Comandos Principales

- `qdd init`: Inicializa el entorno QDD en el proyecto e inyecta la configuración MCP en IDEs compatibles (Cursor, Claude Code, Antigravity).
- `qdd audit`: Ejecuta la auditoría estática de código sobre los 9 pilares de calidad (Zero-Else, OWASP, Twelve-Factor, AI Governance, Clean Code, etc.).
- `qdd certify`: Valida las certificaciones del proyecto y genera el certificado histórico de calidad.
- `qdd doctor`: Diagnóstico determinista de salud y auto-reparación del entorno.
- `qdd report`: Genera un reporte ejecutivo consolidado en formato Texto, Markdown o JSON para CI/CD.
- `qdd evolution`: Consulta consultiva para determinar la siguiente mejora natural del proyecto.
- `qdd dashboard`: Centro de Comando Web interactivo con streaming en tiempo real (SSE) y ejecución de agentes.
- `qdd run`: Pipeline secuencial de comandos o modo supervisor con auto-reparación (`qdd run --keep-alive`).

---

## 🤖 Integración con Inteligencia Artificial (MCP Server)

QDD expone un servidor Model Context Protocol (MCP) que permite a asistentes como **Google Antigravity**, **Claude Code** y **Cursor** consultar reglas, diagnosticar problemas y ejecutar tareas de forma gobernada:

- `/qdd init`
- `/qdd validate`
- `/qdd certify`
- `/qdd resolve` (Equipo Resolutor FAANG)
- `/qdd report`
- `/qdd evolution`
- `/qdd docs`

---

## 🛡️ Reglas Globales de Codificación
1. **Cero-Else:** QDD prohíbe el uso de `else`, exigiendo cláusulas de guarda y retornos tempranos (*Early Returns*).
2. **Salida Más Rápida Primero:** Los errores y validaciones se manejan al inicio de cada función.
3. **Aprendizaje Perpetuo:** Todo bug encontrado se documenta como Finding y requiere un test unitario obligatorio.

---

## 📄 Licencia
MIT © QDD Framework Team
