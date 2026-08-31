---
name: qdd-human-certifier
description: >-
  Autonomously executes and certifies end-to-end human user journeys, usability, accessibility (WCAG 2.2), and multi-device UX across any web application using a modular Directed Acyclic Graph (DAG) engine. Use when the user requests 'qdd certify human', 'certify human', 'probar comportamiento humano', or asks to validate UI workflows locally before deployment.
---

# QDD Composable Human QA & Usability Graph Engine 🎭🧩📱♿

Este skill permite a los agentes de IA ejecutar de forma autónoma y modular pruebas de comportamiento humano sobre cualquier aplicación web, resolviendo automáticamente las **dependencias de flujo en un Grafo Acíclico Dirigido (DAG)**.

---

## 🧩 ARQUITECTURA DE GRAFO DE PRUEBAS (DAG)

Cada flujo se encapsula en una unidad modular e independiente dentro de `scripts/human_qa/units/` (o la ruta configurada en `.qdd/human_qa.config.json`):

```mermaid
flowchart TD
    A["auth:login_nominal\n(Raíz de Autenticación)"] --> B["dashboard:metrics_table"]
    A --> C["settings:profile_update"]
    A --> U1["usability:rage_clicks_debounce"]
    A --> U2["usability:cognitive_friction_tta"]
    A --> AC1["accessibility:wcag_contrast_audit"]
    A --> AC2["accessibility:layout_drift_zindex"]
    A --> R1["resilience:dual_browser_sync"]
    A --> R2["resilience:network_offline_recovery"]
    B --> D["feature:deep_workflow"]
```

---

## 🚀 MODOS DE EJECUCIÓN

1. **Flujo Específico (con resolución automática de dependencias)**:
   ```bash
   node scripts/human_qa/index.mjs auth:login_nominal
   ```
2. **Categoría Completa**:
   ```bash
   node scripts/human_qa/index.mjs usability
   node scripts/human_qa/index.mjs accessibility
   node scripts/human_qa/index.mjs resilience
   ```
3. **Grafo Completo del Sistema**:
   ```bash
   node scripts/human_qa/index.mjs all
   ```

---

## ⚡ SERVIDOR LOCAL NATIVO ZERO-LATENCY

El motor incluye un servidor HTTP nativo puro (`local_frontend_server.mjs`) que:
- Inicia en menos de **10ms**.
- Sirve los bundles de producción compilados (`dist/`, `build/`, `.next/`, etc.) sin latencia externa.
- Maneja fallback SPA para rutas profundas.

---

## 🛡️ CRITERIOS DE CERTIFICACIÓN OBLIGATORIOS
- **100% PASS** en todas las unidades seleccionadas.
- **Doble Viewport**: Desktop (1440x900) y Mobile (390x844).
- **Cero sentencia `else`** en el código de prueba.
