# Política de seguridad

## Reportar una vulnerabilidad

No abras un issue público. Usa el formulario privado de GitHub: pestaña **Security** →
[**Report a vulnerability**](https://github.com/trdago/qdd-framework/security/advisories/new). Solo lo ven quienes
mantienen el repositorio.

Cuenta qué versión usas (`qdd version`), en qué sistema, cómo se reproduce y qué permitiría hacer. No incluyas credenciales
ni datos de otras personas.

## Versiones con soporte

Las correcciones salen en la última versión de `latest` y en la última `-rc` del canal `preview`.

## Qué cuida QDD

- Los tokens del baúl quedan en el llavero del sistema o en 1Password; no se imprimen ni llegan a la IA.
- Las credenciales de QDD Móvil viven en `QDD_HOME` con permisos `0600`; una clave secreta del relay se rechaza.
- Lo que una IA no debe ejecutar queda como pedido y corre solo con la confirmación de una persona.
- Lo que sale de tu máquina está en el [README](README.md#qué-sale-de-tu-máquina).
