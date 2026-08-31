# PROTOCOLO DE CERTIFICACIÓN DE COMPORTAMIENTO HUMANO (QDD HUMAN QA)

## 1. PRINCIPIO DE CERTIFICACIÓN HUMANA
Todo flujo crítico de usuario (autenticación, formularios, navegación, dashboards, pasarelas y portales públicos) debe ser certificado mediante simulación de interacción humana real antes de ser liberado a producción.

## 2. REGLAS DE EJECUCIÓN OBLIGATORIAS
1. **Zero Flakiness (Cero Esperas Fijas Ciegas):**
   - Queda prohibido el uso de `waitForTimeout` arbitrarios para sincronización de elementos.
   - Se debe esperar el estado reactivo del DOM (`waitForSelector`, `waitForFunction`, `waitForURL`, etc.).

2. **Doble Viewport Obligatorio (Desktop & Mobile):**
   - Toda suite de certificación debe ejecutarse validando Viewport Desktop (`1440x900`) y Viewport Mobile (`390x844`).
   - Ningún elemento interactivo clave debe quedar cortado o inaccesible en pantallas táctiles.

3. **Zero-Else & Early Return:**
   - Todo script de aserción y orquestación debe estructurarse con salidas rápidas (`return`), sin sentencias `else`.

4. **Preservación e Idempotencia de Datos:**
   - Las pruebas destructivas deben operar en sandbox o respaldar y restaurar el estado inicial (`DataBackupGuard`).

5. **Auditoría de Usabilidad y Accesibilidad:**
   - Todo flujo debe verificar el ratio de contraste WCAG 2.2 y medir la fricción cognitiva (TTA - Time to Action) y la protección contra clics repetitivos (Rage Clicks Debounce).
