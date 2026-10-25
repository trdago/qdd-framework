"""Compara el resumen del recorrido con los fallos esperados.
Uso: python3 -I evaluar.py <resumen.tsv> <esperado.tsv>"""
import sys

resumen, esperado_path = sys.argv[1], sys.argv[2]
esperado = {}
for linea in open(esperado_path, encoding="utf-8"):
    if linea.startswith("#") or not linea.strip():
        continue
    proyecto, paso, motivo = linea.rstrip("\n").split("\t", 2)
    esperado[(proyecto, paso)] = motivo

total = ok = 0
inesperados = []
for n, linea in enumerate(open(resumen, encoding="utf-8")):
    if n == 0:
        continue
    proyecto, paso, codigo, seg, banderas = (linea.rstrip("\n").split("\t") + [""] * 5)[:5]
    total += 1
    if codigo == "0":
        ok += 1
        continue
    if (proyecto, paso) in esperado:
        continue
    inesperados.append(f"  {proyecto}/{paso}: exit {codigo} {banderas}")

bloqueos = total - ok - len(inesperados)
print(f"{ok} de {total} pasos sin fallo ({100 * ok // max(total, 1)} %), {bloqueos} bloqueos esperados con instrucción")
if inesperados:
    print("FALLOS INESPERADOS:")
    print("\n".join(inesperados))
sys.exit(1 if inesperados else 0)
