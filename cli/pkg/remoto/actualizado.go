package remoto

import (
	"errors"
	"os"
	"time"
)

// Rule: Prohibido usar 'else'
// Al actualizar qdd, el servicio corre el qdd viejo en memoria hasta que se reinicia (FND-120). El bucle mira el
// archivo del que arrancó: si cambió, termina con ErrActualizado y el servicio (Restart=on-failure, KeepAlive o
// RestartOnFailure) lo vuelve a arrancar con el nuevo.

// ErrActualizado: el qdd de este servicio se reemplazó en el disco; hay que arrancar el nuevo.
var ErrActualizado = errors.New("qdd se actualizó en este PC: el servicio vuelve a arrancar con la versión nueva")

// huella es lo que cambia cuando se reemplaza un archivo: su tamaño y su fecha.
type huella struct {
	tam int64
	mod time.Time
}

func huellaDe(ruta string) (huella, bool) {
	if ruta == "" {
		return huella{}, false
	}
	info, err := os.Stat(ruta)
	if err != nil {
		return huella{}, false
	}
	return huella{tam: info.Size(), mod: info.ModTime()}, true
}

// seActualizo: el qdd del que arrancó está en el disco y ya no es el mismo. Mientras falta (npm lo está
// reemplazando) espera al siguiente sondeo.
func (b *Bucle) seActualizo() bool {
	if !b.conHuella {
		return false
	}
	ahora, ok := huellaDe(b.Bin)
	return ok && ahora != b.huellaBin
}
