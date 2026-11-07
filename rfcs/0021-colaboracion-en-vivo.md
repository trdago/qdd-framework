# RFC 0021: colaboración en vivo entre consolas

**Estado:** propuesta (2026-11-07). La persona aprobó el flujo `colaboracion-en-vivo` («me parece bien el flujo, continúa»).
Quedan abiertas las tres decisiones del final.

## Problema

Con LLM, escribir código es barato y lo caro es que todos entiendan lo mismo. Cada consola trabaja en su rama, y eso está
bien. Pero algunos cambios no pueden esperar al merge: si una consola cambia el modelo de datos, las demás tienen que
saberlo apenas ocurre, para adaptarse antes de construir sobre algo que ya no existe.

Hoy eso se coordina a mano. El 2026-11-07, en este PC, las consolas `qdd-6e`, `qdd-movile-13` y `sfrc-35` se pasaron por
mensajes los nombres de un evento nuevo (`sincronizar`), un campo (`reglas_atrasadas`) y un cambio de un blueprint que
no debía salir junto con otro. Cada mensaje lo escribió una IA y lo leyó otra.

## Idea: separar saber de traer

- **Saber** es inmediato y barato: un aviso corto, sin código.
- **Traer** espera el merge, salvo que a quien recibe le importe ya.

### El criterio: ¿rompe a quien no lo sabe?

Un cambio es de **contrato** si otra consola puede depender de él sin saberlo:

- migraciones y esquema de datos;
- API pública, eventos y mensajes entre programas;
- variables de entorno (`.env.example`);
- invariantes de fichas y blueprints (`.qdd/project/modules/`, `.qdd/blueprints.lock.yaml`).

Un contrato avisa al tiro (⚡). Lo demás va al resumen y espera el merge (📋).

La clasificación no usa LLM. Cada proyecto declara sus contratos con rutas, como las `rutas` de las suites
(RFC 0020), y QDD compara los archivos del cambio con esas rutas. Sin declaración, QDD propone las de arriba al detectar
el proyecto, y la persona las ajusta.

```yaml
contratos:
  - nombre: esquema
    rutas: [supabase/migrations]
  - nombre: eventos-remoto
    rutas: [cli/pkg/remoto/estado.go, rfcs/0013-consola-remota.md]
usa:
  - qdd/eventos-remoto   # proyecto/contrato del que depende
```

La IA que hizo el cambio solo escribe la línea de resumen, que ya tiene en su sesión.

## Cómo funciona

1. **Una consola termina un cambio:** al hacer commit o marcar ✅ una tarea, nunca por cada tecla. Así hay pocos eventos.
2. **QDD lo clasifica** contra los contratos.
3. **⚡ Señal de contrato:** un registro de ~1 KB con quién, proyecto, rama, contrato, la línea de resumen, el commit y una
   huella. No lleva código.
4. **La otra consola ve si le importa, sin LLM.** Le importa si su proyecto declara `usa` de ese contrato o si su rama
   tocó las mismas rutas. Si no, la señal solo va al resumen.
5. **Su IA lo descubre:** en el siguiente mensaje de la persona, `qdd consola ciclo --contexto` le pone la señal en el
   contexto, como hoy los pedidos. Con `qdd equipo traer <señal>` baja solo los archivos del contrato y se adapta. Es el
   único paso que gasta tokens, y solo cuando le importa.
6. **Viñeta «Resumen» en el panel:** como la de flujos, con una línea por cambio, por consola y rama, y los ⚡ arriba.
   Cada línea puede pedir 🔔 «avísame cuando esté en git».
7. **«Integrar todo a mi rama»:** trae las ramas de las otras consolas y las mergea en la mía. Los conflictos los resuelve
   mi IA con el resumen como contexto.

### Acordar en el flujo

El flujo ya viaja en vivo al teléfono, y solo se reenvía lo que cambió (huella). Las otras consolas lo leen igual. Cada
nodo lleva un ✓ por participante; con todos, queda acordado y puede generar su tarea (`tarea: <id>`, que ya existe).
Cuando esa tarea termina, es un cambio más: ⚡ si toca un contrato, 📋 si no.

## Fases

El relay decide qué es posible. Cada PC entra solo a su propio canal, porque el canal es un secreto que conocen el PC y
sus teléfonos (RFC 0013). Dos PCs no se ven sin un cambio en el relay, que es de qdd-movile.

| Fase | Alcance | Transporte | Relay |
|---|---|---|---|
| 1 | Consolas del mismo PC: proyectos y worktrees distintos | Archivos en `<QDD_HOME>/equipo/` | Sin cambios |
| 2 | Los PCs de una misma cuenta | Un canal de la cuenta, compartido por sus PCs | Canal nuevo |
| 3 | Otras personas | Un canal de equipo con miembros y permisos | Canal, miembros y plan |

**La fase 1 no sube nada.** En el mismo PC, git trae un commit directo desde la carpeta del otro proyecto
(`git fetch <ruta>`), aunque esa rama no esté en `origin`. Un worktree del mismo repositorio ya comparte los objetos.
El contenido nunca pasa por el relay ni por otro servicio.

Desde la fase 2, el contenido viajaría por git. QDD subiría un ref oculto (`refs/qdd/senales/<id>`) solo con los archivos
del contrato, y el relay llevaría el aviso.

## Costo

- **Eventos:** uno por commit o tarea terminada que toca un contrato, y en la práctica son pocos.
- **Tamaño:** ~1 KB por señal. En las fases 2 y 3 es un mensaje de Realtime. El plan pro de QDD Móvil tiene 120 000 al
  mes y lleva ~9 % del mes.
- **LLM:** cero para detectar, transportar y filtrar. Solo lo usa la consola a la que le importa la señal, y la que la
  originó para escribir una línea.
- **Fase 1:** sin red. Leer las señales nuevas cuesta lo mismo que hoy leer la cola de pedidos.

## Qué revisa doctor

- Contratos declarados con rutas que no existen.
- Una señal ilegible en `<QDD_HOME>/equipo/`: se aparta.
- Un `usa` que nombra un proyecto o contrato que no está en el registro.

## Decisiones abiertas (de la persona)

1. **Alcance:** ¿solo tus consolas o también otras personas? **Recomiendo** empezar por la fase 1 (mismo PC), que no
   necesita al relay, y medir antes de seguir.
2. **Ref oculto (desde la fase 2):** ¿QDD lo sube solo o pregunta? **Recomiendo** que lo suba solo si el proyecto lo
   declara (`equipo.subir_contratos: true`), porque publicar algo sale del PC.
3. **«Integrar todo»:** ¿mergea directo o primero muestra qué trae? **Recomiendo** mostrar primero y mergear con un clic,
   como los pedidos.

## Lo que no hace

- No sincroniza archivos por debajo ni edita la rama de nadie: avisa, y cada consola decide traer.
- No usa LLM para decidir qué es un contrato.
- No lleva código por el relay.
