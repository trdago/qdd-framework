# RFC 0005: Reglas como principios con detección adaptable

**Status:** Draft
**Created:** 2026-10-28
**Origen:** revisión externa de QDD sobre un proyecto Node/serverless en producción (2026-10-28).

## 1. Summary

Una regla de QDD deja de ser un chequeo escrito para un lenguaje y pasa a ser un **principio**: qué se exige, por qué y con qué severidad. La forma de verificarlo se **adapta al contexto de cada proyecto**: lo que el proyecto declara, las herramientas que ya usa o, como respaldo, un detector genérico de QDD.

Cuando un principio no se puede verificar en un proyecto, el gate lo informa como **no evaluado**. Nunca aprueba en silencio.

## 2. Motivation

La revisión externa encontró que el gate puede dar una confianza falsa. Lo verificamos en el código:

| Hallazgo | Causa en el código |
|---|---|
| La API de producción estaba abierta a internet y `qdd audit` no lo mencionó, aunque QDD dice certificar OWASP. | `owasp.go` solo analiza archivos `.go`, con dos reglas (SQL injection y secretos escritos en el código). En Node evalúa cero cosas y no lo dice. |
| El puntaje subió de 70 a 100 creando `docs/adr/` y moviendo carpetas. | `enterprise.go` hace `os.Stat("docs/adr")`. `QA-02-E2E-TESTS` busca 4 nombres de carpeta en la raíz. Una carpeta vacía en el lugar correcto aprueba. |
| El backend no tenía pruebas y QDD no lo reportó. | `detectTestCommands` revisa subcarpetas solo si la raíz no tiene pruebas. Con un frontend en la raíz, `backend/` se ignora. |
| "Cero else" parece un gusto de estilo. | `CERT-040` la justifica con "reducir la complejidad ciclomática", lo que es falso: la decisión la agrega el `if`, no el `else`. Además la marca `CRITICAL` y su mensaje solo dice "QDD exige". Su razón real, que es facilitar la auditoría humana, no aparece escrita en ninguna parte. |
| `init` agrega reglas que el proyecto no usa (gobernanza de IA sin IA, plugins Python en un proyecto Node). | `DefaultPolicies()` activa todo y `init` copia `core/plugins` en cualquier proyecto. |

Además, los YAML de `.qdd/core/certification/` **no los lee el motor**: los chequeos están escritos en Go. Lo que el certificado promete y lo que el gate revisa ya no coinciden.

La causa es una sola: **cada regla nació atada a un lenguaje o a una estructura de carpetas, no a un principio.**

## 3. Design

### 3.1 El principio es la fuente de verdad

Cada certificación declara sus principios en YAML. El motor los lee: un principio que no está en un YAML no existe.

```yaml
id: P-ACCESS-CONTROL
nombre: Todo punto de entrada expuesto declara su control de acceso
por_que: |
  El control de acceso roto es el riesgo #1 de OWASP. Un endpoint público
  por omisión es un incidente, no una deuda.
severidad: bloqueante          # bloqueante | recomendacion
categoria: seguridad           # seguridad | correccion | auditabilidad | madurez
evidencia: contenido           # contenido (se revisa qué dice) | existencia (solo que esté)
aplica_si: expone_entradas     # rasgo del contexto (3.3); si no aplica, no se evalúa ni se informa
```

- `por_que` es obligatorio y aparece en cada violación. Una regla sin una razón que se pueda defender no se publica.
- `categoria` separa en el informe lo que protege al sistema de lo que facilita leerlo o mantenerlo.
- `evidencia: existencia` solo se permite en principios de `madurez`, nunca en los de `seguridad` ni `correccion`.

### 3.2 Detectores: cómo se verifica en cada contexto

Un principio tiene uno o más detectores. QDD los resuelve en este orden y usa el primero que aplique:

1. **Lo que declara el proyecto** en `.qdd/config.yaml` (o un blueprint de la organización):
   ```yaml
   principios:
     P-ACCESS-CONTROL:
       comando: "npm run check:authorizers"     # sale con 0 si cumple
     P-E2E-EVIDENCE:
       rutas: ["frontend/e2e/**/*.spec.ts"]
   ```
2. **La herramienta que el proyecto ya usa.** Por ejemplo, si ESLint está configurado con `no-else-return`, QDD lee su resultado y no duplica la regla.
3. **El detector genérico de QDD** para el rasgo detectado (por ejemplo, el adaptador de `serverless.yml`).
4. **Ninguno: el principio queda *no evaluado*.**

