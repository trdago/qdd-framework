---
description: QDD Framework documentation generation workflow (Plugin/Skill)
---

# QDD Documentation Generator Protocol
**Plugin:** `qdd_docs_plugin`
**Comando Asimilado:** `/qdd docs`

## Propósito
Extiende el comportamiento de QDD para generar la documentación oficial de un proyecto usando plantillas agnósticas.

## REGLA CRÍTICA: Anti-Olvido (Chunking Obligatorio)
**NUNCA INTENTES GENERAR O PROCESAR EL JSON ESTRUCTURAL COMPLETO DE UNA SOLA VEZ.**
Debes construir el JSON en 4 fases secuenciales:

### Fase 1: Macro Arquitectura (Metadatos)
- `titulo`, `subtitulo`, `fecha`, `tipo_documento`, `institucion`.
- `integrantes` y `resumen_ejecutivo`.

### Fase 2: Contexto Funcional y Core
- Objetivos, Capacidades Operativas, Arquitectura Técnica y Restricciones.

### Fase 3: Micro-Técnico (El Core Pesado)
- Diccionario de Datos (Metadata y Atributos).
- Endpoints (Contratos Técnicos, payloads, queries).
- Gestión de Errores (Códigos HTTP).

### Fase 4: Cierre y Consolidación
- Conclusiones y unificación en `input/informe_dinamico_QDD.json`.

## Ejecución del Motor Documental
Una vez que `input/informe_dinamico_QDD.json` esté completo, debes compilar el documento Word:
```bash
myenv/bin/python .adf/tools/doc_engine.py plantilla/Formato_Agnostico_QDD.docx input/informe_dinamico_QDD.json output/Documentacion_Oficial_QDD.docx
```
