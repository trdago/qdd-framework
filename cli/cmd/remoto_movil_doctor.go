package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/qdd-framework/qdd/pkg/flujo"
	"github.com/qdd-framework/qdd/pkg/remoto"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'
// QDD Móvil (RFC 0013): doctor dice por qué un proyecto no llegaría al teléfono, sin usar la red y solo si este PC lo
// activó. REM-05: emparejar pide el teléfono de la persona, así que --fix no lo intenta. REM-06: --fix arranca el servicio
// solo si no corre. REM-07: lo último que dijo el relay. REM-08: --fix anota el proyecto; quedar fuera del plan lo decide
// la persona. REM-09: el último envío; el texto del error queda en el registro del servicio, nunca en el reporte. REM-10:
// una cola ilegible se aparta sola y puede traer un mensaje de la persona: doctor la nombra y no la toca. REM-11: el relay
// guardado frente al directorio firmado; es lo único de este grupo que usa la red, y solo con --fix (FND-171).

const grupoMovil = "12. QDD Móvil (teléfono)"

const (
	grupoFlujos       = "13. Flujos (qdd flujo)"
	itemFlujosValidos = "Flujos del proyecto válidos (FLU-01)"
)

const (
	itemMovilEmparejado = "Este PC está emparejado (REM-05)"
	itemMovilCorre      = "El servicio de qdd remoto corre (REM-06)"
	itemMovilRelay      = "Este PC llega al relay (REM-07)"
	itemMovilProyecto   = "Este proyecto se publica (REM-08)"
	itemMovilEnvio      = "El último envío al teléfono salió (REM-09)"
	itemMovilColas      = "Colas del teléfono sin apartar (REM-10)"
	itemMovilDirectorio = "El relay coincide con su directorio firmado (REM-11)"

	movilSinAnotar     = "no está en el registro"
	maxLatidosPerdidos = 3
)

func buildMovilChecks(root string, now time.Time) *CheckGroup {
	g := &CheckGroup{Name: grupoMovil}
	if !estaEmparejado() && !servicioMovilInstalado() {
		return conItemDirectorio(g)
	}
	if !estaEmparejado() {
		g.Items = append(g.Items, &CheckItem{Name: itemMovilEmparejado, Error: "falta emparejarlo: ejecuta qdd login y escanea el código con el teléfono"})
		return conItemDirectorio(g)
	}
	g.Items = append(g.Items, &CheckItem{Name: itemMovilEmparejado, Success: true})
	// El directorio va antes que el servicio: --fix repara en este orden y el servicio tiene que arrancar con el relay bueno.
	g = conItemDirectorio(g)
	if !remoto.Corriendo() {
		g.Items = append(g.Items, &CheckItem{Name: itemMovilCorre, Error: "no corre y el teléfono no recibe nada (qdd doctor --fix lo arranca)"}, itemMovilProyectoDe(root))
		return conItemColas(g, root)
	}
	g.Items = append(g.Items, &CheckItem{Name: itemMovilCorre, Success: true}, itemMovilRelayDe(now), itemMovilProyectoDe(root))
	return conItemColas(conItemEnvio(g), root)
}

// conItemDirectorio: sin relay guardado no hay nada que comparar, y un PC que nunca activó QDD Móvil no reporta nada.
func conItemDirectorio(g *CheckGroup) *CheckGroup {
	c, err := remoto.LeerRelay()
	if err != nil {
		return g
	}
	g.Items = append(g.Items, itemMovilDirectorioDe(c, estaEmparejado()))
	return g
}

func conItemColas(g *CheckGroup, root string) *CheckGroup {
	apartadas := consola.ColasApartadas(root)
	if len(apartadas) == 0 {
		return g
	}
	g.Items = append(g.Items, &CheckItem{Name: itemMovilColas, Error: "se apartaron por ilegibles y pueden traer un mensaje tuyo; revísalas: " + strings.Join(apartadas, ", ")})
	return g
}

// buildFlujosChecks nombra los flujos que no se pueden usar (FLU-01). Corregirlos es decisión de quien los escribió.
func buildFlujosChecks(root string) *CheckGroup {
	g := &CheckGroup{Name: grupoFlujos}
	lista := flujo.Listar(root, func(string) string { return "" })
	if len(lista) == 0 {
		return g
	}
	rotos := []string{}
	for _, r := range lista {
		if r.Problema != "" {
			rotos = append(rotos, r.ID+": "+r.Problema)
		}
	}
	if len(rotos) == 0 {
		g.Items = append(g.Items, &CheckItem{Name: itemFlujosValidos, Success: true})
		return g
	}
	g.Items = append(g.Items, &CheckItem{Name: itemFlujosValidos, Error: strings.Join(rotos, "; ")})
	return g
}

func servicioMovilInstalado() bool {
	s, err := remotoServicio()
	if err != nil {
		return false
	}
	instalado, _, err := s.DejaVivoLoLanzado()
	return err == nil && instalado
}

func itemMovilRelayDe(now time.Time) *CheckItem {
	ultimo, err := remoto.UltimoLatido()
	if err != nil {
		return &CheckItem{Name: itemMovilRelay, Error: "todavía no recibió respuesta del relay: revisa la red y el registro de qdd remoto"}
	}
	hace := now.Sub(ultimo)
	if hace > maxLatidosPerdidos*remoto.CadaLatido {
		return &CheckItem{Name: itemMovilRelay, Error: fmt.Sprintf("sin respuesta del relay hace %d min: revisa la red y el registro de qdd remoto", int(hace.Minutes()))}
	}
	return &CheckItem{Name: itemMovilRelay, Success: true}
}

