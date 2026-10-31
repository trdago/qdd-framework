//go:build windows

package consola

import "golang.org/x/sys/windows"

// stillActive es el código de salida de un proceso que no terminó (STILL_ACTIVE).
const stillActive = 259

// procesoVivo dice si el proceso existe y no terminó.
func procesoVivo(pid int) bool {
	if pid <= 0 {
		return false
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	var codigo uint32
	return windows.GetExitCodeProcess(h, &codigo) == nil && codigo == stillActive
}
