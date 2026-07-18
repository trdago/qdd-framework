# RFC 0001: Diseño del comando `qdd init`

**Status:** Draft
**Author:** [Tu Nombre/Equipo]
**Created:** 2026-07-01

## 1. Summary

El comando `qdd init` es el punto de entrada al QDD Framework para cualquier proyecto existente. Su propósito principal no es solo crear carpetas vacías, sino "entender" el entorno sobre el cual se está ejecutando para adaptar el Runtime y la Especificación de manera automática.

## 2. Motivation

Actualmente, inicializar herramientas de IA en un repositorio suele requerir que el usuario configure manualmente prompts, variables de entorno, ignore files, y determine qué herramientas están disponibles. 

En QDD, queremos que la adopción sea inmediata y precisa (AI-Native). Si ejecutamos `qdd init` en un proyecto en Go, el framework debe saber que utilizará los plugins y templates de Go. Si hay un `docker-compose.yml`, debe deducir la arquitectura de servicios.

## 3. Design

### 3.1. Detección de Entorno
Al ejecutar `qdd init`, la CLI realizará un escaneo pasivo del repositorio:
- **Lenguaje:** Escaneo de archivos (e.g. `go.mod` -> Go, `package.json` -> Node, `pom.xml` -> Java).
- **Arquitectura:** Detección de `Dockerfile`, directorios `cmd/`, `src/`, `app/`, etc.
- **Bases de Datos:** Detección de ORMs (Prisma, Gorm) o archivos de migración.
- **Proveedores Cloud:** Detección de `serverless.yml`, `.aws/`, `terraform/`, etc.
- **Raíz de Código:** Detección de la convención de código ya existente (`src/`, `cmd/`+`pkg/` en Go, `app/`, etc.). Si no existe una convención clara, `qdd init` DEBE proponerla explícitamente en Modo Consultivo (ej. `src/`) antes de generar nada. El código de aplicación nunca debe terminar dentro de `.qdd/` — ver CERT-033-PROJECT-STRUCTURE-SEPARATION.

### 3.2. Generación del Estado Inicial
Una vez que el entorno es detectado, la CLI creará el directorio `.qdd/` en la raíz del proyecto.

```text
.qdd/
├── config.yaml          # Configuración base detectada
├── runtime/             # Scripts e integraciones locales
├── specification/       # Especificación QDD copiada localmente
├── templates/           # Templates adaptados al lenguaje detectado
├── plugins/             # Plugins iniciales habilitados
├── findings/            # Directorio vacío para futuros findings
├── certification/       # Directorio de certificación
├── evidence/            # Evidencia de bugs y tests
├── metrics/             # Archivos de métricas base
├── dashboard/           # UI o reportes de estado
└── state.json           # Estado actual del entorno
```

### 3.3. Interacción del Usuario
El comando debería operar de manera silenciosa por defecto o con un output claro de lo que ha detectado.

Ejemplo:
```bash
$ qdd init
[+] Detectado: Lenguaje Go (go.mod)
[+] Detectado: Infraestructura AWS (terraform/)
[+] Inicializando directorio .qdd/
[+] Descargando QDD Specification v0.1...
[+] Configurando plugins: [go, terraform, aws]
[!] QDD inicializado exitosamente. Siguiente paso: conecta tu IDE con IA vía MCP y pídele que ejecute la tool `qdd_learn`
```

## 4. Drawbacks

- El escáner puede volverse complejo si hay monorepos con múltiples lenguajes. Tendremos que definir un comportamiento por defecto para monorepos.
- Descargar la especificación requiere conexión a internet, por lo que el comando debería tener soporte offline con una versión embebida de los templates.

## 5. Alternatives

- **Configuración manual:** Pedir al usuario que conteste una serie de preguntas por terminal (estilo `npm init`). Decidimos rechazarlo por defecto porque va en contra de la automatización total (AI-Native), aunque podría estar disponible con un flag `--interactive`.

## 6. Unresolved Questions

- ¿Cómo gestionaremos las actualizaciones de `.qdd/specification/` si el framework se actualiza?
- ¿Qué información exacta se almacenará en `state.json` el día 1?
