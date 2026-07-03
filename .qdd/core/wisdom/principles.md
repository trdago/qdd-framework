# QDD Engineering Principles (Wisdom Registry)

Estas reglas dictan el comportamiento estricto del Motor Cognitivo y de Ejecución de QDD. Toda planificación, auditoría y código generado debe obedecer incondicionalmente estos principios:

1. **NO-ELSE**: El uso de 'else' está prohibido. Utiliza siempre retornos tempranos (early returns) o cláusulas de guarda.
2. **Timeouts Obligatorios**: Toda integración o consulta a una fuente externa (API, BD, servicio) DEBE tener un control fijo de tiempo (timeout explícito). Nunca asumas que las fuentes externas siempre contestarán.
3. **Trazabilidad de APIS**: Todas las solicitudes a APIs externas deben incluir trazabilidad para garantizar que se sepa exactamente qué pasó y cuándo.
4. **Determinismo Estricto**: Los sistemas NUNCA deben improvisar ni usar caminos por default implícitos. Todos los caminos de ejecución deben ser deterministas y estar explícitamente pensados y codificados.
5. **Contratos Estrictos**: Todo sistema/módulo debe tener un contrato de comunicación claramente establecido, respetado y exigido a lo largo del código.
6. **Auto-TDD ante Bugs**: Todo bug detectado DEBE ser reparado generando primero un test unitario (TDD) que garantice que jamás volverá a pasar (Prevención de Regresión).
7. **Control de Memoria**: Siempre se deben gestionar y controlar las posibles fugas de memoria (limpiar recursos, buffers, conexiones, canales, etc.).
8. **Gobernanza de Contratos**: NINGUNA mejora puede romper contratos públicos automáticamente. Si una solución requiere modificar URL, Endpoint, Payloads, OpenAPI o reglas públicas, se debe solicitar autorización explícita antes de continuar.
9. **Aprendizaje Perpetuo**: Cada bug detectado debe convertirse obligatoriamente en: Un Finding, una nueva prueba, una nueva regla de certificación y una evidencia permanente. Nunca se debe olvidar un bug previamente resuelto.
11. **Certificación y Modo Consultivo**: Todo componente clave debe respaldarse con estándares industriales (OWASP, Clean Architecture, OpenAPI, etc.). La IA siempre debe entrar en modo consultivo y proponer la adopción de estos estándares al detectar nuevas funcionalidades.
12. **Calidad Evolutiva y ADR**: El framework intentará evolucionar constantemente el nivel de madurez del proyecto. Toda certificación aceptada debe registrarse de manera permanente en el conocimiento (ADR) y ser respetada a partir de ese momento.
13. **Inmutabilidad del Core**: La carpeta `.qdd/core/` es de solo lectura y estricta propiedad del framework. NUNCA debes modificar, agregar o eliminar archivos dentro de `.qdd/core/`. Cualquier regla, certificación o conocimiento específico del proyecto debe ir EXCLUSIVAMENTE en la carpeta `.qdd/project/`.
