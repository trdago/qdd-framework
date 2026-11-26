# RFC

El diseño de cada capacidad grande: el problema, las opciones y lo que se decidió. Un RFC explica **por qué**; qué hace hoy
cada comando está en la [Command Reference](../docs/command-reference.md). No existe el 0008.

| RFC | Tema | Cómo está en la 2.10 |
|---|---|---|
| [0001](0001-qdd-init.md) | Diseño de `qdd init` | Implementado |
| [0002](0002-qdd-cognitive-layer.md) | Capa cognitiva | Reemplazado: el motor interno nunca se construyó |
| [0003](0003-blueprints.md) | Blueprints | Implementado; la fase 3, en parte |
| [0004](0004-blueprints-centrales.md) | Blueprints centrales | Implementado |
| [0005](0005-reglas-como-principios.md) | Reglas como principios | En parte: fase 1 (cobertura) |
| [0006](0006-estado-por-frente-en-consola.md) | Estado y consola | Implementado |
| [0007](0007-fuentes-de-blueprints.md) | Varias fuentes de blueprints | Implementado |
| [0009](0009-blueprints-que-reemplazan.md) | Un blueprint que reemplaza a otro | Implementado |
| [0010](0010-uso-medido-y-preview.md) | Uso medido y canal preview | Implementado |
| [0011](0011-herramientas-en-el-panel.md) | Herramientas en el panel | Implementado |
| [0012](0012-grafo-de-trabajo.md) | Grafo de trabajo | En parte: ver, editar y comentar. Falta la fase 4 |
| [0013](0013-consola-remota.md) | QDD Móvil | Implementado |
| [0014](0014-panel-en-antigravity.md) | Panel en Antigravity y VS Code | Implementado |
| [0015](0015-modo-goal.md) | Modo goal | Implementado |
| [0016](0016-panel-en-go-evaluacion.md) | ¿El panel en Go? | Cerrado: se adoptó el flujo de datos, no el panel |
| [0017](0017-tablero-que-no-se-queda-atras.md) | Un tablero que no se queda atrás | Implementado |
| [0018](0018-comparar-proyectos-en-el-panel.md) | Comparar proyectos y dejar obsoleto el dashboard | En parte: fases 1 a 3. Falta la 4 |
| [0019](0019-turno-unico.md) | Turno único | Implementado |
| [0020](0020-probar-solo-lo-necesario.md) | Probar solo lo necesario | Implementado |
| [0021](0021-colaboracion-en-vivo.md) | Colaboración en vivo entre consolas | Aceptado, sin implementar |
| [0022](0022-rendimiento-estandar.md) | Rendimiento estándar | Implementado |
| [0023](0023-baul-de-accesos.md) | Baúl de accesos y ambientes | En parte: baúl y plan de DNS. `ambiente levantar` todavía no cambia nada |
| [0024](0024-reportar-sin-github.md) | Reportar sin GitHub | Implementado |
| [0025](0025-flujo-para-clientes.md) | Flujos para clientes | En parte: fases 1 a 4. Falta el audio |
| [0026](0026-qdd-remove.md) | `qdd remove`: quitar QDD de la máquina | Propuesto: el código, después de la 2.10.0 |

## Proponer uno

Copia la forma de uno reciente: estado, origen, resumen, problema, diseño, alternativas, lo que no entra y fases. Pasa a
aceptado cuando lo decide una persona que mantiene el proyecto. Al cambiar de estado, actualiza su encabezado y esta tabla.
