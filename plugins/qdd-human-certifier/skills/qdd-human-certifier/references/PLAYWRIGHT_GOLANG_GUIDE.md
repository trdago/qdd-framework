# Guía de Certificación Humana con Playwright en Golang 🎭🐹

Esta guía define el estándar técnico oficial para la implementación de pruebas y certificaciones de interacción humana utilizando **Golang** y el paquete oficial `github.com/playwright-community/playwright-go`.

---

## 1. 📦 DEPENDENCIAS E INSTALACIÓN

En el `go.mod` de la suite de pruebas o del CLI:

```go
require (
    github.com/playwright-community/playwright-go v0.4201.1
)
```

Instalación de binarios y navegadores de Playwright:
```bash
go run github.com/playwright-community/playwright-go/cmd/playwright@v0.4201.1 install --with-deps
```

---

## 2. 🛡️ PATRONES DE CÓDIGO GO ESTRICTO QDD (Zero-Else & Early Return)

Todo el código de automatización debe respetar rigurosamente:
1. **Cero `else`:** Nunca encadenar `if ... else` o `else if`. Usar salidas rápidas.
2. **Salida más rápida primero:** Validar errores inmediatamente tras cada llamada a Playwright.
3. **Limpieza con `defer`:** Garantizar que páginas, contextos y el driver de Playwright siempre se cierren.

### Ejemplo de Inicialización de Contextos Duales (Desktop & Mobile)

```go
package humanqa

import (
    "fmt"
    "path/filepath"
    "time"

    "github.com/playwright-community/playwright-go"
)

type ExecutionHarness struct {
    PW          *playwright.Playwright
    Browser     playwright.Browser
    DesktopCtx  playwright.BrowserContext
    MobileCtx   playwright.BrowserContext
    EvidenceDir string
    TraceID     string
}

func NewExecutionHarness(evidenceDir, traceID string) (*ExecutionHarness, error) {
    if evidenceDir == "" {
        return nil, fmt.Errorf("evidenceDir no puede estar vacío")
    }

    pw, err := playwright.Run()
    if err != nil {
        return nil, fmt.Errorf("fallo iniciando Playwright: %w", err)
    }

    browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
        Headless: playwright.Bool(true),
    })
    if err != nil {
        pw.Stop()
        return nil, fmt.Errorf("fallo lanzando Chromium: %w", err)
    }

    // Configuración Desktop: 1440x900
    desktopCtx, err := browser.NewContext(playwright.BrowserNewContextOptions{
        Viewport: &playwright.Size{Width: 1440, Height: 900},
        RecordVideo: &playwright.RecordVideo{
            Dir: filepath.Join(evidenceDir, "videos", "desktop"),
        },
    })
    if err != nil {
        browser.Close()
        pw.Stop()
        return nil, fmt.Errorf("fallo creando contexto desktop: %w", err)
    }

    // Configuración Mobile: 390x844 (iPhone 14/15)
    mobileCtx, err := browser.NewContext(playwright.BrowserNewContextOptions{
        Viewport:          &playwright.Size{Width: 390, Height: 844},
        DeviceScaleFactor: playwright.Float(3.0),
        IsMobile:          playwright.Bool(true),
        HasTouch:          playwright.Bool(true),
        RecordVideo: &playwright.RecordVideo{
            Dir: filepath.Join(evidenceDir, "videos", "mobile"),
        },
    })
    if err != nil {
        desktopCtx.Close()
        browser.Close()
        pw.Stop()
        return nil, fmt.Errorf("fallo creando contexto mobile: %w", err)
    }

    return &ExecutionHarness{
        PW:          pw,
        Browser:     browser,
        DesktopCtx:  desktopCtx,
        MobileCtx:   mobileCtx,
        EvidenceDir: evidenceDir,
        TraceID:     traceID,
    }, nil
}

func (h *ExecutionHarness) Close() {
    if h.DesktopCtx != nil {
        h.DesktopCtx.Close()
    }
    if h.MobileCtx != nil {
        h.MobileCtx.Close()
    }
    if h.Browser != nil {
        h.Browser.Close()
    }
    if h.PW != nil {
        h.PW.Stop()
    }
}
```

---

## 3. 📸 CAPTURA DE EVIDENCIA EN CADA PASO

Cada paso relevante debe capturar screenshots y registrar eventos de red en la carpeta de evidencia asignada al run:

```go
func CaptureStepScreenshot(page playwright.Page, evidenceDir, stepName, viewport string) error {
    if page == nil {
        return fmt.Errorf("página no inicializada")
    }

    filename := fmt.Sprintf("%s_%s_%d.png", stepName, viewport, time.Now().UnixMilli())
    targetPath := filepath.Join(evidenceDir, "screenshots", filename)

    _, err := page.Screenshot(playwright.PageScreenshotOptions{
        Path:     playwright.String(targetPath),
        FullPage: playwright.Bool(true),
    })
    if err != nil {
        return fmt.Errorf("error capturando screenshot %s: %w", filename, err)
    }

    return nil
}
```

---

## 4. ⚡ INTERCEPTOR DE RED Y LOGS DE CONSOLA

Registra todas las peticiones fallidas o respuestas anómalas:

```go
func AttachNetworkInspector(page playwright.Page, logFile string) {
    if page == nil {
        return
    }

    page.On("response", func(res playwright.Response) {
        status := res.Status()
        if status < 400 {
            return
        }

        // Registro de error de red HTTP 4xx / 5xx
        logEntry := fmt.Sprintf("[%s] HTTP %d: %s\n", time.Now().Format(time.RFC3339), status, res.URL())
        appendToFile(logFile, logEntry)
    })

    page.On("console", func(msg playwright.ConsoleMessage) {
        if msg.Type() != "error" {
            return
        }

        logEntry := fmt.Sprintf("[%s] CONSOLE ERROR: %s\n", time.Now().Format(time.RFC3339), msg.Text())
        appendToFile(logFile, logEntry)
    })
}
```

---

## 5. 🎯 EJECUCIÓN AUTÓNOMA Y EVALUACIÓN

El script en Go empaqueta toda la evidencia en la carpeta de ejecución y genera un `manifest.json`. Al finalizar la ejecución, el **LLM es quien analiza las capturas, verifica los diffs en base de datos y emite la certificación final**.
