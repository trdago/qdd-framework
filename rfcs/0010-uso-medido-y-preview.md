# RFC 0010: Uso real medido, canal preview y autorización según el modo de la IA

- Estado: aceptado (decisiones del usuario, 2026-10-30)
- Versión: qdd 2.8.0 (primera preview: v2.8.0-rc.1)

## Problema

1. Una versión con funciones nuevas salía como `latest` apenas pasaba las pruebas. Nadie la había usado en un proyecto
   real, así que los defectos que solo aparecen con el uso llegaban a todos.
2. La IA pedía autorización por cosas que solo continuaban el trabajo del proyecto, aunque la sesión estuviera en modo
   auto. Y no siempre la pedía antes de cambiar algo que el proyecto ya había decidido.
3. Los pedidos hechos se quedaban 30 minutos en el tablero después de verlos.

## Decisiones

### Uso medido (`qdd uso`, `pkg/uso`)

- qdd anota su propio uso real por versión en `<QDD_HOME>/uso/<versión>.json`, siempre dentro de un proyecto QDD. Cada
  una de estas cosas es un evento:
  - un comando de qdd;
  - una herramienta MCP;
  - un pulso del agente (`qdd estado pulso`, que el mod envía con cada herramienta como mucho cada 5 s, haya o no una
    tarea en curso);
  - un mensaje de la persona con el panel (`qdd consola ciclo`).
- Se suma el tiempo entre dos eventos separados por menos de 5 minutos. Un evento aislado no suma: las horas se miden,
  nunca se estiman.
- Una compilación `dev` no se mide. El sondeo del panel (`qdd consola panel`, `statusline`) no es uso.
- `qdd uso` muestra lo medido.
- `qdd uso evidencia <versión>` escribe en `.qdd/project/evidence/uso/vX.Y.Z.yaml` las horas de la versión y de sus
  candidatas, por proyecto. Ese archivo se commitea con la release.

### Canal preview y VER-05

- Un tag `vX.Y.Z-rc.N` es una **preview**: se publica en npm con `--tag preview` (`npm i -g qdd-framework@preview`) y como
  pre-release en GitHub.
- Un tag `vX.Y.Z` va a **latest**.
- VER-05-UNPROVEN-RELEASE es opcional por proyecto: se activa con `uso_minimo_horas: N` en `.qdd/policies.yaml`. qdd
  usa 5.
- Con VER-05 activo, una versión estable que sube minor o major respecto del último tag estable necesita la evidencia
  con al menos N horas. Un fix y una preview no esperan.

### Ciclo por mensaje y modo de la IA

- El mod de Claude Code oye `classic.UserPromptSubmit` y llama
  `qdd consola ciclo --modo <permission_mode>` sin esperar.
- Un pedido hecho antes del último ciclo ya se vio y sale del tablero. El recuadro del resultado también se limpia.
- El modo se guarda local por repositorio, como la cola. Solo se anotan los modos de Claude Code, y uno de hace más de
  12 h queda «sin medir».
- `qdd_project_context` y `qdd modules context` traen la sección «Autorización según el modo de la IA» con la regla del
  modo:
  - **auto** (y acceptEdits, bypassPermissions, dontAsk): lo que continúa la línea del proyecto sigue sin preguntar.
    Hay que pedir autorización solo antes de cambiar algo ya definido:
    - el objetivo, las decisiones o las invariantes de una ficha aprobada;
    - una decisión o una invariante de un blueprint;
    - una versión, un tag, una publicación o un despliegue;
    - algo que borra datos o no se puede deshacer.
  - **default**: la consola ya pide permiso por cada acción; no se suman preguntas.
  - **plan**: solo se planifica.
- `doctor` (grupo de la consola) muestra el uso medido de la versión que corre y el modo de la IA. Un registro de uso
  ilegible es una falla, y `--fix` lo aparta.

### Panel

- Sale la sección «⏳ Esperando»: el ⏳ ya está en la tabla y en el resumen, y su detalle en `qdd estado ver`.

## Cómo se prueba

- `cd cli && go test ./pkg/uso/ ./pkg/audit/ ./pkg/consola/ ./pkg/modules/ ./cmd/`
- `claude plugin test cli/pkg/consola/mod/qdd-tareas` y `node --test cli/pkg/consola/mod/opencode/panel.test.js`
