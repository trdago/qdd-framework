# Especificación Técnica del Protocolo Ping-Pong en Antigravity

## 1. El Problema del Polling Tradicional
En arquitecturas de agentes tradicionales, los scripts externos suelen implementar bucles bloqueantes:
```python
# ❌ ANTI-PATRÓN PROHIBIDO (POLLING CIEGO)
while True:
    status = check_agent_status()
    if status == "DONE":
        break
    time.sleep(5)  # Desperdicio de CPU y latencia impredecible
```

Este anti-patrón genera:
- Desperdicio de recursos y saturación de API.
- Retardos artificiales (la respuesta estuvo lista a los 100ms pero se espera al siguiente ciclo de 5s).
- Riesgo de bloqueos silenciosos o timeouts no controlados.

---

## 2. El Modelo Reactivo de Antigravity (Event-Driven Ping-Pong)

Antigravity opera bajo un modelo de **Reactive Wakeup**:
1. El llamador envía un **Ping** con el contexto de la tarea.
2. La API de Antigravity abre un canal de streaming asíncrono no-bloqueante (`AsyncIterator`).
3. Conforme el modelo genera razonamiento interno (`thoughts`) y llamadas a herramientas (`tool_calls`), los eventos se despachan inmediatamente al listener.
4. Al finalizar la generación o requerir interacción, se emite el evento **Pong** con el payload de respuesta final.

---

## 3. Estructura de Mensajes Ping-Pong

### A. Payload del Ping (Inbound)
```json
{
  "protocol": "qdd-ping-pong/v1",
  "message_type": "PING",
  "session_id": "sess_98234a_b1",
  "turn_id": 1,
  "timestamp": "2026-08-31T08:30:00Z",
  "prompt": "Ejecuta la certificación de accesibilidad en el portal.",
  "context": {
    "workspace_root": "/path/to/project",
    "qdd_rules": ["zero-else", "early-return"]
  },
  "options": {
    "stream_thoughts": true,
    "stream_tools": true,
    "timeout_ms": 60000
  }
}
```

### B. Payload del Pong (Outbound / Ack)
```json
{
  "protocol": "qdd-ping-pong/v1",
  "message_type": "PONG",
  "session_id": "sess_98234a_b1",
  "turn_id": 1,
  "timestamp": "2026-08-31T08:30:04Z",
  "status": "SUCCESS",
  "duration_ms": 4120,
  "output": {
    "content": "Certificación completada con éxito. Score WCAG: 98/100.",
    "tools_executed": [
      { "name": "run_command", "status": "COMPLETED", "exit_code": 0 }
    ],
    "artifacts_created": ["accessibility_report.md"]
  }
}
```

---

## 4. Handshake de Reintentos y Resiliencia
- Si ocurre una desconexión transitoria durante el streaming, el cliente utiliza el `turn_id` y `session_id` para reanudar el consumo del stream sin re-ejecutar herramientas de forma duplicada (Idempotencia).
