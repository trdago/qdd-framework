#!/usr/bin/env python3
# Compara el costo en reposo de cada forma del panel (RFC 0016). No se manda ningún mensaje a Claude: no gasta tokens.
# Uso: medir-panel.py <worktree> <segundos> <salida.json> <checkout principal>. Los binarios van en <worktree>/tmp/bin.
import json, os, shutil, subprocess, sys, time
W = sys.argv[1]
SEG = int(sys.argv[2])
SALIDA = sys.argv[3]
B = f'{W}/tmp/banco'
HOME_QDD = f'{W}/tmp/banco-home'
MAIN = sys.argv[4]  # el checkout con el estado.yaml real, que se copia al banco
BIN = f'{W}/tmp/bin'
MOD = f'{W}/cli/pkg/consola/mod/qdd-tareas'
SOCK = 'qddbanco'
HZ = os.sysconf('SC_CLK_TCK')
shutil.rmtree(B, ignore_errors=True); shutil.rmtree(HOME_QDD, ignore_errors=True)
os.makedirs(f'{B}/.qdd/project'); os.makedirs(HOME_QDD)
shutil.copy(f'{MAIN}/.qdd/project/estado.yaml', f'{B}/.qdd/project/estado.yaml')
shutil.copytree(f'{W}/.qdd/project/modules', f'{B}/.qdd/project/modules')

def tmux(*a, **k):
    limpio = {k2: v for k2, v in os.environ.items() if not k2.startswith('CLAUDE')}
    return subprocess.run(['tmux', '-L', SOCK, *a], capture_output=True, text=True, env=limpio, **k)

PATH = os.environ['PATH']
SIN = '--settings \'{"enabledPlugins":{"qdd-tareas@qdd":false}}\''
DIRECTO = f'env PATH={BIN}:{PATH}'
ESC = {
  'sin_panel_wrapper': f'claude {SIN}',
  'sondeo_wrapper':    'claude',
  'sin_panel_directo': f'{DIRECTO} claude {SIN}',
  'sondeo_directo':    f'{DIRECTO} claude',
  'flujo_directo':     f'{DIRECTO} claude {SIN} --plugin-dir {MOD}',
}
pids = {}
tmux('kill-server')
tmux('start-server', ';', 'set', '-g', 'exit-empty', 'off', ';', 'set', '-g', 'remain-on-exit', 'on')
for i, (nom, cmd) in enumerate(ESC.items()):
    arg = ['new-session', '-d', '-s', nom, '-x', '170', '-y', '50', '-c', B, '-e', f'QDD_HOME={HOME_QDD}', cmd]
    r = tmux(*arg)
    if r.returncode: print('fallo', nom, r.stderr, flush=True)
time.sleep(37)

def hijos():
    h = {}
    for p in os.listdir('/proc'):
        if p.isdigit():
            try:
                pp = int(open(f'/proc/{p}/stat').read().rsplit(')', 1)[1].split()[1])
                h.setdefault(pp, []).append(int(p))
            except OSError:
                pass
    return h

def arbol(pid, h):
    res, pila = [], [pid]
    while pila:
        x = pila.pop(); res.append(x); pila += h.get(x, [])
    return res

def foto():
    h = hijos(); out = {}
    for nom in ESC:
        panes = tmux('list-panes', '-t', nom, '-F', '#{pane_pid}').stdout.split()
        if not panes:
            out[nom] = None; continue
        root = int(panes[0]); propia = 0; cpu = 0; pss = 0; rss = 0; cuenta = 0; nombres = []
        # El proceso raíz acumula en cutime/cstime todo lo que ya terminó (sus hijos cortos); los vivos se suman aparte.
        for p in arbol(root, h):
            try:
                if open(f'/proc/{p}/comm').read().strip() == 'Xvfb':
                    continue  # la pantalla falsa de la prueba no es costo de la ventana
                f = open(f'/proc/{p}/stat').read().rsplit(')', 1)[1].split()
                cpu += int(f[11]) + int(f[12])
                cpu += int(f[13]) + int(f[14])
                if p == root:
                    propia = int(f[11]) + int(f[12])
                sm = open(f'/proc/{p}/smaps_rollup').read()
                for l in sm.splitlines():
                    if l.startswith('Pss:'): pss += int(l.split()[1])
                    if l.startswith('Rss:'): rss += int(l.split()[1])
                cuenta += 1
                nombres.append(open(f'/proc/{p}/comm').read().strip())
            except OSError:
                pass
        out[nom] = dict(cpu=cpu, propia=propia, pss=pss, rss=rss, procs=cuenta, nombres=sorted(set(nombres)))
    return out

a = foto(); t0 = time.time()
pss_max = {}
muestras = {n: [] for n in ESC}
while time.time() - t0 < SEG:
    time.sleep(15)
    f = foto()
    for n in ESC:
        if f[n]: muestras[n].append(f[n]['pss'])
b = foto(); dt = time.time() - t0
pantallas = {n: tmux('capture-pane', '-p', '-t', n).stdout for n in ESC}
h_ = hijos()
vivos = {}
for n in ESC:
    raiz = tmux('list-panes', '-t', n, '-F', '#{pane_pid}').stdout.split()
    vivos[n] = [open(f'/proc/{p}/cmdline').read().replace(chr(0), ' ')[:90] for p in arbol(int(raiz[0]), h_)[1:] if os.path.exists(f'/proc/{p}/cmdline')] if raiz else []
res = {}
for n in ESC:
    if not a[n] or not b[n]:
        res[n] = {'error': 'no siguió viva'}; continue
    m = muestras[n]
    res[n] = dict(claude_pct=round((b[n]['propia'] - a[n]['propia']) / HZ / dt * 100, 2), hijos_pct=round(((b[n]['cpu'] - a[n]['cpu']) - (b[n]['propia'] - a[n]['propia'])) / HZ / dt * 100, 2), cpu_pct=round((b[n]['cpu'] - a[n]['cpu']) / HZ / dt * 100, 2), cpu_s=round((b[n]['cpu'] - a[n]['cpu']) / HZ, 2),
                  pss_mb=round(sum(m) / max(len(m), 1) / 1024, 1), rss_mb=round(b[n]['rss'] / 1024, 1), procs=b[n]['procs'], nombres=b[n]['nombres'])
json.dump(dict(segundos=round(dt), resultado=res, pantallas=pantallas, vivos=vivos), open(SALIDA, 'w'), ensure_ascii=False, indent=1)
tmux('kill-server')
