# QDD Agentic Loop (Certification-Driven Development)

## Propósito
Este documento define el protocolo estricto que cualquier Agente de IA debe seguir cuando un usuario solicita iniciar un sprint (`/qdd sprint`) o ejecutar tareas dentro del marco QDD.

## Fases del Bucle (The CDD Loop)

### 1. Inicialización (Gatekeeper)
Cuando el usuario pide "crear un sprint":
1. **NO ejecutes** el tool `qdd_sprint` inmediatamente si la solicitud es vaga o ambigua (ej: "haz un login").
2. Debes **preguntar** al usuario por:
   - El objetivo principal (goal).
   - El Happy Path (el flujo ideal).
   - Los Error Paths (caminos de fallo, estricto requerirlos).
   - Edge Cases opcionales (casos límite o de estrés).
   - Restricciones técnicas.
   *(Nota: Como IA, debes generar los 'executable_tests' usando tu conocimiento del proyecto, NO se los preguntes al usuario).*
3. **UX de Opciones Múltiples:** Para agilizar el proceso, NO hagas preguntas abiertas al usuario. **PROPÓN TÚ** las opciones. Usa tu capacidad para generar 2 o 3 alternativas viables (para el Happy Path, Error Paths, etc.) y preséntalas como una lista enumerada o usando tu herramienta nativa de opciones múltiples, incluyendo siempre la opción "Otro (escribir)". El usuario solo debería tener que hacer clic o responder "opción 1" la mayor parte del tiempo.
4. **Manejo de Tareas en Paralelo y Contexto UX:** Si estás manejando múltiples tareas, requerimientos o sprints en la misma conversación (en paralelo), **NUNCA** hagas una pregunta genérica. Siempre que necesites que el usuario realice una acción, seleccione una opción o entregue una respuesta, **debes decir explícitamente a qué requerimiento, funcionalidad o sprint te estás refiriendo** (Ej: `[Sprint 2: Login UI] ¿Prefieres la opción A o B?`). Esta habilidad de "contextualización" es obligatoria para garantizar una excelente experiencia de usuario sin importar qué LLM se utilice (Antigravity, Claude, Cursor, etc).
5. Una vez que el usuario te dé esta información de forma clara o seleccione una opción, **ejecuta el tool `qdd_sprint`** con todos los argumentos correspondientes.

### 2. Ejecución Autónoma (El Loop de Mutación)
Tras la ejecución del tool, se generará un archivo YAML (`.qdd/project/sprints/sprint-X-cert.yaml`) con estado `FAILING`.
1. **Lee** todos los escenarios (`happy_path`, `error_paths`, `edge_cases`) y restricciones técnicas del YAML.
2. **Mutación:** Escribe o modifica el código fuente necesario. **IMPORTANTE:** Debes programar el código y escribir tests automatizados para CADA UNO de los escenarios descritos, no solo para el Happy Path.
3. **Auditoría (Validación):** Ejecuta estrictamente los comandos definidos en la sección `executable_tests` del YAML (por ejemplo: `go test`, `npm run test`, `qdd audit`).

### 3. Self-Correction (Auto-corrección)
- **Si el test falla:** NO pidas ayuda al usuario. Analiza el error del log de consola, inyéctalo en tu contexto, planifica una corrección y vuelve a mutar el código (Paso 2).
- Continúa este bucle (Código -> Test -> Falla -> Corrección -> Test) de forma autónoma.

### 4. Certificación
- **Si el test pasa:** El objetivo del sprint está cumplido.
- Informa al usuario que el loop ha finalizado exitosamente y que el certificado está en estado `CERTIFIED`.
- Invita al usuario a ejecutar `/qdd release` si corresponde.

**Recuerda:** En QDD, el Agente IA está gobernado por el certificado, no por su propia interpretación de la tarea. El certificado (YAML) es la única fuente de verdad y de término de tarea.
