//go:build !linux && !darwin && !windows

package consola

// Rule: Prohibido usar 'else'
// Otros sistemas: no se mide nada; la barra sale sin la línea de recursos.

func usoDisco(string) (float64, bool) { return 0, false }

func usoRAM() (float64, bool) { return 0, false }

func leerCPU() (muestraCPU, bool) { return muestraCPU{}, false }

func otrasGPUs() []*float64 { return nil }
