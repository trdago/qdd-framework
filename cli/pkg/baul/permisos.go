package baul

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/rendimiento"
)

// Rule: Prohibido usar 'else'
// El permiso es por sesión, por proyecto, por token y por fin, y vale hasta que se cierra la consola (decisión de la
// persona, 2026-11-07). Lo da la persona: con el «Sí» de un pedido del panel que el modo goal no lanza, o en su terminal.

// ErrSinPermiso: esta sesión no tiene permiso para ese token y ese fin; quedó un pedido en el panel.
var ErrSinPermiso = errors.New("sin permiso")

// CodigoTodaviaNo es el «todavía no» de los pedidos: el permiso espera el «Sí» de la persona.
const CodigoTodaviaNo = 75

// Permiso es lo que la persona permitió.
type Permiso struct {
	Sesion   string    `json:"sesion,omitempty"`
	Terminal int       `json:"terminal,omitempty"` // PID de la terminal de la persona, sin sesión de Claude Code
	Proyecto string    `json:"proyecto"`
	Token    string    `json:"token"`
	Fin      string    `json:"fin"`
	Dado     time.Time `json:"dado"`
}

// Solicitud espera el «Sí» de la persona; su id va en el comando del pedido.
type Solicitud struct {
	ID       string    `json:"id"`
	Sesion   string    `json:"sesion"`
	Proyecto string    `json:"proyecto"`
	Token    string    `json:"token"`
	Fin      string    `json:"fin"`
	Creada   time.Time `json:"creada"`
}

type permisos struct {
	Permisos    []Permiso   `json:"permisos"`
	Solicitudes []Solicitud `json:"solicitudes"`
}

// vigente: el de una sesión vive lo que ella late en el proyecto; el de una terminal, lo que el proceso.
func vigente(p Permiso, ahora time.Time) bool {
	if p.Sesion != "" {
		return consola.SesionLate(p.Proyecto, p.Sesion, ahora)
	}
	return procesoVivo(p.Terminal)
}

func procesoVivo(pid int) bool {
	if pid <= 0 {
		return false
	}
	lista, err := rendimiento.Procesos()
	if err != nil {
		return false
	}
	return slices.ContainsFunc(lista, func(p rendimiento.Proceso) bool { return p.PID == pid && !p.Zombie })
}

// conPermisos lee, poda los que murieron, deja cambiar y guarda, con el baúl bloqueado.
func conPermisos(ahora time.Time, f func(*permisos) error) error {
	return conBloqueo(func() error {
		var ps permisos
		if err := leerJSON("permisos.json", &ps); err != nil {
			return err
		}
		ps.Permisos = slices.DeleteFunc(ps.Permisos, func(p Permiso) bool { return !vigente(p, ahora) })
		ps.Solicitudes = slices.DeleteFunc(ps.Solicitudes, func(s Solicitud) bool {
			return !vigente(Permiso{Sesion: s.Sesion, Proyecto: s.Proyecto}, ahora)
		})
		if err := f(&ps); err != nil {
			return err
		}
		return escribirJSON("permisos.json", ps)
	})
}

func mismo(p Permiso, sesion string, terminal int, proyecto, token, fin string) bool {
	return p.Sesion == sesion && p.Terminal == terminal && p.Proyecto == proyecto && p.Token == token && p.Fin == fin
}

func tienePermiso(sesion string, terminal int, proyecto, token, fin string, ahora time.Time) (bool, error) {
	var hay bool
	err := conPermisos(ahora, func(ps *permisos) error {
		hay = slices.ContainsFunc(ps.Permisos, func(p Permiso) bool { return mismo(p, sesion, terminal, proyecto, token, fin) })
		return nil
	})
	return hay, err
}

// ComandoPermitir es el comando del pedido que la persona confirma.
func ComandoPermitir(id string) string {
	return "qdd baul permitir " + id
}

// pedirPermiso deja (o reusa) la solicitud y su pedido de la persona en el panel del proyecto.
func pedirPermiso(proyecto, sesion, token, fin string, ahora time.Time) error {
	var sol Solicitud
	err := conPermisos(ahora, func(ps *permisos) error {
		i := slices.IndexFunc(ps.Solicitudes, func(s Solicitud) bool {
			return s.Sesion == sesion && s.Proyecto == proyecto && s.Token == token && s.Fin == fin
		})
		if i >= 0 {
			sol = ps.Solicitudes[i]
			return nil
		}
		sol = Solicitud{ID: idNuevo(), Sesion: sesion, Proyecto: proyecto, Token: token, Fin: fin, Creada: ahora.UTC()}
		ps.Solicitudes = append(ps.Solicitudes, sol)
		return nil
	})
	if err != nil {
		return err
	}
	motivo := cortar(fmt.Sprintf("🔑 %s: %s", token, fin), consola.MaxMotivo)
	if _, err := consola.PedirEnOrden(proyecto, ComandoPermitir(sol.ID), motivo, proyecto, consola.Orden{DePersona: true}, ahora); err != nil {
		return err
	}
	return Anotar(Linea{Quien: sesion, Proyecto: proyecto, Accion: "pedir", Token: token, Fin: fin}, ahora)
}

// Permitir convierte la solicitud en permiso, solo si corre dentro del pedido que la persona confirmó.
func Permitir(proyecto, id, pedido string, confirmado func(pedido, comando string) bool, ahora time.Time) error {
	if pedido == "" || !confirmado(pedido, ComandoPermitir(id)) {
		return fmt.Errorf("el permiso lo da la persona con «s  Sí» en el panel: este comando corre solo desde ese pedido")
	}
	var dado Permiso
	err := conPermisos(ahora, func(ps *permisos) error {
		i := slices.IndexFunc(ps.Solicitudes, func(s Solicitud) bool { return s.ID == id && s.Proyecto == proyecto })
		if i < 0 {
			return fmt.Errorf("no hay una solicitud %q en este proyecto (la sesión se cerró o ya se permitió)", id)
		}
		s := ps.Solicitudes[i]
		dado = Permiso{Sesion: s.Sesion, Proyecto: s.Proyecto, Token: s.Token, Fin: s.Fin, Dado: ahora.UTC()}
		ps.Solicitudes = slices.Delete(ps.Solicitudes, i, i+1)
		ps.Permisos = append(ps.Permisos, dado)
		return nil
	})
	if err != nil {
		return err
	}
	return Anotar(Linea{Quien: "persona", Proyecto: proyecto, Accion: "permitir", Token: dado.Token, Fin: dado.Fin}, ahora)
}

// permitirTerminal anota el «sí» que la persona dio en su terminal.
func permitirTerminal(pid int, proyecto, token, fin string, ahora time.Time) error {
	err := conPermisos(ahora, func(ps *permisos) error {
		ps.Permisos = append(ps.Permisos, Permiso{Terminal: pid, Proyecto: proyecto, Token: token, Fin: fin, Dado: ahora.UTC()})
		return nil
	})
	if err != nil {
		return err
	}
	return Anotar(Linea{Quien: "persona", Proyecto: proyecto, Accion: "permitir", Token: token, Fin: fin}, ahora)
}

func idNuevo() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func cortar(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