Contrato de un detector (Go):

```go
type Detector interface {
    Aplica(ctx Contexto) bool                 // ¿sabe verificar este contexto?
    Evaluar(ctx Contexto) (Resultado, error)  // Cumple | Viola(violaciones) | NoEvaluable(motivo)
}
```

Los detectores son la única parte que conoce un lenguaje o un formato. Agregar soporte para un stack nuevo es agregar un detector: el principio no cambia.

### 3.3 Contexto del proyecto

Antes de auditar, QDD arma un `Contexto` recorriendo **todo** el repositorio, no solo la raíz ni la primera carpeta que encuentra:

- **Unidades:** cada `package.json`, `go.mod`, `pubspec.yaml`, `pyproject.toml`, etc. Cada unidad tiene su stack, sus herramientas y sus comandos de prueba.
- **Rasgos:** `expone_entradas` (serverless, rutas HTTP, API Gateway), `usa_llm` (dependencias de proveedores de IA), `despliega` (pipelines), `persiste_datos`, etc.
- **Lo declarado** en `.qdd/config.yaml` y en los blueprints importados.

El mismo `Contexto` lo usan `init`, `audit`, `doctor` y el MCP. Así, lo que `init` instala y lo que `audit` exige salen de la misma lectura del proyecto.

### 3.4 Informe y puntaje

El informe deja de ser un número solo. Primero muestra qué se pudo revisar:

```
Principios aplicables: 14    Evaluados: 11    No evaluados: 3
  ✗ P-ACCESS-CONTROL       seguridad     backend/serverless.yml: 3 funciones sin authorizer
  ? P-NO-INJECTION         seguridad     no evaluado en backend (Node): sin detector
  ✓ P-TESTS-PER-UNIT       correccion    frontend, backend
  · P-READABLE-FLOW        auditabilidad 4 recomendaciones
```

- El puntaje mantiene la escala actual (bloqueante −10, recomendación −2) y agrega la **cobertura** (`11/14 evaluados`). No se informa un 100 sin su cobertura.
- Un principio de `seguridad` o `correccion` no evaluado se informa como **no evaluado**. No aprueba en silencio ni queda en blanco: el informe dice qué detector falta o qué declarar.
- `certify` deja de emitir "certificado OWASP". Emite "principios de seguridad: N evaluados de M", con la lista de los que no se evaluaron.

### 3.5 Desactivar un principio es una decisión humana registrada

Un principio solo se desactiva de dos maneras:

- **desde el dashboard**, que ya edita `policies.yaml` mediante `/api/policies`;
- **porque el usuario lo pide explícitamente** a la IA o en la CLI (`qdd principio desactivar <id> --por-que "..."`).

La IA nunca desactiva un principio por iniciativa propia, ni siquiera para que el gate apruebe. Ninguna herramienta MCP desactiva principios sin una petición explícita del usuario. Si la IA actúa por esa petición, registra la frase del usuario en `por_que`.

La desactivación queda registrada:

```yaml
# .qdd/policies.yaml
desactivados:
  P-READABLE-FLOW:
    por: "dpailahueque"          # usuario de git, o "dashboard:<usuario>"
    fecha: 2026-10-28
    por_que: "proyecto legado en migración; se reactiva al terminar"
```

- El informe no oculta un principio desactivado: lo muestra como `– desactivado` con quién, cuándo y por qué, y lo cuenta aparte de la cobertura (`11/14 evaluados, 1 desactivado`).
- Una desactivación sin `por_que` no es válida: el gate la informa como `P-00-INVALID-DISABLE` y el principio sigue activo.
- Un invariante de un blueprint no se puede desactivar en el proyecto (RFC 0004): el cambio va al repositorio de la organización.

### 3.6 `qdd doctor`

Esto aplica el principio de RFC 0004 §3.7: si doctor no puede diagnosticarlo, la regla quedó ambigua. Se agrega el grupo **"5. Cobertura de principios"**:

| Diagnóstico | `qdd doctor --fix` |
|---|---|
| Principio aplicable sin detector para una unidad | Indica qué declarar en `config.yaml` o qué herramienta configurar. No inventa un comando. |
| Comando declarado que no existe o no termina | Lo informa con su salida. No lo reemplaza. |
| Regla activa cuyo rasgo no está en el contexto (IA sin IA) | La desactiva en `policies.yaml` y lo informa. |
| Herramienta del proyecto que contradice un principio (ESLint con `no-else-return: off`) | Muestra la contradicción. La decisión es del humano. |

