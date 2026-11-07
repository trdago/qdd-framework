# RFC 0020: probar solo lo necesario

**Estado:** aceptado (2026-11-07). La persona pidió mantener la calidad y ganar velocidad de desarrollo.

## Problema

`qdd audit` corre todas las suites en cada cambio. Si cambia el color de un botón, no hace falta repetir las pruebas de reglas de negocio. La espera se paga en cada commit, y más cuando una IA itera.

## Lo que se midió primero

Medido el 2026-11-07 en este repositorio, sobre los últimos 300 commits. Cambió el plan: lo más lento no era correr pruebas de más.

| Hallazgo | Dato |
|---|---|
| Cuatro pruebas lanzaban un agente real en cada `go test` (FND-172) | 127 s de 180 s de pruebas de Go; 99 de los 100 s de `cmd` |
| Go sin caché | 101 s antes del arreglo, 33 s después |
| `cmd`, que se repite con casi cualquier cambio en Go | 107 s antes, 2,6 s después |
| La caché de Go solo vigila `cli/` (FND-173) | Una prueba de docs pasó desde la caché con el doc roto, y falló con `-count=1` |
| Suite del plugin | 18 a 31 s; 42 de 300 commits tocan su carpeta |
| npm y OpenCode | 0,5 s y 1,4 s: no vale la pena saltarlas |

## Resultado

`qdd audit` en este repositorio, medido el 2026-11-07. Cada corrida lanzada a mano, con la suite del plugin declarando sus `rutas`. Antes, Go sin caché sumaba 101 s y la suite del plugin 18 a 31 s.

| Caso | Tiempo | Suite del plugin |
|---|---|---|
| Nada cambió | 9 s | no se repite |
| Cambio solo en Go | 13 s | no se repite |
| Cambio en el panel, como un color | 31 s | se repite |
| `--completo` | 27 a 51 s | se repite |

## Decisiones

1. **Primero, lo que no se salta nada.** Se quitaron los agentes reales de las pruebas y se cerró el agujero de la caché.
2. **Se decide por el contenido, no por una comparación entre ramas.** Una suite declara `rutas` en `test_commands`. Si ya pasó con la misma huella (comando, rutas, sistema operativo y contenido de sus archivos), no se repite. Es lo que Go hace con sus paquetes, llevado a cualquier suite y lenguaje.

   ```yaml
   test_commands:
     - comando: npm run test:ui
       rutas: [src/ui]
     - comando: pytest tests/reglas
       rutas: [src/reglas, src/modelos]
   ```

3. **Ante la duda, corre.** Corre sin rutas, con una ruta sin archivos, fuera de git, si lo recordado es ilegible o tiene más de 24 h, en el CI, con `qdd audit --completo` y al certificar. Una suite que falla nunca se recuerda.
4. **Lo que se salta se dice.** `[QA] SIN CAMBIOS: <suite> (pasó a las HH:MM con estos mismos archivos…)`.

## Lo que se descartó

- **Comparar contra la rama base.** Aquí `main` está muy atrás y casi todo parecería cambiado.
- **Usar los comandos de las fichas.** Están escritos a mano: en el arreglo del relay, el filtro de la ficha `remoto` alcanzaba 3 de las 41 pruebas de esos archivos, y 104 de 300 commits tocaban algún archivo sin ficha.
- **Marcar el resultado como «aprobado parcial».** Una suite saltada ya pasó con esos mismos archivos: no hay nada sin probar por ese cambio.

## Límites

- **`rutas` lo declara una persona o una IA.** Una suite que dependa de algo que no declara no se repetirá cuando eso cambie. La red de seguridad es el CI, que corre todo antes de mergear y publicar, salvo una suite cuya herramienta no instale: esa queda «sin medir». Por eso el CI instala Claude Code (FND-174). Hasta este arreglo, la suite del panel no corría en el CI.
- **Las dependencias también son rutas.** Si una suite usa un lockfile o una configuración fuera de su carpeta, se declaran: `rutas: [src/ui, package-lock.json]`. Los archivos ignorados por git no cuentan, y `node_modules` es uno de ellos.
- **Por archivo no se distingue un color de una regla.** Si estilos y lógica viven en un archivo, ambos cambian la huella. Se separan en suites distintas solo si viven en archivos distintos.
- **Una herramienta puede cambiar sin que cambie un archivo.** Por eso lo recordado vence a las 24 h.
- **Otros lenguajes pueden tener el mismo agujero de caché que FND-173.** Un proyecto que use la caché de su ejecutor tiene que declarar lo que esas pruebas leen.

## Pendiente

- **Elegir pruebas dentro de una suite** (por ejemplo, `jest --findRelatedTests`): se declara como comando del proyecto, sin regla nueva de QDD.
- **Pruebas inestables:** la suite del plugin falló 2 pruebas por tiempo de espera en una corrida de la medición y pasó en la siguiente, sin cambios. Saltarla menos veces no la arregla.
