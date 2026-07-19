# Sprint: Certificación Estricta Famas

## 1. Contextualización (Fase 1)
**Objetivo:** Agregar una nueva certificación al corazón del framework QDD que garantice que el agente (Famas) tiene estrictamente prohibido:
1. Escribir fallos silenciosos (errores ignorados, vacíos o no reportados).
2. Generar mocks para flujo de código de ejecución; los mocks solo podrán ser generados y usados dentro de contextos de test (y únicamente cuando la infraestructura real no sea aplicable).

## 2. Persistencia Fractal (Fase 2)
Este documento formaliza el requerimiento de sprint dentro de la arquitectura de la máquina de estados QDD.

## 3. Implementación Planificada (Fase 3 y 4)
- Creación de `CERT-042-FAMAS-STRICT.yaml` en `.qdd/core/certification/`.
- Actualización de Master Governance Rules en:
  - `.cursorrules`
  - `.windsurfrules`
- Actualización de reglas globales inquebrantables en `.agents/workflows/qdd.md`.

## 4. Validación Continua (Fase 5)
Asegurar que estas reglas queden documentadas en los archivos base para guiar todas las ejecuciones futuras de Famas (u otros agentes) en el framework QDD.
