# Sprint 9

## Objetivos (Sprint Goal)
En Windows, QDD Móvil (qdd remoto correr) corre desde que la persona inicia sesión sin abrir ninguna ventana de consola, se relanza solo si termina, deja su registro en un archivo con tope que doctor nombra, y un PC con la tarea anterior pasa a la nueva al actualizar qdd o con 'qdd doctor --fix'. Decisión de la persona (2026-11-15): «Lanzador oculto» (qddw.exe), sin ícono en la bandeja por ahora.

## Certification-Driven Development (CDD v1.1)
Este sprint se rige por un contrato estricto de escenarios exhaustivos.
El agente debe implementar el código y los tests para el **Happy Path**, **Error Paths** y **Edge Cases** definidos en `sprint-9-cert.yaml`.

---
*Gobernanza QDD: Este archivo y su YAML asociado guían el desarrollo. Usa 'qdd certify' o ejecuta los tests directamente.*
