package rendimiento

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

// contadoresMemoria es PROCESS_MEMORY_COUNTERS de psapi.
type contadoresMemoria struct {
	cb                         uint32
	pageFaultCount             uint32
	peakWorkingSetSize         uintptr
	workingSetSize             uintptr
	quotaPeakPagedPoolUsage    uintptr
	quotaPagedPoolUsage        uintptr
	quotaPeakNonPagedPoolUsage uintptr
	quotaNonPagedPoolUsage     uintptr
	pagefileUsage              uintptr
	peakPagefileUsage          uintptr
}

var procMemoria = windows.NewLazySystemDLL("psapi.dll").NewProc("GetProcessMemoryInfo")

// memoriaDurante: en Windows el máximo se pregunta mientras el proceso vive (PeakWorkingSetSize).
func memoriaDurante(pid int) *float64 {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.PROCESS_VM_READ, false, uint32(pid))
	if err != nil {
		return nil
	}
	defer windows.CloseHandle(h)
	var c contadoresMemoria
	c.cb = uint32(unsafe.Sizeof(c))
	r, _, _ := procMemoria.Call(uintptr(h), uintptr(unsafe.Pointer(&c)), uintptr(c.cb))
	if r == 0 {
		return nil
	}
	mb := float64(c.peakWorkingSetSize) / (1024 * 1024)
	return &mb
}

func memoriaAlTerminar(*os.ProcessState) *float64 { return nil }
