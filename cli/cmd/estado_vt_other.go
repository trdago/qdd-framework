//go:build !windows

package cmd

// habilitarVT no hace nada fuera de Windows: las terminales ya interpretan ANSI.
func habilitarVT() {}
