package baul

import (
	"fmt"
	"slices"
	"time"

	"github.com/zalando/go-keyring"
)

// Rule: Prohibido usar 'else'
// Renombrar (pedido de la persona, 2026-11-20: «quitar, agregar, modificar los nombres»): el valor cambia de nombre en
// su lugar sin salir de qdd, la entrada conserva todo lo demás y los permisos y solicitudes siguen al token.

// Renombrar le cambia el nombre a un token. Si el índice no se puede escribir, el valor vuelve a su nombre anterior.
func Renombrar(viejo, nuevo string, ahora time.Time) error {
	if !nombreValido.MatchString(nuevo) {
		return fmt.Errorf("nombre %q: usa minúsculas, números, - y _ (hasta 40)", nuevo)
	}
	if viejo == nuevo {
		return fmt.Errorf("%s ya se llama así", viejo)
	}
	e, err := Buscar(viejo)
	if err != nil {
		return err
	}
	if _, err := Buscar(nuevo); err == nil {
		return fmt.Errorf("ya hay un token llamado %s: quítalo o elige otro nombre", nuevo)
	}
	if err := moverValor(e, nuevo); err != nil {
		return err
	}
	if err := renombrarEnIndice(viejo, nuevo); err != nil {
		_ = moverValor(Entrada{Nombre: nuevo, Tipo: e.Tipo, Donde: e.Donde}, viejo)
		return err
	}
	if e.Donde == EnLlavero && e.Tipo != "aws-sso" {
		_ = keyring.Delete(servicioLlavero, viejo)
	}
	if err := conPermisos(ahora, func(ps *permisos) error {
		for i := range ps.Permisos {
			if ps.Permisos[i].Token == viejo {
				ps.Permisos[i].Token = nuevo
			}
		}
		for i := range ps.Solicitudes {
			if ps.Solicitudes[i].Token == viejo {
				ps.Solicitudes[i].Token = nuevo
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return Anotar(Linea{Accion: "renombrar", Token: nuevo, Antes: viejo}, ahora)
}

// moverValor deja el valor con el nombre nuevo. En el llavero lo copia (el anterior se borra cuando el índice ya
// cambió); en 1Password le cambia el título al ítem; aws-sso no tiene valor.
func moverValor(e Entrada, nuevo string) error {
	if e.Tipo == "aws-sso" {
		return nil
	}
	if e.Donde == EnUnoPass {
		if err := correrOp("item", "edit", tituloOp(e.Nombre), "--title", tituloOp(nuevo)); err != nil {
			return fmt.Errorf("1Password no le cambió el nombre: %w", err)
		}
		return nil
	}
	valor, err := keyring.Get(servicioLlavero, e.Nombre)
	if err != nil {
		return fmt.Errorf("el llavero no entregó %s: %w", e.Nombre, err)
	}
	return keyring.Set(servicioLlavero, nuevo, valor)
}

// renombrarEnIndice cambia el nombre de la entrada con el baúl bloqueado, y comprueba ahí que el nuevo siga libre.
func renombrarEnIndice(viejo, nuevo string) error {
	return conBloqueo(func() error {
		indice, err := Indice()
		if err != nil {
			return err
		}
		if slices.ContainsFunc(indice, func(x Entrada) bool { return x.Nombre == nuevo }) {
			return fmt.Errorf("ya hay un token llamado %s", nuevo)
		}
		i := slices.IndexFunc(indice, func(x Entrada) bool { return x.Nombre == viejo })
		if i < 0 {
			return ErrNoExiste
		}
		indice[i].Nombre = nuevo
		return escribirJSON("indice.json", indice)
	})
}