### 3.7 `init` mínimo

`init` instala solo lo que corresponde al `Contexto`:

- Principios cuyo `aplica_si` se cumple. Gobernanza de IA solo si `usa_llm`.
- Reglas para **el asistente que se detecta en uso**, o para el que se indique con `--asistente`, no para los cuatro.
- Plugins y ejemplos solo con `--ejemplos`.
- `qdd bug` genera el esqueleto de la prueba en el stack de la unidad afectada, no en Go.

### 3.8 Medir el valor del gate

Cada finding agrega `detectado_por: qdd | humano | ia | produccion`. `qdd report` muestra cuántos defectos reales encontró el gate frente a los que encontraron otras fuentes. Con eso se puede responder con datos si la auditoría aporta, como pide la revisión externa.

### 3.9 Indicadores

Los indicadores que muestran el dashboard, `qdd_score`, `report` y `certify` siguen el mismo principio que las reglas. Una revisión del dashboard (2026-10-29) encontró que hoy no lo cumplen:

| Problema | Dónde | Finding |
|---|---|---|
| Historial de certificaciones simulado, "horas ahorradas" con constantes y descripción fija de cada finding. | `dashboard/state.go:543`, `:590`, `:773`, `:599` | FND-055 |
| El estado de un finding se interpreta distinto en el dashboard y en el MCP. Hay 24 findings resueltos sin `status`, que el dashboard cuenta como abiertos. | `state.go:580`, `mcp_telemetry.go:141` | FND-056 |
| Dos escalas de grado para el mismo puntaje. | `state.go:819`, `mcp_telemetry.go:149` | FND-057 |
| Ciclo de refresco: el dashboard se reconstruye cada 500 ms porque escribe dentro de lo que observa. | `state.go:97-185` | FND-058 |
| El grafo de código solo indexa `.go`, y una prueba solo se reconoce por `_test.go` o `.spec.ts`. | `graph/db.go:302`, `:339` | §3.3 |
| La topología etiqueta por defecto `.go`, `.vue`, `.css`, `.html` y las rutas `ui/` y `pkg/`, que es la estructura de QDD. | `topology/mapper.go:74` | §3.3 |
| "Módulo certificado" significa "no tiene `else`", y un endpoint se marca "falta OWASP" porque su módulo tiene un `else`. | `topology/mapper.go:309`, `:354` | §3.1 |
| El pilar de un finding (seguridad, estabilidad…) se adivina por palabras clave. | `state.go:620` | §3.1 |
| "System Telemetry" muestra la memoria del proceso del dashboard, no algo del proyecto. | `state.go:406` | — |

**Invariante: los datos simulados y los mocks solo existen en las pruebas.** El producto nunca los contiene ni los muestra. Si un indicador no tiene una fuente real, muestra **sin medir**.

Diseño:

1. **Cada indicador se declara**, igual que un principio:
   ```yaml
   id: I-FINDINGS-ABIERTOS
   mide: findings con status open
   fuente: .qdd/project/findings/*.yaml   # esquema validado (FND-056)
   formula: count(status == open)
   unidad: findings
   ```
   Un solo paquete (`pkg/indicadores`) los calcula, y el dashboard, el MCP, `report` y `certify` los consumen. No puede haber dos fórmulas ni dos escalas para el mismo indicador.
2. **Las fuentes salen del `Contexto`** (§3.3): unidades y stacks reales, detectores de entradas y pruebas, y rutas declaradas. El grafo y la topología indexan cada unidad con el adaptador de su stack. Si una unidad no tiene adaptador, su indicador queda *sin medir* y no se omite.
3. **"Certificado" significa "cumple sus principios aplicables"**, desglosado por categoría (§3.1). La categoría de un finding se declara en el finding (`categoria:`), no se adivina por palabras clave.
4. **El valor se mide con datos reales:** `detectado_por` (§3.8), el tiempo entre que se abre y se cierra un finding (fechas de git) y los defectos que llegaron a producción (`detectado_por: produccion`). No hay constantes de "horas ahorradas".
5. **El esquema del finding** tiene `status` obligatorio (`open | resolved | wontfix`, en minúscula), `categoria` y `detectado_por`. El gate lo valida, y `qdd doctor --fix` normaliza las mayúsculas y pregunta lo que falta. Doctor nunca inventa un estado.
6. **El dashboard no escribe dentro de lo que observa** (FND-058). La telemetría del proceso pasa a una vista de diagnóstico, separada de los indicadores del proyecto.

