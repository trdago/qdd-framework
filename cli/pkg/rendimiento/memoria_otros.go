//go:build !linux && !darwin && !windows

package rendimiento

import "os"

func memoriaDurante(int) *float64 { return nil }

func memoriaAlTerminar(*os.ProcessState) *float64 { return nil }
