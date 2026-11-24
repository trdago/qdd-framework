# Contribuir a QDD

Gracias por querer mejorar QDD. Este repositorio se desarrolla con QDD: pasa su propio gate y sigue su propio método.

## Entorno

- **Go 1.25+**, **Node 20.19+ o 22.12+** y npm.
- La suite del panel usa la CLI de Claude Code; sin ella esa suite queda «sin medir», no fallida.

```bash
git clone https://github.com/trdago/qdd-framework.git
cd qdd-framework
npm install     # instala los hooks de git
make build      # deja ./qdd en la raíz
make test
```

`make install` y `make audit` **reemplazan el `qdd` de tu equipo** (`~/.local/bin`) por el de esta copia. Para probar sin
tocarlo, usa `./qdd audit` después de `make build`.

## De qué rama partir

- Los arreglos para la versión estable salen del último tag estable y van a `main`.
- Lo nuevo va a la rama de integración de la próxima versión: la base del PR abierto más grande hacia `main`, o pregunta en
  un issue. El CI corre en los PR hacia `main` y hacia `feat/**`.

## Cómo se trabaja

1. **Lee la ficha** del módulo que vas a tocar: `qdd modules context <rutas>`. Si tu cambio altera una decisión o una
   invariante, actualiza la ficha en el mismo PR.
2. **Pruebas primero.** Comprueba que la prueba falla antes del cambio. Cubre también los caminos de error.
3. **Cero `else`, salida temprana.** Cada vez menos personas leen el código línea a línea, así que el que sí se lee tiene
   que entenderse de un vistazo.
4. **Todo bug es un finding** en `.qdd/project/findings/`, con su prueba de regresión en `test_result`. Un commit `fix:` lo
   nombra.
5. **Documenta en el mismo PR** toda capacidad nueva; una prueba compara la referencia con el binario.
6. **Sin nombres reales** de organizaciones, clientes o proyectos en la documentación: el repositorio es público.
7. **`qdd audit` en verde** antes de cada commit.

Lo temporal va en `./tmp`, que git ignora.

## Commits y versiones

- [Conventional Commits](https://www.conventionalcommits.org/): `feat:`, `fix:`, `docs:`, `test:`, `chore:`…
- La versión la define el contenido: un `feat` pide MINOR y `!` o `BREAKING CHANGE:`, MAJOR (`VER-03`). Se declara en
  `package.json`, `npm/package.json` y `cli/cmd/version.txt`, con su entrada en el [CHANGELOG](CHANGELOG.md).
- Los cambios grandes empiezan por un [RFC](rfcs/README.md).

## Pull Request

Cuenta qué problema resuelve, qué cambia para quien usa QDD y cómo lo verificaste. Una persona que mantiene el proyecto lo
integra cuando el CI está en verde.

## Reportar un problema

- **Con GitHub:** un [issue](https://github.com/trdago/qdd-framework/issues).
- **Sin GitHub:** `qdd reportar --titulo "…" --verifico "…"`; ves el texto exacto antes de que salga. Quien reporta verifica
  el arreglo.
- **De seguridad:** nunca en público; sigue [SECURITY](SECURITY.md).
