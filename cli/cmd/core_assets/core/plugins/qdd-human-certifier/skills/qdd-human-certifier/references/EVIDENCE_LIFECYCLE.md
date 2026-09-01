# Ciclo de Vida y Estructura de Evidencia por Ejecución 📁📊

En el framework QDD, **cada ejecución individual de certificación humana produce una carpeta única e inmutable de evidencias**.

---

## 1. 📂 ESTRUCTURA DE CARPETA POR EJECUCIÓN

Ruta estándar:
```text
.qdd/evidence/run_<YYYYMMDD_HHMMSS>_<RUN_ID>/
```

### Contenido Detallado del Directorio:

```text
run_20260831_220512_a8b9c/
├── manifest.json            # Metadatos del run y resumen estructural
├── screenshots/             # Capturas visuales por paso y por viewport
│   ├── step_01_login_desktop.png
│   ├── step_01_login_mobile.png
│   ├── step_02_create_entity_desktop.png
│   └── step_03_grid_validation_desktop.png
├── traces/                  # Archivos .zip de Playwright Trace
│   ├── trace_desktop.zip
│   └── trace_mobile.zip
├── db_snapshots/            # Volcados de estado en BD vinculados al TRACE_ID
│   ├── 00_pre_execution.json
│   ├── 01_post_creation.json
│   └── 02_final_state.json
├── network_logs/            # Auditoría HTTP/WS y respuestas de APIs
│   ├── http_traffic.har
│   └── errors_4xx_5xx.log
└── backend_logs/            # Logs del servidor y de base de datos
    └── app_output.log
```

---

## 2. 📋 ESQUEMA DEL MANIFEST (`manifest.json`)

```json
{
  "version": "1.0.0",
  "runId": "run_20260831_220512_a8b9c",
  "timestamp": "2026-08-31T22:05:12Z",
  "traceToken": "QDD_TR_8A9B2C",
  "target": "auth:login_nominal,settings:entity_crud",
  "viewports": [
    { "name": "desktop", "width": 1440, "height": 900 },
    { "name": "mobile", "width": 390, "height": 844 }
  ],
  "dbSnapshotsCount": 3,
  "screenshotsCount": 6,
  "executionMetrics": {
    "totalDurationMs": 4350,
    "cognitiveTTA": {
      "loginNominalMs": 1240,
      "entityCreationMs": 2100
    },
    "rageClicksDetected": 0
  },
  "llmVerification": {
    "evaluatedBy": "LLM_CERTIFIER_AGENT",
    "verdict": "PASS",
    "confidence": 0.98,
    "notes": "Persistencia en base de datos validada con registro ID QDD_TR_8A9B2C. Sin solapamientos visuales ni excepciones silenciosas."
  }
}
```

---

## 3. 🧠 PROTOCOLO DE INSPECCIÓN DEL LLM

El LLM que ejecuta la prueba debe seguir este checklist estricto antes de emitir la certificación:

1. **Lectura del `manifest.json`:** Verificar que todos los nodos del grafo se ejecutaron.
2. **Revisión de `screenshots/`:**
   - Comprobar que no existan barras de desplazamiento rotas o elementos fuera de pantalla en Mobile (`390x844`).
   - Comprobar que los botones de acción principal sean visibles sin necesidad de scroll excesivo.
   - Comprobar que los mensajes de éxito/confirmación aparezcan con contraste accesible.
3. **Revisión de `db_snapshots/`:**
   - Confirmar que el `traceToken` se encuentra en las tablas correspondientes con los valores exactos.
   - Confirmar que no hay registros huérfanos o transacciones a medio completar.
4. **Revisión de `network_logs/`:**
   - Asegurar que no existan respuestas `500 Internal Server Error`, `502 Bad Gateway` o `404 Not Found` no intencionadas.
5. **Decisión y Redacción:**
   - Si todo es conforme, emitir `PASS`.
   - Si hay una discrepancia visual o en base de datos, emitir `FAIL`, registrar el `Finding` detallado y generar el test unitario de regresión.
