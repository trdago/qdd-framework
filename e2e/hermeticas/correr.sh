#!/bin/sh
# Pruebas herméticas (FND-246): corre las pruebas de Go con un HOME falso y con espías en lugar de los programas que
# tocan la máquina o la red (claude, gh, npm, systemctl…), y sin red si hay bwrap. Falla si alguna prueba escribió en las carpetas de la
# persona (QDD_HOME, ~/.claude, unidades de systemd) o lanzó uno de esos programas. Uso: e2e/hermeticas/correr.sh
# Corren además como desde un hook de git: con GIT_DIR y GIT_INDEX_FILE exportados hacia un repositorio que tiene que
# quedar igual, sin commits, tags, remotes ni ramas nuevas (FND-117, FND-263).
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
# Si este script corre desde un hook, su propio git no puede usar el repositorio de ese hook.
unset GIT_DIR GIT_WORK_TREE GIT_INDEX_FILE GIT_COMMON_DIR GIT_OBJECT_DIRECTORY GIT_ALTERNATE_OBJECT_DIRECTORIES GIT_NAMESPACE GIT_PREFIX
hook="$caja/hook"
GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1 git -c init.defaultBranch=main init -q "$hook"
GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1 git -C "$hook" -c user.name=hook -c user.email=hook@hook commit -q --allow-empty -m "repositorio del hook"
huella() { git -C "$hook" for-each-ref --format='%(refname) %(objectname)'; git -C "$hook" remote -v; git -C "$hook" config --local --list | sort; }
antes=$(huella)
cache=$(go env GOCACHE)
modcache=$(go env GOMODCACHE)
cd "$raiz/cli"
# Sin red (bwrap --unshare-net deja solo localhost, que usan los servidores de prueba): una prueba que llama a internet
# falla en vez de salir. Un borrador de prueba del baúl llegó a llamar a la API de Cloudflare con un token falso.
sinred=""
if command -v bwrap >/dev/null 2>&1 && bwrap --dev-bind / / --unshare-net true 2>/dev/null; then
	go mod download
	sinred="bwrap --dev-bind / / --unshare-net --die-with-parent --"
fi
[ -n "$sinred" ] || echo "[!] Sin bwrap las pruebas corren con red: una llamada real a internet no se detecta."
env -u QDD_HOME -u CLAUDECODE -u CLAUDE_CODE_SESSION_ID -u CLAUDE_CONFIG_DIR \
	HOME="$caja/home" XDG_CONFIG_HOME="$caja/home/.config" XDG_RUNTIME_DIR="$caja/run" \
	GOCACHE="$cache" GOMODCACHE="$modcache" GOFLAGS=-buildvcs=false PATH="$caja/bin:$PATH" \
	GIT_DIR="$hook/.git" GIT_INDEX_FILE="$hook/.git/index" GOPROXY=off \
	$sinred go test -count=1 ./... > "$caja/go-test.log" 2>&1 || { tail -40 "$caja/go-test.log"; echo "las pruebas fallaron dentro de la caja"; exit 1; }
falla=0
for d in .local/share/qdd .claude .config/systemd .config/qdd; do
	[ -e "$caja/home/$d" ] || continue
	echo "una prueba escribió en ~/$d de la persona:"
	find "$caja/home/$d" | sed "s#$caja/home/#  ~/#" | head -20
	falla=1
done
despues=$(huella)
if [ "$antes" != "$despues" ]; then
	echo "una prueba escribió en el repositorio del hook (GIT_DIR exportado):"
	printf '%s\n' "$despues" | head -20
	falla=1
fi
if [ -s "$caja/llamadas.log" ]; then
	echo "una prueba lanzó programas reales:"
	sort "$caja/llamadas.log" | uniq -c | sort -rn | head -20
	falla=1
fi
[ "$falla" -eq 0 ] && echo "[✓] Pruebas herméticas: ninguna escribió en las carpetas de la persona ni en el repositorio del hook, ni lanzó programas reales"
exit "$falla"
