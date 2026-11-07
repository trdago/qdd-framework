# Docker en desarrollo: solo para la base de datos

En QDD la app corre directo en el PC mientras se desarrolla: el frontend con su servidor (por ejemplo, vite) y el backend con su comando (por ejemplo, `go run`). Docker queda para la base de datos.

Quien igual quiera la app en Docker puede hacerlo. `qdd audit` avisa con `DEV-01-APP-EN-DOCKER` y no bloquea. Este documento explica por qué se vuelve lento, para decidir con esa información.

No es un problema de Docker: en producción un contenedor no tiene estos costos, porque el código no cambia a cada rato. El costo aparece en el desarrollo, donde todo gira alrededor de cambiar un archivo y ver el resultado.

## Por qué se vuelve lento

### 1. El código y el contenedor viven en dos sistemas distintos

En Windows y en Mac, los contenedores de Linux corren dentro de una máquina virtual y el código suele estar fuera de ella. Cada archivo que la app lee tiene que cruzar esa frontera, y compartir muchos archivos sube el uso de CPU y hace lento todo el acceso a disco, según [la documentación de Docker](https://docs.docker.com/get-started/docker-concepts/running-containers/sharing-local-files/).

Docker recomienda en Windows guardar el código en el sistema de archivos de Linux, porque desde allí el acceso es mucho más rápido que desde `C:`: [buenas prácticas de WSL 2](https://docs.docker.com/desktop/features/wsl/best-practices/).

Un proyecto real lo sufre más porque su carpeta `node_modules` trae miles de archivos pequeños, y compilar o arrancar la app los lee.

### 2. La recarga automática deja de enterarse

Sin Docker, guardas un archivo y la pantalla cambia. Eso depende de que el sistema avise que el archivo cambió. Docker aclara que en Windows ese aviso solo llega al contenedor si el archivo está en el sistema de archivos de Linux ([misma guía](https://docs.docker.com/desktop/features/wsl/best-practices/)).

La solución habitual es que la herramienta revise los archivos una y otra vez, y [Vite advierte que eso gasta mucha CPU](https://vite.dev/config/server-options). Entonces se paga de una de dos formas: recargando a mano o con el PC ocupado revisando archivos.

### 3. Hay que reconstruir la imagen

Si el código se copia dentro de la imagen, cada cambio pide reconstruirla. Docker reutiliza lo que no cambió, pero [al cambiar una capa rehace todas las que vienen después](https://docs.docker.com/build/cache/optimize/). Según cómo esté escrito el Dockerfile, un cambio de una línea puede volver a instalar dependencias.

### 4. Se pierde lo que ya estaba listo

Directo en el PC, las herramientas reutilizan lo que ya compilaron y descargaron. En un contenedor esas cachés se pierden al recrearlo, salvo que se configuren volúmenes para cada una. Además, los paquetes con código nativo [no se pueden compartir entre sistemas](https://docs.docker.com/compose/how-tos/file-watch/), así que las dependencias se instalan por separado para el contenedor.

### 5. La máquina virtual ocupa memoria y CPU

En Windows y en Mac esa máquina virtual compite con el editor, el navegador, las pruebas y la IA por los mismos recursos.

### 6. Con una IA, la demora se multiplica

Una persona prueba un cambio cada varios minutos. Una IA cambia, prueba y corrige muchas veces seguidas, así que cada segundo de espera se repite decenas de veces en una misma tarea.

## Por qué la base de datos sí

Una base de datos no tiene ese problema, porque su código no es el que estás cambiando:

- Arranca una vez y queda corriendo, sin recargas ni imágenes que reconstruir.
- No necesita tus archivos: sus datos pueden vivir dentro de la máquina virtual. Docker dice que [así el rendimiento es mucho mejor](https://docs.docker.com/desktop/features/synchronized-file-sharing/).
- A cambio, da un entorno limpio y la misma versión que producción, sin instalar nada en el PC.

Si prefieres no tener Docker, una base de datos portable también sirve.

## Qué tan lento es

Depende del sistema y del proyecto, y QDD todavía no tiene una medición propia. Lo anterior viene de la documentación citada y del funcionamiento de estas herramientas. El peor caso es Windows con el código en `C:`. En Linux con Docker Engine no hay máquina virtual, así que los puntos 1, 2 y 5 casi desaparecen, pero los puntos 3 y 4 siguen.

## Si igual lo necesitas

Algunos casos lo justifican, como una dependencia del sistema que no se puede instalar en el PC. Para que duela menos:

- En Windows, deja el código dentro de WSL, no en `C:`.
- Deja `node_modules` y las cachés en volúmenes, no compartidos con el PC.
- Monta el código en vez de copiarlo a la imagen.
- En el Dockerfile, pon primero lo que menos cambia.
- Corre en Docker solo la parte que lo necesita, y el resto directo en el PC.

## Qué revisa QDD

`qdd audit` avisa cuando encuentra:

- Un `docker-compose.yml` o `compose.yaml` con un servicio que construye la imagen (`build`) o monta el código del proyecto, salvo que sea de datos (Postgres, MySQL, Redis y similares).
- Un `.devcontainer`.

Da un solo aviso por proyecto. No avisa por un Dockerfile, que puede ser solo para producción, ni por un compose de producción (`docker-compose.prod.yml`), de despliegue (dentro de `deploy/`, salvo `.local` o `.dev`) o de pruebas (dentro de `e2e/` o `tests/`).

Tiene dos límites. Un `docker-compose.yml` con un nombre común que solo se usa para desplegar también avisa, porque el archivo no dice para qué se usa: renómbralo a `docker-compose.prod.yml`. Y no ve un `docker run` dentro de un script: para eso, QDD le pide a la IA que avise antes de proponerlo.
