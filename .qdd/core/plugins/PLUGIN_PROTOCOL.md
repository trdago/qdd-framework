# QDD Plugin Protocol (Manifiesto de Extensibilidad)

Este documento define las directrices y el protocolo oficial para desarrollar, empaquetar e integrar plugins dentro del ecosistema del framework QDD (Quality-Driven Development).

## 1. Filosofía de Arquitectura
La arquitectura de **Plugins** permite a QDD integrarse nativamente con cualquier lenguaje, framework o proveedor de IA sin alterar el binario principal (el motor CLI).
Los plugins pueden funcionar mediante:
- Inyección de dependencias (para integraciones en Go).
- Subprocesos y llamadas RPC (Remote Procedure Call).
- Adaptadores MCP (Model Context Protocol).

Cualquier plugin desarrollado para este ecosistema debe ser tratado como un ciudadano de primera clase, sujeto a las mismas reglas de calidad del núcleo del framework.

## 2. Restricciones Cognitivas (Cognitive Bounds)
Todo plugin integrado en QDD debe adherirse estrictamente al Círculo Virtuoso del framework:
1. **Zero-Else:** Queda estrictamente prohibido el uso de la sentencia `else` en el código fuente del plugin. Se debe emplear siempre *Early Return* (Salida Rápida).
2. **Arquitectura Determinista:** El plugin no debe hacer suposiciones ocultas. Todo debe estar tipado y validado.
3. **Golden Sets Mandatorios:** Si se descubre un bug en un plugin, se debe crear un test unitario y un Golden Set que reproduzca el error para que nunca vuelva a ocurrir.

## 3. Empaquetado del Plugin
No se requieren instaladores complejos. Para proponer un plugin a QDD, el autor debe empaquetar:
- **Código Fuente:** Aislado en su propio directorio (ej. `plugins/<nombre-del-plugin>/`).
- **Certificado de Pull Request (`QDD_PR_CERTIFICATE.md`):** Manifiesto obligatorio donde se declara la naturaleza del plugin y las rutas afectadas.
- **Casos de Uso / Pruebas:** Archivos necesarios para validar el comportamiento del plugin bajo estrés.

## 4. Flujo de Integración (AI-Guided)
La integración de un plugin es orquestada por el arnés cognitivo de QDD mediante el comando nativo `/qdd pr`.

1. El autor llena el certificado `QDD_PR_CERTIFICATE.md`, marcando explícitamente `[x] Plugin`.
2. Se le entrega el certificado a la IA encargada (Antigravity u otra compatible).
3. La IA asimila el certificado mediante `/qdd pr`, audita el código contra las reglas del Core (buscando violaciones al Zero-Else) y ubica el plugin en `.qdd/core/plugins/` (o donde el proyecto lo requiera).
4. La IA ejecuta la regeneración de *Core Assets* (`make core-assets`) para que el framework absorba el nuevo plugin durante su próxima compilación sin dañar el núcleo existente.

---
*Este manifiesto es inviolable y garantiza que el crecimiento del framework QDD se mantenga robusto y guiado por la calidad.*
