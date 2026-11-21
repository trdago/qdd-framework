package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/flujo"
	"github.com/qdd-framework/qdd/pkg/invitaciones"
)

// Rule: Prohibido usar 'else'
// «Flujos compartidos» en qdd doctor (RFC 0025, fase 6): solo si el proyecto compartió algo. Revisar no usa la red;
// --fix sí (revocar y subir fotos) y nunca toca una propuesta. FLC-04 es FLU-02, en el grupo de flujos.

const (
	grupoFlujosCompartidos     = "15. Flujos compartidos (qdd flujo compartir)"
	itemFlujosHuerfanas        = "Ninguna invitación viva de un flujo que ya no existe (FLC-01)"
	itemFlujosPropuestasViejas = "Ninguna propuesta sin resolver de más de 7 días (FLC-02)"
	itemFlujosFotos            = "El relay tiene la foto al día de cada flujo compartido (FLC-03)"
)

// relayDoctorFlujos es lo que --fix necesita del relay; las pruebas lo cambian por uno falso.
type relayDoctorFlujos interface {
	invitaciones.Relay
	invitaciones.Revocador
}

var relayDeDoctorFlujos = func(ctx context.Context) (relayDoctorFlujos, error) {
	return clienteDeInvitaciones(ctx)
}

func existeFlujo(root string) func(string) bool {
	return func(id string) bool {
		_, err := flujo.Cargar(root, id)
		return err == nil
	}
}

func buildFlujosCompartidosChecks(root string, ahora time.Time) *CheckGroup {
	g := &CheckGroup{Name: grupoFlujosCompartidos}
	r := invitaciones.Revisar(root, existeFlujo(root), ahora)
	if !r.Compartido {
		return g
	}
	huerfanas := &CheckItem{Name: itemFlujosHuerfanas, Success: len(r.Huerfanas) == 0}
	if !huerfanas.Success {
		huerfanas.Error = nombresDeInvitaciones(r.Huerfanas) + ": su flujo ya no existe y el invitado sigue viendo la última foto (--fix la revoca)"
	}
	viejas := &CheckItem{Name: itemFlujosPropuestasViejas, Success: len(r.Viejas) == 0}
	if !viejas.Success {
		viejas.Error = nombresDePropuestas(r.Viejas) + ": apruébalas o recházalas en «Para ti»; --fix no toca propuestas"
	}
	fotos := &CheckItem{Name: itemFlujosFotos, Success: len(r.FotosPorSubir) == 0}
	if !fotos.Success {
		fotos.Error = "el relay tiene una foto vieja de " + strings.Join(r.FotosPorSubir, ", ") + " (--fix la sube)"
	}
	g.Items = append(g.Items, huerfanas, viejas, fotos)
	return g
}

func nombresDeInvitaciones(lista []invitaciones.Invitacion) string {
	var out []string
	for _, i := range lista {
		quien := i.Para
		if quien == "" {
			quien = i.ID
		}
		out = append(out, fmt.Sprintf("la de %s en «%s»", quien, i.Flujo))
	}
	return strings.Join(out, ", ")
}

func nombresDePropuestas(lista []invitaciones.Vista) string {
	var out []string
	for _, v := range lista {
		out = append(out, v.ID+" en «"+v.Flujo+"»")
	}
	return strings.Join(out, ", ")
}

// repairFlujosCompartidos revoca las invitaciones huérfanas y sube las fotos pendientes; sin relay, lo dice y sigue.
func repairFlujosCompartidos(root string, g *CheckGroup, ahora time.Time) {
	if !falla(g, itemFlujosHuerfanas) && !falla(g, itemFlujosFotos) {
		return
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	relay, err := relayDeDoctorFlujos(ctx)
	if err != nil {
		fmt.Printf("[!] flujos compartidos: %v\n", err)
		return
	}
	if falla(g, itemFlujosHuerfanas) {
		hechas, err := invitaciones.RevocarHuerfanas(ctx, root, relay, existeFlujo(root), ahora)
		reportarReparacion("invitaciones revocadas", hechas, err)
	}
	if !falla(g, itemFlujosFotos) {
		return
	}
	e, err := entornoDePropuestas(ctx, root, ahora)
	if err != nil {
		fmt.Printf("[!] fotos de flujos: %v\n", err)
		return
	}
	e.Relay = relay
	if err := invitaciones.SubirFotosPendientes(ctx, e); err != nil {
		fmt.Printf("[!] fotos de flujos: %v\n", err)
	}
}

func falla(g *CheckGroup, nombre string) bool {
	for _, it := range g.Items {
		if it.Name == nombre {
			return !it.Success
		}
	}
	return false
}

func reportarReparacion(que string, hechas []string, err error) {
	if len(hechas) > 0 {
		fmt.Printf("[+] %s: %s\n", que, strings.Join(hechas, ", "))
	}
	if err != nil {
		fmt.Printf("[!] %s: %v\n", que, err)
	}
}
