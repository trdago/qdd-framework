# Guía de Simulación de Comportamiento Humano

## 1. Micro-Delays y Cadencia Natural
Para evitar ser detectado por mecanismos anti-bot y validar animaciones de UI:
- Tiempos de escritura aleatorios por tecla (30ms a 75ms).
- Micro-pausas entre acciones complejas (100ms a 250ms).

## 2. Validación de Viewports
- **Desktop:** `1440x900`, `devicePixelRatio: 1`, cursor con soporte hover.
- **Mobile:** `390x844` (iPhone 13/14/15), `devicePixelRatio: 3`, eventos de toque `hasTouch: true`, `isMobile: true`.

## 3. Detección de Fricción Cognitiva (TTA)
El TTA (*Time to Action*) mide la cantidad de tiempo y clics que le toma a un usuario completar un objetivo crítico.
Ratios aceptables:
- Login nominal: < 3.0s
- Formulario de acción primaria: < 5.0s
