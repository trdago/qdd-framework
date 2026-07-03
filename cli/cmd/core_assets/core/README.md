# ⚠️ PRECAUCIÓN: NO EDITAR ESTA CARPETA ⚠️

La carpeta `.qdd/core/` contiene el **Universal Core Baseline** del Framework QDD.
Su contenido es de **solo lectura** y es generado/actualizado automáticamente al ejecutar `qdd init`. 

Cualquier modificación manual en esta carpeta será destruida en la próxima actualización del framework.

## ¿Dónde colocar mis reglas?
Si necesitas agregar certificaciones, conocimiento, ADRs, o directivas para tu propio proyecto, **debes colocarlas en la carpeta `.qdd/project/`**. 

El framework QDD leerá y consolidará dinámicamente ambas carpetas (`core` + `project`) durante su ejecución.
