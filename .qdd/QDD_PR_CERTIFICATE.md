# QDD Pull Request Certificate (QPR)

> **Instrucciones para el contribuyente:** 
> Rellena este certificado y compártelo como enlace o archivo adjunto al momento de invocar `/qdd pr` o `/qdd pull request`. El Agente de IA leerá este documento, extraerá tu propuesta y la asimilará dentro del core de QDD validando todas las reglas del framework (Zero-Else, Golden Sets, etc.).

## 1. Metadata del Aporte
- **Autor / GitHub Username:** [@username]
- **Tipo de Aporte:** [ ] Core Feature | [ ] Plugin | [ ] Bugfix | [ ] Refactor
- **Nivel de Impacto:** [ ] Bajo | [ ] Medio | [ ] Alto (Arquitectura)

## 2. Descripción de la Propuesta
Describe claramente cuál es la mejora, por qué es necesaria y qué problema soluciona en el framework.

[Escribe aquí tu descripción]

## 3. Archivos Involucrados o Enlaces al Código
Lista aquí los enlaces a tus gists, repositorios, o adjunta los archivos que conforman la contribución. Si tu aporte requiere añadir nuevos archivos, lista la ruta esperada (ej. `plugins/mi-plugin/main.go`).

- **Archivo 1:** [Link o contenido adjunto] -> `Ruta esperada en el proyecto`
- **Archivo 2:** [Link o contenido adjunto] -> `Ruta esperada en el proyecto`

## 4. Promesa de Calidad (QDD Compliance)
Al someter este aporte declaro que el código adjunto / propuesto:
- [ ] No utiliza cláusulas `else` (Zero-Else Policy).
- [ ] Implementa Retornos Tempranos (Early Returns / Guard Clauses).
- [ ] Incluye casos de borde obligatorios si aplica.
- [ ] No hace Mocks para validaciones principales (A menos que sea en testing aislado).

---
*Uso Exclusivo de la IA (No modificar):*
Al recibir este archivo, el agente QDD ejecutará el comando `/qdd pr`, mapeando la intención hacia el **Círculo Virtuoso (Fase 1 a 5)**. En caso de fallas estructurales, el agente intentará auto-reparar el código propuesto para que cumpla los lineamientos, o notificará al autor.
