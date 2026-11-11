package cmd

import (
	"fmt"
	"strings"

	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'
// RFC 0013: doctor revisa el registro de proyectos de la máquina (lo que publica qdd remoto), solo si existe. REM-02:
// un registro ilegible es falla y --fix lo aparta (no lo borra; se vuelve a anotar solo). REM-01: una ruta que ya no
// existe se publicaría con error para siempre; --fix la quita. REM-03: un servicio instalado antes de FND-128 mata lo que
// lanzó al reiniciarse; --fix solo le agrega la línea que falta. REM-04: en Windows, la tarea tiene que lanzar la versión
// instalada, que vive en su propia carpeta (FND-142); --fix solo le cambia el binario.

var remotoServicio = servicioRemoto

const grupoRemoto = "9. Consola remota (qdd remoto)"

const (
	itemRemotoLegible  = "Registro de proyectos legible (REM-02)"
	itemRemotoRutas    = "Proyectos del registro que existen (REM-01)"
	itemRemotoServicio = "El servicio no mata lo que lanzó al reiniciarse (REM-03)"
	itemRemotoVersion  = "La tarea lanza la versión instalada (REM-04)"
	itemRemotoCuenta   = "La tarea de inicio es solo de tu usuario (REM-12)"
)

func buildRemotoChecks() *CheckGroup {
	g := &CheckGroup{Name: grupoRemoto}
	if !registro.Existe() {
		return g
	}
	reg, err := registro.Leer()
	if err != nil {
		g.Items = append(g.Items, &CheckItem{Name: itemRemotoLegible, Error: err.Error()})
		return g
	}
	g.Items = append(g.Items, &CheckItem{Name: itemRemotoLegible, Success: true, Error: fmt.Sprintf("%d proyecto(s)", len(reg.Proyectos))})
	muertas := registro.Muertas(reg)
	if len(muertas) == 0 {
		g.Items = append(g.Items, &CheckItem{Name: itemRemotoRutas, Success: true})
		g.Items = append(g.Items, itemsServicioRemoto()...)
		return g
	}
	g.Items = append(g.Items, &CheckItem{Name: itemRemotoRutas, Error: "ya no existen (qdd doctor --fix las quita): " + strings.Join(muertas, ", ")})
	g.Items = append(g.Items, itemsServicioRemoto()...)
	return g
}

func itemsServicioRemoto() []*CheckItem {
	s, err := remotoServicio()
	if err != nil {
		return nil
	}
	instalado, deja, err := s.DejaVivoLoLanzado()
	if err != nil || !instalado {
		return nil
	}
	items := []*CheckItem{itemDejaVivo(deja)}
	if s.TareaSinEstaVersion() {
		items = append(items, &CheckItem{Name: itemRemotoVersion, Error: "al reiniciarse lanzaría otra versión de qdd (qdd doctor --fix la actualiza)"})
	}
	if s.TareaSinCuenta() {
		items = append(items, &CheckItem{Name: itemRemotoCuenta, Error: "se dispara para cualquier usuario, y Windows pide administrador para registrarla: quizás nunca quedó registrada (qdd doctor --fix la limita a tu usuario y la registra)"})
	}
	return items
}

func itemDejaVivo(deja bool) *CheckItem {
	if deja {
		return &CheckItem{Name: itemRemotoServicio, Success: true}
	}
	return &CheckItem{Name: itemRemotoServicio, Error: "un pedido del teléfono muere si qdd se actualiza mientras corre (qdd doctor --fix lo renueva)"}
}

// repairRemoto aparta un registro ilegible o quita las rutas que ya no existen.
func repairRemoto(g *CheckGroup) {
	for _, item := range g.Items {
		if item.Success {
			continue
		}
		if item.Name == itemRemotoServicio {
			fmt.Println("  [~] Renovando el servicio de qdd remoto...")
			s, err := remotoServicio()
			if err == nil {
				err = s.RenovarDejaVivo(remotoEjecutor)
			}
			imprimirReparado([]string{"el servicio ya no mata lo que lanzó"}, err)
			continue
		}
		if item.Name == itemRemotoVersion || item.Name == itemRemotoCuenta {
			fmt.Println("  [~] Poniendo al día la tarea de qdd remoto...")
			s, err := remotoServicio()
			if err == nil {
				err = s.RenovarTarea(remotoEjecutor)
			}
			imprimirReparado([]string{"la tarea lanza la versión instalada y es solo de tu usuario"}, err)
			continue
		}
		if item.Name == itemRemotoLegible {
			fmt.Println("  [~] Apartando el registro de proyectos ilegible...")
			apartado, err := registro.Apartar()
			imprimirReparado([]string{"quedó en " + apartado + "; los proyectos se vuelven a anotar con su próximo mensaje"}, err)
			registrarAutorreparacion("doctor: registro de proyectos", "el registro de proyectos que escribió QDD estaba ilegible", []string{apartado})
			continue
		}
		if item.Name == itemRemotoRutas {
			fmt.Println("  [~] Quitando del registro los proyectos que ya no existen...")
			quitadas, err := registro.QuitarMuertas()
			imprimirReparado(quitadas, err)
		}
	}
}
