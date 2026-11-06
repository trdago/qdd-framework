package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/qdd-framework/qdd/pkg/remoto"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'
// QDD Móvil (RFC 0013): doctor dice por qué un proyecto no llegaría al teléfono, sin usar la red y solo si este PC lo
// activó. REM-05: emparejar pide el teléfono de la persona, así que --fix no lo intenta. REM-06: --fix arranca el servicio
// solo si no corre. REM-07: lo último que dijo el relay. REM-08: --fix anota el proyecto; quedar fuera del plan lo decide
// la persona. REM-09: el último envío; el texto del error queda en el registro del servicio, nunca en el reporte.

const grupoMovil = "12. QDD Móvil (teléfono)"

const (
	itemMovilEmparejado = "Este PC está emparejado (REM-05)"
	itemMovilCorre      = "El servicio de qdd remoto corre (REM-06)"
	itemMovilRelay      = "Este PC llega al relay (REM-07)"
	itemMovilProyecto   = "Este proyecto se publica (REM-08)"
	itemMovilEnvio      = "El último envío al teléfono salió (REM-09)"

	movilSinAnotar     = "no está en el registro"
	maxLatidosPerdidos = 3
)

func buildMovilChecks(root string, now time.Time) *CheckGroup {
	g := &CheckGroup{Name: grupoMovil}
	_, errCred := remoto.LeerCredenciales()
	if errCred != nil && !servicioMovilInstalado() {
		return g
	}
	if errCred != nil {
		g.Items = append(g.Items, &CheckItem{Name: itemMovilEmparejado, Error: "falta emparejarlo: ejecuta qdd login y escanea el código con el teléfono"})
		return g
	}
	g.Items = append(g.Items, &CheckItem{Name: itemMovilEmparejado, Success: true})
	if !remoto.Corriendo() {
		g.Items = append(g.Items, &CheckItem{Name: itemMovilCorre, Error: "no corre y el teléfono no recibe nada (qdd doctor --fix lo arranca)"}, itemMovilProyectoDe(root))
		return g
	}
	g.Items = append(g.Items, &CheckItem{Name: itemMovilCorre, Success: true}, itemMovilRelayDe(now), itemMovilProyectoDe(root))
	return conItemEnvio(g)
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
