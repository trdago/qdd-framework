# Command Reference

La interfaz de línea de comandos de QDD (QDD CLI) se divide en dos rutas:
1. **Fast Path**: Comandos deterministas que se ejecutan localmente sin IA.
2. **Cognitive Path (QCL)**: El motor conversacional para crear o arreglar código.

## The Safe Boundary: Análisis vs Mutación

En QDD existe una línea estricta que separa **leer/auditar** de **modificar el código**. Los comandos de auditoría están diseñados para ser 100% seguros (Read-Only) y jamás alterarán tu código fuente.

### 🛡️ Comandos Seguros (Read-Only / Auditoría)
Estos comandos puedes ejecutarlos sin miedo. Su único trabajo es leer tu repositorio y reportar su estado:

| Comando | Descripción |
|---------|-------------|
| `qdd learn` | Explora el código base para inyectar lenguajes y arquitecturas al `config.yaml`. **Seguro.** |
| `qdd status` | Panel de control. Escanea el repositorio para mostrar certificaciones activas y *Findings* (bugs) abiertos. **Seguro.** |
| `qdd score` | Calcula tu calificación de calidad matemática (Ej: 100/100 World-Class). **Seguro.** |
| `qdd audit` | Ejecuta un Linter estático asegurando las reglas del framework (ej. Cero uso de `else`). **Seguro.** |
| `qdd certify` | Revisa la carpeta `.qdd/certification/` y emite un veredicto de calidad del proyecto. **Seguro.** |
| `qdd dashboard` | Inicia el Centro de Comando Web. Despliega un panel interactivo (Drill-down) para inspeccionar métricas, Sprints, Findings y Certificaciones en tiempo real, operando con aislamiento de puertos. **Seguro.** |

### ⚡ Comandos de Mutación (Estructurales)
Estos comandos modifican el repositorio agregando carpetas o archivos de gobernanza:

| Comando | Descripción |
|---------|-------------|
| `qdd init` | Inicializa el entorno creando el directorio `.qdd/`, `config.yaml` y `state.json`. |
| `qdd sprint <n>` | Crea la plantilla de trabajo para una nueva iteración modificando `.qdd/sprints/`. |
| `qdd release <version>` | Genera un Git Tag oficial y actualiza la versión del framework en `state.json`. |
| `qdd sync` / `qdd sync-ai` | Sincroniza las reglas nativas (QDD Protocol) con los asistentes de IA (Cursor, Claude Code, Antigravity) configurando los Slash Commands de manera idempotente sin usar prompt injections. |

## Cognitive Path (Pipeline Inteligente)

Para invocar a la Inteligencia Artificial, simplemente escribe tu intención:

```bash
qdd "agrega autenticación a la API"
qdd "resuelve la deuda técnica en el validador"
```

### 🧠 Capacidades del QCL (QDD Cognitive Layer)
- **Guardián (Gatekeeper)**: El CLI no te permitirá hacer cambios ciegos. Si falta conocimiento esencial en `config.yaml`, abortará la misión hasta que ejecutes `qdd learn`.
- **Detección de Intención**: Diferencia si quieres hacer un *Feature*, un *Fix*, o un *Ask*.
- **Resolución de Ambigüedades**: Si tu orden es muy vaga (ej. `qdd "agrega algo"`), pausará el flujo y te mostrará opciones interactivas en tu consola.
- **Análisis de Riesgo**: Antes de programar, evaluará si tu petición romperá la retrocompatibilidad.
- **Estrategia (Strategy Planner)**: Diseñará qué artefactos crear antes de tocar el código (ej. Findings, ADRs).
