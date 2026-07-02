# RFC 0002 — QDD Cognitive Layer (QCL)

**Status:** Draft
**Author:** QDD Community
**Created:** 2026-07-01

## Contexto

Durante la evolución del Framework identificamos una limitación importante.

Actualmente el usuario debe decidir qué comando QDD ejecutar y, en muchos casos, debe construir manualmente un prompt para obtener el resultado esperado.

Esto contradice uno de los principios fundamentales de QDD:
> El Framework debe encargarse de la ingeniería del proceso para que el usuario pueda concentrarse únicamente en el problema de negocio.

Por este motivo incorporaremos una nueva capa arquitectónica denominada **QDD Cognitive Layer (QCL)**.

Esta capa será responsable de transformar la intención del usuario en un plan de ejecución completo siguiendo la Specification de QDD.

## Objetivo

El usuario nunca debe escribir prompts.
El usuario únicamente expresa una intención.

Ejemplos:
* "Necesito agregar autenticación."
* "Creo que este endpoint es muy lento."
* "Explícame este proyecto."
* "Necesitamos agregar una nueva funcionalidad."
* "Quiero preparar una nueva versión."
* "Ayúdame a entender este módulo."

QDD deberá comprender la intención, completar automáticamente el contexto necesario y decidir cómo ejecutar la solicitud.

## Nueva Arquitectura

```text
Usuario
  ↓
QDD Cognitive Layer
  ↓
Execution Engine
  ↓
AI Provider
  ↓
Proyecto
```

La IA deja de recibir prompts escritos por el usuario. Recibe un Plan de Ejecución construido por el Framework.

## Responsabilidades del QDD Cognitive Layer

El QCL será responsable de:
* Comprender la intención.
* Detectar objetivos, restricciones y ambigüedades.
* Analizar el estado actual del proyecto (Specification, Runtime, State, Plugins).
* Construir el Plan de Ejecución.
* Solicitar aprobación cuando corresponda.
* Entregar el Plan al Execution Engine.

El QCL **nunca modifica el proyecto**. Su responsabilidad termina cuando el Plan de Ejecución ha sido aprobado.

## Arquitectura Interna

El QCL estará compuesto por módulos especializados siguiendo Clean Architecture:

1. **Intent Analyzer:** Interpreta lenguaje natural, detecta intención y clasifica. (Salida: `Intent Model`)
2. **Context Analyzer:** Lee Runtime, Specification, estado, plugins. (Salida: `Project Context`)
3. **Risk Analyzer:** Detecta riesgos críticos (contratos públicos, endpoints, seguridad). Detiene si hay riesgos.
4. **Strategy Planner:** Determina la mejor secuencia de comandos (e.g. `learn` -> `discover` -> `feature` -> `certify`).
5. **Execution Plan Builder:** Construye el `Execution Plan` estructurado (YAML/JSON).
6. **Approval Manager:** Determina si requiere autorización humana.

## Execution Plan

Contrato independiente del proveedor IA. Ejemplo YAML:
```yaml
goal: Agregar endpoint Cancelar Solicitud
intent: FEATURE
strategy:
  - learn
  - discover
  - feature
  - certify
artifacts:
  - certification
  - tests
  - findings
  - evidence
quality:
  - world-class
compatibility: strict
```

## Compatibilidad y Cognitive Memory

El QCL mantendrá memoria estructurada (decisiones, bugs, arquitectura) y garantizará *Backward Compatibility* e iteración segura.
