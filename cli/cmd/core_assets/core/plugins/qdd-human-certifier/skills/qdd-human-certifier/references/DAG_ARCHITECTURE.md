# Arquitectura del Grafo Acíclico Dirigido (DAG) en QDD Human QA

## 1. Concepto
Las pruebas de usuario complejas dependen secuencialmente del estado anterior (ej. no se puede probar una tabla de métricas sin haber iniciado sesión).
El orquestador de grafo de QDD resuelve este problema mediante un ordenamiento topológico automático basado en el algoritmo de Kahn / DFS.

## 2. Definición de Unidad de Prueba
Cada unidad exporta un objeto estándar:

```javascript
export default {
  id: 'dashboard:metrics_table',
  category: 'dashboard',
  description: 'Certifica la visualización y paginación de la tabla de métricas',
  dependencies: ['auth:login_nominal'],
  async run({ page, mobilePage, context, config }) {
    // Lógica de interacción y aserciones
    return { status: 'PASS', durationMs: 120 };
  }
};
```

## 3. Resolución de Subgrafos
Cuando se invoca un nodo específico (ej. `node index.mjs dashboard:metrics_table`), el orquestador:
1. Extrae transitivamente todos los ancestros (`auth:login_nominal`).
2. Genera el orden lineal topológico de ejecución.
3. Ejecuta cada nodo pasando el contexto acumulado.
4. Si un nodo falla, cancela los descendientes dependientes con estado `SKIPPED_DEP_FAILED`.
