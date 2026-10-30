package consola

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// Rule: Prohibido usar 'else'
// Windows: disco con GetDiskFreeSpaceEx, RAM con GlobalMemoryStatusEx y CPU con GetSystemTimes (el tiempo de kernel
// incluye el ocioso). Las GPU que no son NVIDIA no tienen un contador simple: se omiten.

var (
	kernel32             = windows.NewLazySystemDLL("kernel32.dll")
	globalMemoryStatusEx = kernel32.NewProc("GlobalMemoryStatusEx")
	getSystemTimes       = kernel32.NewProc("GetSystemTimes")
)

type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

func usoDisco(dir string) (float64, bool) {
	p, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		return 0, false
	}
	var disponible, total, libre uint64
	if err := windows.GetDiskFreeSpaceEx(p, &disponible, &total, &libre); err != nil {
		return 0, false
	}
	return porcentajeUsado(total, libre, disponible)
}

func usoRAM() (float64, bool) {
	m := memoryStatusEx{Length: uint32(unsafe.Sizeof(memoryStatusEx{}))}
	if r, _, _ := globalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&m))); r == 0 || m.TotalPhys == 0 {
		return 0, false
	}
	return float64(m.TotalPhys-m.AvailPhys) / float64(m.TotalPhys) * 100, true
}

func leerCPU() (muestraCPU, bool) {
	var ocioso, kernel, usuario windows.Filetime
	r, _, _ := getSystemTimes.Call(uintptr(unsafe.Pointer(&ocioso)), uintptr(unsafe.Pointer(&kernel)), uintptr(unsafe.Pointer(&usuario)))
	if r == 0 {
		return muestraCPU{}, false
	}
	n := func(t windows.Filetime) uint64 { return uint64(t.HighDateTime)<<32 | uint64(t.LowDateTime) }
	return muestraCPU{Total: n(kernel) + n(usuario), Ocioso: n(ocioso)}, true
}

func otrasGPUs() []*float64 { return nil }