// itemMovilDirectorioDe dice lo que pasó la última vez, sin usar la red: una consulta fallida no apaga nada, pero un PC
// que no puede confirmar su relay no se enteraría de una mudanza. Solo un relay propio de un PC emparejado no se compara.
// El reporte no lleva la URL.
func itemMovilDirectorioDe(c remoto.ConfigRelay, emparejado bool) *CheckItem {
	if c.Propio && emparejado {
		return &CheckItem{Name: itemMovilDirectorio, Success: true}
	}
	if c.Propio {
		return &CheckItem{Name: itemMovilDirectorio, Error: "guardado como propio y sin emparejar: nada lo confirma (qdd doctor --fix lo compara con el directorio)"}
	}
	u := remoto.LeerConsultaDirectorio()
	if u.OK {
		return &CheckItem{Name: itemMovilDirectorio, Success: true}
	}
	if u.En.IsZero() {
		return &CheckItem{Name: itemMovilDirectorio, Error: "nunca se comparó con el directorio (qdd doctor --fix lo compara)"}
	}
	return &CheckItem{Name: itemMovilDirectorio, Error: "sigue con el relay guardado: " + u.Motivo + " (qdd doctor --fix lo reintenta)"}
}

func itemMovilProyectoDe(root string) *CheckItem {
	reg, err := registro.Leer()
	if err != nil {
		return &CheckItem{Name: itemMovilProyecto, Error: "el registro de proyectos no se lee; lo revisa el grupo de la consola remota"}
	}
	clave := estado.ClaveLocal(root)
	if !tieneProyecto(reg.Proyectos, clave) {
		return &CheckItem{Name: itemMovilProyecto, Error: movilSinAnotar + " (qdd doctor --fix lo anota)"}
	}
	plan, err := remoto.LeerPlan()
	if err != nil {
		return &CheckItem{Name: itemMovilProyecto, Success: true}
	}
	limite := plan.LimiteProyectos()
	if tieneProyecto(remoto.Publicables(reg, limite), clave) {
		return &CheckItem{Name: itemMovilProyecto, Success: true}
	}
	return &CheckItem{Name: itemMovilProyecto, Error: fmt.Sprintf("queda fuera del plan: publica los %d proyectos usados más recientemente", limite)}
}

func tieneProyecto(lista []registro.Proyecto, clave string) bool {
	for _, p := range lista {
		if p.Ruta == clave {
			return true
		}
	}
	return false
}

func conItemEnvio(g *CheckGroup) *CheckGroup {
	ok, en, err := remoto.LeerPublicacion()
	if err != nil {
		return g
	}
	if ok {
		g.Items = append(g.Items, &CheckItem{Name: itemMovilEnvio, Success: true})
		return g
	}
	g.Items = append(g.Items, &CheckItem{Name: itemMovilEnvio, Error: "falló a las " + en.Local().Format("15:04") + " y no ha vuelto a salir: el motivo está en el registro de qdd remoto"})
	return g
}

func repairMovil(root string, g *CheckGroup, now time.Time) {
	for _, item := range g.Items {
		if item.Success {
			continue
		}
		if item.Name == itemMovilDirectorio {
			fmt.Println("  [~] Comparando el relay con su directorio firmado...")
			imprimirReparado(repararRelayMovil())
			continue
		}
		if item.Name == itemMovilCorre {
			fmt.Println("  [~] Arrancando el servicio de qdd remoto...")
			imprimirReparado([]string{"el servicio quedó arrancado"}, arrancarServicioMovil())
			continue
		}
		if item.Name == itemMovilProyecto && strings.HasPrefix(item.Error, movilSinAnotar) {
			fmt.Println("  [~] Anotando este proyecto en el registro...")
			imprimirReparado([]string{"el proyecto se publica desde el próximo cambio"}, registro.Anotar(root, now))
		}
	}
}

// repararRelayMovil dice lo que hizo. Si no puede consultar, no cambia nada y devuelve el motivo. El servicio solo lee el
// relay al arrancar, así que un cambio con él corriendo lo reinicia.
func repararRelayMovil() ([]string, error) {
	actual, err := remoto.LeerRelay()
	if err != nil {
		return nil, err
	}
	c, cambio, err := remoto.ConfirmarRelay(context.Background(), remotoConsultarRelay, actual, estaEmparejado())
	if err != nil {
		return nil, fmt.Errorf("sigue con el relay guardado: %w; desde esta red no se puede confirmar, prueba desde otra", err)
	}
	if !cambio {
		return []string{"el relay coincide con el directorio"}, nil
	}
	hecho := []string{fmt.Sprintf("quedó el relay del directorio (serie %d)", c.Serie)}
	if !remoto.Corriendo() {
		return hecho, nil
	}
	if err := reiniciarServicioMovil(); err != nil {
		return hecho, err
	}
	return append(hecho, "el servicio se reinició con ese relay"), nil
}

func reiniciarServicioMovil() error {
	s, err := remotoServicio()
	if err != nil {
		return err
	}
	instalado, _, err := s.DejaVivoLoLanzado()
	if err != nil {
		return err
	}
	if !instalado {
		return errors.New("qdd remoto correr sigue con el relay anterior: ciérralo y vuelve a correrlo")
	}
	for _, c := range s.ComandosReiniciar() {
		if err := remotoEjecutor(c[0], c[1:]...); err != nil {
			return err
		}
	}
	return nil
}

func arrancarServicioMovil() error {
	s, err := remotoServicio()
	if err != nil {
		return err
	}
	instalado, _, err := s.DejaVivoLoLanzado()
	if err != nil {
		return err
	}
	if !instalado {
		_, err := s.Instalar(remotoEjecutor)
		return err
	}
	for _, c := range s.Comandos(true) {
		if err := remotoEjecutor(c[0], c[1:]...); err != nil {
			return err
		}
	}
	return nil
}
