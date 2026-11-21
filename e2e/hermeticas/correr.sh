#!/bin/sh
# Pruebas herméticas (FND-246): corre las pruebas de Go con un HOME falso y con espías en lugar de los programas que
# tocan la máquina o la red (claude, gh, npm, systemctl…). Falla si alguna prueba escribió en las carpetas de la
# persona (QDD_HOME, ~/.claude, unidades de systemd) o lanzó uno de esos programas. Uso: e2e/hermeticas/correr.sh
# contrato: gitenv#83f46fc1
set -eu
raiz=$(cd "$(dirname "$0")/../.." && pwd)
caja=$(mktemp -d)
trap 'rm -rf "$caja"' EXIT
mkdir -p "$caja/bin" "$caja/home" "$caja/run"
for p in claude gh npm npx systemctl curl op secret-tool cloudflared ollama docker; do
	printf '#!/bin/sh\necho "%s $*  (en $PWD)" >> "%s/llamadas.log"\nexit 1\n' "$p" "$caja" > "$caja/bin/$p"
	chmod +x "$caja/bin/$p"
done
cache=$(go env GOCACHE)
modcache=$(go env GOMODCACHE)
cd "$raiz/cli"
env -u QDD_HOME -u CLAUDECODE -u CLAUDE_CODE_SESSION_ID -u CLAUDE_CONFIG_DIR \
	HOME="$caja/home" XDG_CONFIG_HOME="$caja/home/.config" XDG_RUNTIME_DIR="$caja/run" \
	GOCACHE="$cache" GOMODCACHE="$modcache" GOFLAGS=-buildvcs=false PATH="$caja/bin:$PATH" \
	go test -count=1 ./... > "$caja/go-test.log" 2>&1 || { tail -40 "$caja/go-test.log"; echo "las pruebas fallaron dentro de la caja"; exit 1; }
falla=0
for d in .local/share/qdd .claude .config/systemd .config/qdd; do
	[ -e "$caja/home/$d" ] || continue
	echo "una prueba escribió en ~/$d de la persona:"
	find "$caja/home/$d" | sed "s#$caja/home/#  ~/#" | head -20
	falla=1
done
if [ -s "$caja/llamadas.log" ]; then
	echo "una prueba lanzó programas reales:"
	sort "$caja/llamadas.log" | uniq -c | sort -rn | head -20
	falla=1
fi
[ "$falla" -eq 0 ] && echo "[✓] Pruebas herméticas: ninguna escribió en las carpetas de la persona ni lanzó programas reales"
exit "$falla"
