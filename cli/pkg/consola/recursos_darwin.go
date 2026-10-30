package consola

// Rule: Prohibido usar 'else'
// macOS: por ahora solo el disco (statfs). La RAM, la CPU y las GPU necesitan APIs de Mach que Go no expone sin cgo; se
// omiten en vez de inventarlas.

func usoRAM() (float64, bool) { return 0, false }

func leerCPU() (muestraCPU, bool) { return muestraCPU{}, false }

func otrasGPUs() []*float64 { return nil }
