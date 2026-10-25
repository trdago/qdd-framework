"""Valida una sesión MCP: stdout solo puede tener JSON-RPC y cada id esperado
debe responder sin error. Uso: python3 -I mcp_check.py <stdout.log> <id>..."""
import json
import sys

path, ids = sys.argv[1], {int(x) for x in sys.argv[2:]}
problemas, vistos = [], set()
for n, linea in enumerate(open(path, encoding="utf-8", errors="replace"), 1):
    linea = linea.strip()
    if not linea:
        continue
    try:
        msg = json.loads(linea)
    except ValueError:
        problemas.append(f"línea {n} no es JSON: {linea[:120]}")
        continue
    if "id" in msg:
        vistos.add(msg["id"])
    if "error" in msg:
        problemas.append(f"id {msg.get('id')} error: {msg['error']}")
    if msg.get("result", {}).get("isError"):
        problemas.append(f"id {msg.get('id')} isError: {json.dumps(msg['result'])[:200]}")
for falta in sorted(ids - vistos):
    problemas.append(f"sin respuesta para id {falta}")
for p in problemas:
    print("MCP-FALLO:", p)
print(f"respuestas={len(vistos)} problemas={len(problemas)}")
sys.exit(1 if problemas else 0)
