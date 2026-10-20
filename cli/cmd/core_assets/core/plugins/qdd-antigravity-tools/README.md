# QDD Antigravity Tools Plugin ⚡📡🤖

Plugin oficial de **QDD Framework** que implementa la conexión autónoma, asíncrona y sin bucles de sondeo (Zero-Polling) con agentes de **Google Antigravity**, mediante su SDK oficial (`google.antigravity`), adaptadores MCP y el protocolo reactivo **Ping-Pong**.

---

## 🌟 Características Principales

1. **Protocolo Ping-Pong (Zero-Polling Handshake):**
   - Elimina los anti-patrones de espera activa (`while True: sleep()`).
   - Sincronización determinista mediante el modelo de eventos y *Reactive Wakeup* de Antigravity.

2. **Doble Runtime (Python & Node.js):**
   - **Python:** `AntigravityPingPongClient`, `QDDToolBridge`, y tipos fuertemente tipados (`PingRequest`, `PongResponse`).
   - **Node.js:** `AntigravityPingPongBridge` con emisor de eventos en tiempo real (`ping_started`, `token`, `pong_received`).

3. **Streaming de Pensamientos y Herramientas:**
   - Interceptación y consumo de razonamiento interno (`response.thoughts`) y llamadas a herramientas (`response.tool_calls`).

4. **Prompt Maestro Paso a Paso:**
   - Plantilla de ingeniería de prompts optimizada para que los agentes operen bajo los estándares del **Manifiesto QDD**.

---

## 🚀 Uso Rápido en Python

```python
import asyncio
from runtime.python.ping_pong_client import AntigravityPingPongClient

async def main():
    client = AntigravityPingPongClient(
        system_instructions="Eres un asistente de arquitectura gobernado por QDD."
    )

    async with client.session() as session:
        pong = await session.ping(
            prompt="Audita la arquitectura del proyecto y valida el estándar Zero-Else.",
            on_token=lambda tok: print(tok, end="", flush=True)
        )
        print(f"\n[PONG ACK] Status: {pong.status} ({pong.duration_ms:.1f}ms)")

if __name__ == "__main__":
    asyncio.run(main())
```

---

## 🚀 Uso Rápido en Node.js

```javascript
import { AntigravityPingPongBridge } from './runtime/node/ping_pong_bridge.mjs';

const bridge = new AntigravityPingPongBridge();
bridge.on('token', (tok) => process.stdout.write(tok));

const pong = await bridge.ping('Ejecuta la certificación del sistema');
console.log(`\nStatus: ${pong.status}`);
```

---

## 📜 Certificación QDD
- **Zero-Else:** 100% verificado (`0` cláusulas `else`).
- **Tests Unitarios:** 6/6 en verde (Node.js & Python).
