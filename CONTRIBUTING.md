# Contributing to QDD Framework

¡Gracias por tu interés en contribuir a QDD Framework! Al ser una plataforma orientada a la calidad (*Quality-Driven Development*), nuestras reglas de contribución son estrictas para mantener la consistencia y fiabilidad del ecosistema.

## 🧠 Filosofía de Contribución

Antes de enviar código, asegúrate de comprender y aplicar los principios de nuestro marco de trabajo:

1. **Zero-Else & Guard Clauses**: Está prohibido el uso de la palabra clave `else`. Utiliza siempre *Early Returns* (Retornos tempranos) para manejar flujos alternativos o errores. Esto reduce la complejidad cognitiva.
2. **Casos de Borde (Edge Cases)**: Los tests no solo deben probar el "camino feliz". Se deben simular escenarios de fallo (timeouts, valores nulos, estados corruptos). **Todo bug encontrado genera un test unitario obligatorio.**
3. **Pipelining**: QDD Framework soporta comandos en tubería. Si construyes una nueva herramienta, asegúrate de que sea componible de forma secuencial y atómica.
4. **UI Predictiva**: En el Dashboard o interacciones del CLI, evita pedir texto libre al usuario. Usa modales, opciones estructuradas o menús selectivos para evitar ambigüedades.

## 🛠️ Entorno de Desarrollo

Para comenzar a desarrollar en QDD Framework:

1. **Requisitos**: Go (1.24+), Node.js (20+) y npm.
2. **Inicializar**:
   ```bash
   git clone https://github.com/trdago/qdd-framework.git
   cd qdd-framework
   ```
3. **Compilar e Instalar**:
   Ejecuta el siguiente comando para compilar el frontend, el backend y copiar el binario a tu entorno local:
   ```bash
   make install
   ```
4. **Ejecutar Pruebas**:
   Antes de hacer un commit, asegúrate de que todos los tests pasen y no haya violaciones arquitectónicas:
   ```bash
   make test
   make audit
   ```

## 📝 Convenciones de Código y Commits

- **Commits Convencionales**: Utilizamos [Conventional Commits](https://www.conventionalcommits.org/). Ejemplos:
  - `feat: añade soporte para plugins MCP`
  - `fix: resuelve data race en el core engine`
  - `docs: actualiza el diagrama del manifiesto`
- **Changelog**: Cualquier cambio significativo debe reflejarse en el archivo `CHANGELOG.md` al realizar un release.

## 🚀 Proceso de Pull Request (PR)

1. Haz un fork del repositorio y crea tu rama desde `main` (ej. `feat/nueva-herramienta` o `fix/bug-v1`).
2. Ejecuta `/qdd validate` o `make audit` localmente.
3. Envía tu PR detallando claramente el problema que resuelves y los tests que añadiste.
4. El Gatekeeper de CI de QDD validará estructuralmente tu código. Si tu código incluye cláusulas `else`, tu PR será rechazado automáticamente.

## 🛡️ Reporte de Vulnerabilidades

Si encuentras un problema de seguridad, **no abras un issue público**. Por favor, revisa las políticas de seguridad o contacta directamente a los administradores del repositorio para una divulgación responsable.