## 4. Ejemplos

### 4.1 Cero else → `P-READABLE-FLOW`

```yaml
id: P-READABLE-FLOW
nombre: Las funciones se leen de arriba abajo
por_que: |
  El código se escribe una vez y se audita muchas. Con cláusulas de guarda,
  quien audita lee primero los casos rechazados y al final el camino normal,
  sin anidación. No previene defectos por sí sola: reduce el costo de auditar.
severidad: recomendacion
categoria: auditabilidad
evidencia: contenido
aplica_si: siempre
```

Detectores, en orden: lo que declare el proyecto → `no-else-return` (ESLint), `RET505` (ruff), `revive:early-return` (Go) si están configurados → el detector genérico actual. Violación:

> `else` en `src/pagos.ts:42`. Usa una cláusula de guarda para que la función se lea de arriba abajo en una auditoría.

### 4.2 OWASP A01 → `P-ACCESS-CONTROL`

Detectores iniciales: `serverless.yml` (función con evento `http`/`httpApi` sin `authorizer` y sin `private: true`), Express/Fastify (rutas registradas sin middleware de autenticación declarado), Go `net/http` (handlers sin middleware declarado). Para los últimos dos casos, el proyecto declara cuál es su middleware de autenticación; QDD no lo adivina.

Un endpoint que debe ser público se declara así en la configuración, con su razón:

```yaml
principios:
  P-ACCESS-CONTROL:
    publicos:
      - ruta: "GET /health"
        por_que: "sonda del balanceador"
```

## 5. Migración de las reglas actuales

| Regla actual | Principio | Cambio |
|---|---|---|
| `OWASP-A03-INJECTION`, `OWASP-A02-CRYPTOGRAPHY` | `P-NO-INJECTION`, `P-NO-PLAIN-SECRETS` | El detector Go actual pasa a ser uno de varios; en otros stacks se informa *no evaluado* hasta que exista su detector. |
| — (no existía) | `P-ACCESS-CONTROL` | Nuevo, bloqueante. |
| `QA-00`, `QA-01` | `P-TESTS-PER-UNIT` | Se ejecutan las pruebas de **cada** unidad, y una unidad sin pruebas es una violación. |
| `QA-02-E2E-TESTS` | `P-E2E-EVIDENCE` | Revisa contenido: al menos un archivo de prueba en las rutas declaradas o detectadas. |
| `CERT-030-MANDATORY-ADR` | `P-DECISIONS-RECORDED` | Al menos un ADR con estado, en la ruta declarada (por defecto `docs/adr/`). |
| `CLEAN-01-NO-ELSE` / `CERT-040` | `P-READABLE-FLOW` | Recomendación de auditabilidad, con su razón (§4.1). |
| Gobernanza de IA (`CERT-05x`) | `aplica_si: usa_llm` | No se evalúa ni se instala en proyectos sin IA. |

Los `RuleID` actuales se mantienen como alias durante una versión mayor, para no romper las supresiones (`FilterIgnoredViolations`) ni los historiales.

## 6. El mismo principio aplicado a la herramienta

La revisión también señaló fallas de madurez de QDD como herramienta. Tienen la misma causa que las reglas: **QDD supone un entorno fijo** (GitHub, Linux, MCP conectado, un solo flujo) en vez de leer el contexto. Se resuelven con el mismo patrón: un principio, varios adaptadores y un respaldo explícito cuando ningún adaptador aplica.

