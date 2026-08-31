# QDD Human Certifier Plugin 🎭🧩📱♿

Plugin oficial de **QDD Framework** para la certificación autónoma y modular del comportamiento humano, usabilidad (TTA/Rage Clicks), accesibilidad (WCAG 2.2) y resiliencia en aplicaciones web.

---

## 🌟 Características Principales

1. **Grafo Acíclico Dirigido (DAG):**
   - Resolución automática de dependencias de flujo (ej: autenticación $\rightarrow$ navegación $\rightarrow$ acción profunda).
   - Ordenamiento topológico dinámico sin duplicados.

2. **Servidor Local Nativo Zero-Latency (< 10ms):**
   - Sirve directorios de producción (`dist/`, `build/`, `out/`) de forma ultra-rápida sin dependencias externas ni latencias de red.

3. **Doble Viewport Obligatorio:**
   - Certificación simultánea en **Desktop (1440x900)** y **Mobile (390x844)**.

4. **Micro-delays y Cadencia Natural:**
   - Simulación de ritmo humano de escritura y navegación.

5. **Guardias de Resguardo de Datos (`DataBackupGuard`):**
   - Aislamiento e idempotencia para pruebas destructivas o mutaciones de estado.

---

## 📦 Instalación y Configuración

En tu proyecto gobernado por QDD, crea o ajusta `.qdd/human_qa.config.json`:

```json
{
  "distPath": "dist",
  "port": 5173,
  "baseUrl": "http://localhost:5173",
  "unitsDir": "scripts/human_qa/units",
  "headless": true
}
```

---

## 🚀 Uso

```bash
# 1. Ejecutar el grafo completo
node plugins/qdd-human-certifier/runtime/index.mjs all

# 2. Ejecutar una categoría específica (ej. usabilidad)
node plugins/qdd-human-certifier/runtime/index.mjs usability

# 3. Ejecutar un nodo específico con resolución de dependencias
node plugins/qdd-human-certifier/runtime/index.mjs auth:login_nominal
```

---

## 📜 Licencia y Manifiesto
Desarrollado bajo los estándares del **Manifiesto QDD** (Zero-Else, Early Return y Production-First). Licencia MIT.
