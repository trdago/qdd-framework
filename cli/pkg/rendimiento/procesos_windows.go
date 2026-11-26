package rendimiento

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// Procesos en Windows: una foto de la tabla con Toolhelp. Windows no tiene zombies: un hijo que nadie espera termina.
func Procesos() ([]Proceso, error) {
	foto, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(foto)
	var e windows.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	if err := windows.Process32First(foto, &e); err != nil {
		return nil, err
	}
	var lista []Proceso
	for {
		lista = append(lista, Proceso{PID: int(e.ProcessID), PPID: int(e.ParentProcessID), Nombre: windows.UTF16ToString(e.ExeFile[:])})
		if windows.Process32Next(foto, &e) != nil {
			return lista, nil
		}
	}
}
