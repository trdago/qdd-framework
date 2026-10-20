#!/usr/bin/env python3
"""
Ejemplo Completo de Activación y Protocolo Ping-Pong con Google Antigravity
Ejecución: python3 examples/python_ping_pong_example.py
"""

import asyncio
import os
import sys

# Agregar ruta base al path
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..")))

from runtime.python.ping_pong_client import AntigravityPingPongClient
from runtime.python.tool_bridge import QDDToolBridge

async def main():
    print("==================================================================")
    print("🤖 QDD Antigravity Tools - Demostración de Activación Ping-Pong")
    print("==================================================================\n")

    # 1. Configurar Tool Bridge
    tool_bridge = QDDToolBridge()
    tool_bridge.register_tool(
        name="obtener_estado_proyecto",
        description="Retorna el estado de gobernanza del proyecto",
        parameters_schema={"type": "object", "properties": {}},
        handler=lambda: {"calidad": 100, "reglas": "Zero-Else", "estado": "Auditoría Aprobada"}
    )
    print(f"[+] {len(tool_bridge.list_tools())} herramienta(s) registradas en el Tool Bridge.")

    # 2. Inicializar Cliente
    client = AntigravityPingPongClient(
        system_instructions="Eres un agente de arquitectura QDD. Cumples con Zero-Else y Early Returns."
    )

    # 3. Iniciar Sesión y Enviar PING
    session_id = "sesion_demo_antigravity_001"
    print(f"[+] Iniciando sesión '{session_id}'...")

    async with client.session(session_id) as session:
        print("[+] Enviando PING 1: Petición de auditoría...")
        
        # Callback para streaming de tokens en vivo
        def on_token_stream(tok: str):
            print(tok, end="", flush=True)

        pong1 = await session.ping(
            prompt="Audita la arquitectura del proyecto y valida el cumplimiento de las reglas.",
            on_token=on_token_stream
        )

        if not pong1.is_success():
            print(f"\n[!] Error en PING 1: {pong1.error}")
            return

        print(f"\n[🏓 PONG 1 RECIBIDO] Turno: {pong1.turn_id} | Duración: {pong1.duration_ms:.2f}ms | Status: {pong1.status}\n")

        # 4. Multi-Turno: Enviar PING 2 en la misma sesión
        print("[+] Enviando PING 2: Consulta de seguimiento...")
        pong2 = await session.ping(
            prompt="Confirma que la unicidad del dato (SSOT) está activa.",
            on_token=on_token_stream
        )

        if not pong2.is_success():
            print(f"\n[!] Error en PING 2: {pong2.error}")
            return

        print(f"\n[🏓 PONG 2 RECIBIDO] Turno: {pong2.turn_id} | Duración: {pong2.duration_ms:.2f}ms | Status: {pong2.status}\n")

    print("==================================================================")
    print("✅ Protocolo Ping-Pong completado con éxito sin bucles de polling.")
    print("==================================================================")

if __name__ == "__main__":
    asyncio.run(main())
