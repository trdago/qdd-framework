#!/bin/bash
# Recorre el camino de una persona nueva en cada proyecto y registra cada paso.
# Uso: journey.sh <qdd> <carpeta-proyectos> <carpeta-salida>
QDD="$1"; B="$2"; OUT="$3"
rm -rf "$OUT" && mkdir -p "$OUT"
SUMMARY="$OUT/resumen.tsv"
printf "proyecto\tpaso\texit\tseg\tbanderas\n" > "$SUMMARY"

flags() {
	local f="$1" out=""
	grep -qiE "panic:|goroutine [0-9]+" "$f" && out="$out PANIC"
	grep -q "GATEKEEPER" "$f" && out="$out GATEKEEPER"
	grep -qE "❌|FALLO|\[!\]|\[🛑" "$f" && out="$out ALERTA"
	grep -qiE "error|inválid|invalid|no se pudo|failed|fatal" "$f" && out="$out ERROR-TXT"
	grep -qE "violaci" "$f" && out="$out VIOLACIONES"
	echo "${out# }"
}

step() {
	local p="$1" name="$2"; shift 2
	local log="$OUT/$p/$name.log" t0 t1 code
	mkdir -p "$OUT/$p"
	t0=$(date +%s.%N)
	( cd "$B/$p" && timeout 180 "$@" ) > "$log" 2>&1 < /dev/null
	code=$?
	t1=$(date +%s.%N)
	printf "%s\t%s\t%s\t%s\t%s\n" "$p" "$name" "$code" "$(awk -v a="$t0" -v b="$t1" 'BEGIN{printf "%.1f", b-a}')" "$(flags "$log")" >> "$SUMMARY"
}

# Mismo paso, pero desde una subcarpeta del proyecto (así trabaja la gente).
step_sub() {
	local p="$1" name="$2"; shift 2
	local sub
	sub=$(cd "$B/$p" && find . -mindepth 1 -maxdepth 2 -type d -not -path "./.*" | sort | tail -1)
	[ -z "$sub" ] && return
	local log="$OUT/$p/$name.log" t0 t1 code
	t0=$(date +%s.%N)
	( cd "$B/$p/$sub" && echo "(desde $sub)" && timeout 180 "$@" ) > "$log" 2>&1 < /dev/null
	code=$?
	t1=$(date +%s.%N)
	printf "%s\t%s\t%s\t%s\t%s\n" "$p" "$name" "$code" "$(awk -v a="$t0" -v b="$t1" 'BEGIN{printf "%.1f", b-a}')" "$(flags "$log")" >> "$SUMMARY"
}

mcp_session() {
	# initialize + tools/list + llamadas típicas de la primera tarea.
	# stdout (canal JSON-RPC) se valida aparte; stderr son logs legítimos.
	local out err
	out=$(mktemp) err=$(mktemp)
	mcp_raw > "$out" 2> "$err"
	echo "--- stderr ---"; cat "$err"
	echo "--- stdout ---"; cat "$out"
	echo "--- validación ---"
	python3 -I "$MEDICION/mcp_check.py" "$out" 1 2 3 4 5 6
	local code=$?
	rm -f "$out" "$err"
	return $code
}

mcp_raw() {
	printf '%s\n' \
		'{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"medicion","version":"1"}}}' \
		'{"jsonrpc":"2.0","method":"notifications/initialized"}' \
		'{"jsonrpc":"2.0","id":2,"method":"tools/list"}' \
		'{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"qdd_project_context","arguments":{}}}' \
		'{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"qdd_status","arguments":{}}}' \
		'{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"qdd_score","arguments":{}}}' \
		'{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"qdd_audit","arguments":{}}}' \
		| (cat; sleep 20) | "$QDD" mcp-server
}

dashboard_probe() {
	"$QDD" dashboard > dashboard.out 2>&1 &
	local pid=$! code=1
	for _ in $(seq 1 20); do
		sleep 0.5
		port=$(grep -oE "localhost:[0-9]+|127.0.0.1:[0-9]+" dashboard.out | head -1)
		[ -n "$port" ] && curl -sf "http://$port/" -o /dev/null && code=0 && break
	done
	cat dashboard.out; echo "puerto=$port http_ok=$code"
	kill $pid 2>/dev/null; wait $pid 2>/dev/null
	rm -f dashboard.out
	return $code
}

