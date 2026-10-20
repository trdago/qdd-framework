# PROTOCOLO DE CONEXIÓN Y GOBERNANZA ANTIGRAVITY (QDD ANTIGRAVITY TOOLS)

## 1. REGLA FUNDAMENTAL DE NO-POLLING (REACTIVE FIRST)
Queda estrictamente prohibido implementar bucles de sondeo ciego (`while True: time.sleep()` o polling repetitivo de endpoints).
Antigravity es una plataforma reactiva:
- Las respuestas se transmiten vía streaming asíncrono (`async for token in response:`).
- Los eventos de herramientas se reciben en tiempo real (`response.tool_calls`).
- La reanudación de turnos es automática al completarse subprocesos o tareas en segundo plano (`Reactive Wakeup`).

## 2. PROTOCOLO PING-PONG DETERMINISTA
Toda comunicación entre un proceso/script externo y un agente de Antigravity debe implementar el patrón **Ping-Pong**:
1. **Ping (Inbound Request):** El script emite una carga estructurada con ID de sesión, objetivo claro, contexto de herramientas y restricciones cognitivas.
2. **Turn Processing:** El agente razona, ejecuta herramientas y reporta estado sin bloquear el hilo principal.
3. **Pong (Outbound Response/Ack):** El agente emite la confirmación de finalización de turno o resultado final, permitiendo que el script receptor reanude su ejecución de forma predecible.

## 3. ZERO-ELSE & EARLY RETURN
- Todo conector, script de streaming y adaptador MCP debe utilizar cláusulas de guardia y retornos tempranos (`return`).
- Queda prohibida la sentencia `else` en cualquier capa de integración.

## 4. GOBERNANZA DE HERRAMIENTAS
Toda herramienta expuesta a Antigravity debe retornar esquemas JSON válidos con tipado estricto y manejo de errores determinista.
