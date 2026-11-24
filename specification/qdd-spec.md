# Especificación de QDD

Lo que define a QDD, sin importar la IA ni el lenguaje. Cómo está implementado:
[Arquitectura](../docs/framework-architecture.md).

## Principios

| Principio | Cómo se comprueba hoy |
|---|---|
| El conocimiento del proyecto vive en el repositorio | Fichas, findings, tablero y decisiones en `.qdd/` |
| Antes de cambiar un módulo se lee su ficha | Puerta de ficha (Claude Code con el panel) |
| Todo bug es un finding, y todo finding resuelto dice qué lo prueba | `FND-01`, `FND-02` y la puerta de finding |
| Nada se commitea ni se publica sin el gate | `qdd audit` en el CI, puerta de audit, `CI-01` y `CI-02` |
| Las decisiones reservadas a la persona esperan a la persona | Pedidos, fichas en borrador, preguntas de blueprint |
| La versión dice el tamaño del cambio | `VER-01` a `VER-05` |
| Lo que no se midió no se aprueba: se dice | Cobertura por principio en el reporte de `qdd audit` |
| Un contrato público no cambia sin acuerdo | Norma. Lo más cercano es `VER-03`, que mira los commits |

## Ciclo

Contexto → prueba que falla → implementación → `qdd audit` → commit → decisión de la persona → versión → evidencia.

## Dónde se invoca

`qdd audit`, `qdd certify` y los demás comandos corren en la terminal. `qdd_sprint` y `qdd_release` existen solo como tools
MCP. Los `/qdd …` son pedidos en lenguaje natural a la IA, no subcomandos: ver la
[Command Reference](../docs/command-reference.md).