run_por_nombre() {
	# Como lo usa la gente: `qdd` desde el PATH, no por ruta absoluta.
	PATH="$(dirname "$QDD"):$PATH" qdd run "doctor"
}

actualizacion() {
	# FND-050: un proyecto certificado con la escala anterior (sin "escala") no
	# debe quedar "Empeorando" al actualizar QDD.
	local h=.qdd/project/metrics/certificate_history.json
	mkdir -p "$(dirname "$h")"
	echo '[{"timestamp":"2026-10-23T10:00:00Z","score":100,"total_violations":0,"tendency":"Estable"}]' > "$h"
	local out
	out=$("$QDD" certify 2>&1; "$QDD" evolution 2>&1)
	echo "$out"
	! echo "$out" | grep -qE "EMPEORANDO|Empeorando|CRITICAL"
}

dashboard_coherente() {
	# FND-051: el dashboard muestra el mismo puntaje que qdd audit. Sin
	# historial, para probar también que mide solo al abrirse.
	local auditado mostrado port state
	rm -f .qdd/project/metrics/certificate_history.json
	auditado=$("$QDD" audit 2>/dev/null | grep -oE "Puntaje QDD: [0-9]+" | grep -oE "[0-9]+")
	"$QDD" dashboard > dashboard.out 2>&1 &
	local pid=$!
	for _ in $(seq 1 60); do
		sleep 0.5
		port=$(grep -oE "localhost:[0-9]+|127.0.0.1:[0-9]+" dashboard.out | head -1)
		[ -z "$port" ] && continue
		state=$(curl -sf "http://$port/api/state") || continue
		echo "$state" | grep -q '"grade":"Sin medir"' && continue
		mostrado=$(echo "$state" | grep -oE '"score":[0-9]+' | head -1 | grep -oE "[0-9]+")
		break
	done
	kill $pid 2>/dev/null; wait $pid 2>/dev/null
	rm -f dashboard.out
	echo "audit=$auditado dashboard=$mostrado"
	[ -n "$auditado" ] && [ "$auditado" = "$mostrado" ]
}

MEDICION="$(cd "$(dirname "$0")" && pwd)"
export -f mcp_session mcp_raw dashboard_probe run_por_nombre actualizacion dashboard_coherente
export QDD MEDICION

for p in $(ls "$B"); do
	step "$p" 01-version "$QDD" --version
	step "$p" 02-init "$QDD" init
	step "$p" 03-init-otra-vez "$QDD" init
	step "$p" 04-doctor "$QDD" doctor
	step "$p" 05-audit "$QDD" audit
	step "$p" 06-evolution "$QDD" evolution
	step "$p" 07-report "$QDD" report
	step "$p" 08-modules "$QDD" modules
	step "$p" 09-modules-context "$QDD" modules context
	step "$p" 10-blueprint-list "$QDD" blueprint list
	step "$p" 11-certify "$QDD" certify
	step "$p" 12-mcp bash -c mcp_session
	step "$p" 13-dashboard bash -c dashboard_probe
	step_sub "$p" 14-audit-desde-subcarpeta "$QDD" audit
	step_sub "$p" 15-modules-desde-subcarpeta "$QDD" modules
	step "$p" 16-run-por-nombre bash -c run_por_nombre
	if [ "$p" = vacio ]; then
		# El camino que el propio QDD indica para un proyecto sin código.
		step "$p" 17-init-declarando "$QDD" init --lenguaje Go --arquitectura "API REST"
		step "$p" 18-modules-tras-declarar "$QDD" modules
	fi
	step "$p" 19-actualizacion-desde-escala-anterior bash -c actualizacion
	step "$p" 20-dashboard-igual-que-audit bash -c dashboard_coherente
	(cd "$B/$p" && git status --short 2>/dev/null) > "$OUT/$p/99-archivos-creados.txt"
done
column -t -s $'\t' "$SUMMARY"