| Hallazgo | Causa verificada | Principio | Solución general |
|---|---|---|---|
| `propose` solo abre PRs en GitHub (SERVEL usa Azure DevOps). | `evolucion.go:366` llama directo a `gh pr create`. | QDD se adapta a la plataforma del remote. | Interfaz `ProveedorPR`, elegida según el host del remote: GitHub (`gh`), Azure DevOps (`az repos pr create`), GitLab (`glab`). **Respaldo:** publica la rama e imprime la URL para abrir el PR a mano. Es el mismo contrato de "no evaluado": nunca falla en silencio. |
| `propose` no incluye el README del repositorio en el commit. | `evolucion.go:450` hace `git add -- <subdir>`: todo lo que está fuera de la carpeta del blueprint queda fuera. | Un aporte incluye todo lo que cambió en el espacio de aporte. | `propose` lista todos los archivos modificados del clon de aporte, pide confirmación al humano y commitea esa lista. No se usa una lista fija. |
| Para registrar el repositorio de blueprints hay que importar uno que no se necesita. | `registry.go:133`: el registro solo ocurre dentro de `blueprint add <url> <id>`. | Cada acción hace una sola cosa. | `qdd blueprint registrar <url>` registra la organización sin importar nada. `add <url> <id>` se mantiene como atajo. |
| "Ejecuta qdd_learn" depende de un servidor MCP que no estaba conectado. | `init.go:96-103` da la instrucción sin comprobar que el MCP esté conectado. | Toda herramienta MCP tiene un equivalente en la CLI, y las instrucciones se adaptan a lo que está disponible. | Cada herramienta MCP declara su comando CLI equivalente, y se crea si falta. `init` y el MCP indican el que funciona en ese entorno. Doctor diagnostica "MCP no conectado" y explica cómo conectarlo. |
| La instalación falló en Windows desde Git Bash. | **Hipótesis por reproducir:** `npm/install.js` extrae el binario con `execSync("tar ...")`. El `tar` de GNU que trae Git Bash interpreta `C:\...` como un host remoto. | La instalación no depende de herramientas del sistema. | Extraer el archivo con Node, sin llamar a la shell. El CI agrega una matriz de instalación en Windows (PowerShell y Git Bash), macOS y Linux. Doctor agrega el diagnóstico "binario instalado y ejecutable". |
| La v2 cambió el flujo respecto de la v1. | La migración es automática (RFC 0004 §3.6), pero nadie le explica al usuario qué cambió en el flujo. | Un cambio de versión mayor se explica una vez, en el momento en que afecta al usuario. | La primera vez que corre una versión mayor nueva, QDD muestra los cambios de flujo de esa versión (desde el CHANGELOG) y lo registra en `state.json` para no repetirlo. |
| `qdd bug` generó una prueba en Go para un proyecto Node, con una plantilla sobre un "proceso supervisado" que no existía. | Las plantillas son fijas y el texto supone `qdd run --keep-alive`. | Las plantillas se derivan del `Contexto` (§3.3). | El esqueleto de la prueba se genera en el stack de la unidad afectada, y el texto menciona solo las fuentes de errores que el proyecto tiene. |

## 7. Fases

1. **Honestidad del informe:** `Contexto` con todas las unidades, estado *no evaluado*, cobertura en el informe, y `por_que` y `categoria` en las violaciones existentes. Corrige CERT-040. Corrige FND-055 a FND-058: sin datos simulados, un solo lector de findings, una sola escala de grado y sin ciclo de refresco. No agrega reglas nuevas.
2. **Detectores de contenido:** `P-ACCESS-CONTROL` (serverless primero), `P-TESTS-PER-UNIT`, `P-E2E-EVIDENCE`, `P-DECISIONS-RECORDED`.
3. **Detectores declarados y herramientas existentes:** `principios:` en `config.yaml` y lectura de los linters del proyecto.
4. **`init` mínimo, grupo 5 de doctor, `detectado_por` y plantillas derivadas del contexto.**
5. **Herramienta (§6):** `ProveedorPR`, `propose` con todos los cambios, `blueprint registrar`, equivalentes CLI de las herramientas MCP, instalación sin shell y matriz de instalación en el CI, aviso de cambios de flujo.

Las fases 1 y 2 van primero porque son las que cortan la falsa confianza. La fase 5 es independiente y puede avanzar en paralelo.

Cada fase lleva sus findings con prueba de regresión, siguiendo el método QDD.

## 8. Drawbacks

- **Breaking (v3.0.0):** proyectos que hoy están en 100 bajan o quedan con cobertura parcial. Es el efecto buscado, pero hay que comunicarlo.
- Los detectores que leen herramientas de terceros dependen de sus formatos de salida, que pueden cambiar.
- Declarar el middleware de autenticación o los endpoints públicos es trabajo para el equipo del proyecto. Sin esa declaración, el principio queda *no evaluado* en esos stacks.

## 9. Decisiones (2026-10-28)

1. **Un principio de `seguridad` o `correccion` no evaluado** se informa sin bloquear en `adopcion: gradual` y **bloquea** en `adopcion: estricta`. En ambos modos se puede desactivar desde el dashboard o a pedido explícito del usuario, con registro (§3.5).
2. **Los principios propios de una organización** (por ejemplo, "todo endpoint pasa por Cognito") viven en sus blueprints, con el mismo formato y con las reglas de invariantes del RFC 0004.
3. **La fase 1 sale como 2.2.0**, porque solo agrega información y no cambia lo que bloquea. Las fases que cambian lo que bloquea llevan versión mayor (CERT-024).
