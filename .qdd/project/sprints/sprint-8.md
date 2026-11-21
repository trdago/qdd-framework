# Sprint 8

## Objetivos (Sprint Goal)
Toda pregunta a la persona, de cualquier tipo, aparece siempre en «▶ Para ti» del panel (y en QDD Móvil, que lee el mismo JSON) mientras sigue abierta, y sale cuando se responde o deja de esperarse. Lo fija una prueba por tipo y un contrato que falla si aparece un tipo nuevo sin la suya (MEJ-057, MEJ-026).

## Certification-Driven Development (CDD v1.1)
Este sprint se rige por un contrato estricto de escenarios exhaustivos.
El agente debe implementar el código y los tests para el **Happy Path**, **Error Paths** y **Edge Cases** definidos en `sprint-8-cert.yaml`.

---
*Gobernanza QDD: Este archivo y su YAML asociado guían el desarrollo. Usa 'qdd certify' o ejecuta los tests directamente.*
