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
// REM-15: el teléfono quitó este PC; arrancar el servicio no sirve (saldría igual), así que doctor pide emparejar de
// nuevo y no revisa lo demás. REM-16: credenciales.json o relay.json rotos; --fix aparta el roto y pide el relay al
// directorio. REM-17: el servicio lanza un qdd que ya no existe (otra versión de node); --fix lo reescribe con el de
// ahora. REM-18: el canal en tiempo real caído mientras el latido sigue; --fix reinicia el servicio, igual que REM-07.
// REM-19: el reloj de este PC frente al del relay, que sella las acciones del teléfono; lo corrige la persona.
// REM-20: la última prueba del canal con el teléfono que corrió el servicio (sonda); --fix no sondea (doctor no usa la red salvo el directorio):
// aplica la reparación de su tramo, comparar el relay con el directorio o reiniciar el servicio, si nada lo reinició ya
// en esta pasada, y el servicio repite la sonda al arrancar. REM-21: el
// último estado completo frente al tope de Realtime, que descarta sin aviso lo que lo pasa (FND-209); --fix no lo toca.

const grupoMovil = "12. QDD Móvil (teléfono)"

const (
	grupoFlujos       = "13. Flujos (qdd flujo)"
	itemFlujosValidos = "Flujos del proyecto válidos (FLU-01)"
	// itemFlujosHistorial: la revisión de cada flujo está en su historial (RFC 0025); un YAML editado a mano no.
	itemFlujosHistorial = "Historial de cada flujo al día (FLU-02)"
)

const (
	itemMovilEmparejado = "Este PC está emparejado (REM-05)"
	itemMovilCorre      = "El servicio de qdd remoto corre (REM-06)"
	itemMovilRelay      = "Este PC llega al relay (REM-07)"
	itemMovilProyecto   = "Este proyecto se publica (REM-08)"
	itemMovilEnvio      = "El último envío al teléfono salió (REM-09)"
	itemMovilColas      = "Colas del teléfono sin apartar (REM-10)"
	itemMovilDirectorio = "El relay coincide con su directorio firmado (REM-11)"
	itemMovilQuitado    = "El teléfono no quitó este PC (REM-15)"
	itemMovilArchivos   = "Credenciales y relay de este PC se leen (REM-16)"
	itemMovilBinario    = "El servicio lanza un qdd que existe (REM-17)"
	itemMovilCanal      = "El canal en tiempo real está unido (REM-18)"
	itemMovilReloj      = "El reloj de este PC coincide con el del relay (REM-19)"
	itemMovilSonda      = "El canal con el teléfono va y vuelve (REM-20)"
	itemMovilPeso       = "El estado para el teléfono cabe en el relay (REM-21)"

	movilSinAnotar     = "no está en el registro"
	maxLatidosPerdidos = 3
)

func buildMovilChecks(root string, now time.Time) *CheckGroup {
	g := &CheckGroup{Name: grupoMovil}
	if remoto.CredencialesIlegibles() {
		g.Items = append(g.Items, &CheckItem{Name: itemMovilArchivos, Error: "credenciales.json no se lee y este PC no sabe con qué teléfono está emparejado (qdd doctor --fix lo aparta; después, qdd login)"})
		return conItemDirectorio(g)
	}
	if !estaEmparejado() && !servicioMovilInstalado() {
		return conItemDirectorio(g)
	}
	if !estaEmparejado() {
		g.Items = append(g.Items, &CheckItem{Name: itemMovilEmparejado, Error: "falta emparejarlo: ejecuta qdd login y escanea el código con el teléfono"})
		return conItemDirectorio(g)
	}
	g.Items = append(g.Items, &CheckItem{Name: itemMovilEmparejado, Success: true})
	cred, _ := remoto.LeerCredenciales()
	if en, ok := remoto.QuitadoEn(cred.Topico); ok {
		g.Items = append(g.Items, &CheckItem{Name: itemMovilQuitado, Error: "el teléfono lo quitó el " + en.Local().Format("02/01 15:04") + " y el relay ya no lo reconoce: arrancar el servicio no sirve; empareja de nuevo con qdd remoto desactivar y qdd login"})
		return conItemColas(g, root)
	}
	g.Items = append(g.Items, &CheckItem{Name: itemMovilQuitado, Success: true})
	if motivo := remoto.RelayNoSeLee(); motivo != "" {
		g.Items = append(g.Items, &CheckItem{Name: itemMovilArchivos, Error: motivo + " y el servicio no arranca sin él (qdd doctor --fix lo pide al directorio firmado)"})
	}
	// El directorio va antes que el servicio: --fix repara en este orden y el servicio tiene que arrancar con el relay bueno.
	g = conItemBinario(conItemDirectorio(g))
	if !remoto.Corriendo() {
		g.Items = append(g.Items, &CheckItem{Name: itemMovilCorre, Error: "no corre y el teléfono no recibe nada (qdd doctor --fix lo arranca)"}, itemMovilProyectoDe(root))
		return conItemColas(g, root)
	}
	g.Items = append(g.Items, &CheckItem{Name: itemMovilCorre, Success: true}, itemMovilRelayDe(now))
	g = conItemCanal(g, now)
	g.Items = append(g.Items, itemMovilProyectoDe(root))
	return conItemReloj(conItemColas(conItemPeso(conItemSonda(conItemEnvio(g), now)), root), now)
}

// conItemBinario: solo systemd y launchd; la tarea de Windows ya la revisa REM-04 contra el qdd de esta versión.
func conItemBinario(g *CheckGroup) *CheckGroup {
	s, err := remotoServicio()
	if err != nil || s.SO == "windows" {
		return g
	}
	if s.BinarioInstalado() == "" {
		return g
	}
	bin, perdido := s.BinarioPerdidoDeEsteHome()
	if !perdido {
		g.Items = append(g.Items, &CheckItem{Name: itemMovilBinario, Success: true})
		return g
	}
	g.Items = append(g.Items, &CheckItem{Name: itemMovilBinario, Error: bin + " ya no existe y el servicio no puede arrancar (qdd doctor --fix lo reescribe con este qdd)"})
	return g
}

// conItemCanal: sin conexion.json el servicio es de una versión anterior y no hay nada que leer.
func conItemCanal(g *CheckGroup, now time.Time) *CheckGroup {
	unido, desde, err := remoto.LeerConexion()
	if err != nil {
		return g
	}
	hace := now.Sub(desde)
	if unido || hace <= maxLatidosPerdidos*remoto.CadaLatido {
		g.Items = append(g.Items, &CheckItem{Name: itemMovilCanal, Success: true})
		return g
	}
	g.Items = append(g.Items, &CheckItem{Name: itemMovilCanal, Error: fmt.Sprintf("caído hace %d min: el teléfono no recibe cambios ni puede aprobar; el motivo está en el registro de qdd remoto (qdd doctor --fix reinicia el servicio)", int(hace.Minutes()))})
	return g
}

// maxDesfaseRelay avisa antes de perder acciones: una acción del teléfono vale 5 minutos desde que el relay la sella.
const maxDesfaseRelay = 2 * time.Minute

// conItemReloj (REM-19, consensuado con qdd-movile el 2026-11-15): la hora de una acción la pone el relay, así que lo que
// importa es el reloj de este PC frente al del relay, que da el último latido. Las acciones descartadas por vieja en
// log.jsonl son la evidencia; si el reloj está bien, llegaron tarde por el canal, y eso es REM-18.
func conItemReloj(g *CheckGroup, now time.Time) *CheckGroup {
	desfase, conHora := remoto.DesfaseRelay()
	desfaseLog, n := remoto.DesfaseReloj(now)
	if !conHora && n > 0 {
		desfase, conHora = desfaseLog, true
	}
	descartadas := ""
	if n > 0 {
		descartadas = fmt.Sprintf(" (%d acciones del teléfono se descartaron en 24 h)", n)
	}
	if conHora && (desfase > maxDesfaseRelay || desfase < -maxDesfaseRelay) {
		sentido := "adelantado"
		if desfase < 0 {
			sentido, desfase = "atrasado", -desfase
		}
		g.Items = append(g.Items, &CheckItem{Name: itemMovilReloj, Error: fmt.Sprintf("este PC va %d min %s respecto del relay y una acción del teléfono vale 5 min%s: activa la hora automática en este PC", int(desfase.Minutes()), sentido, descartadas)})
		return g
	}
	if n > 0 {
		g.Items = append(g.Items, &CheckItem{Name: itemMovilReloj, Error: "el reloj está bien, pero" + descartadas + " llegaron tarde por el canal en tiempo real: revisa REM-18 y el registro de qdd remoto"})
		return g
	}
	g.Items = append(g.Items, &CheckItem{Name: itemMovilReloj, Success: true})
	return g
}

// conItemPeso compara el último estado completo que salió con el tope de Realtime (REM-21, FND-209). Reducirlo es una
// decisión (qué proyectos o qué parte del panel van), así que --fix no lo toca.
func conItemPeso(g *CheckGroup) *CheckGroup {
	p, err := remoto.LeerPeso()
	if err != nil {
		return g
	}
	aviso := remoto.AvisoDePeso(p.Bytes)
	if aviso == "" {
		g.Items = append(g.Items, &CheckItem{Name: itemMovilPeso, Success: true})
		return g
	}
	g.Items = append(g.Items, &CheckItem{Name: itemMovilPeso, Error: aviso + ": si lo pasa, el teléfono se queda sin el panel de este PC"})
	return g
}

// remotoSondear es la sonda de verdad; las pruebas la cambian por una falsa.
var remotoSondear = remoto.Sondear

// conItemSonda dice cómo salió la última sonda del servicio, sin usar la red. Sin ninguna todavía no opina: el servicio
// corre la primera un minuto después de arrancar.
func conItemSonda(g *CheckGroup, now time.Time) *CheckGroup {
	s, err := remoto.LeerSonda()
	if err != nil || s.Ultima.En.IsZero() {
		return g
	}
	r := s.Ultima
	hace := now.Sub(r.En)
	if hace > 3*remoto.CadaSonda {
		g.Items = append(g.Items, &CheckItem{Name: itemMovilSonda, Error: fmt.Sprintf("la última prueba es de hace %d h: el servicio no la está corriendo (qdd doctor --fix lo reinicia para que la corra)", int(hace.Hours()))})
		return g
	}
	if !r.OK && remoto.ReparacionDe(r) == "" {
		g.Items = append(g.Items, &CheckItem{Name: itemMovilSonda, Error: r.Motivo + " (no se repara desde este PC; qdd remoto sonda la repite ahora)"})
		return g
	}
	if !r.OK {
		g.Items = append(g.Items, &CheckItem{Name: itemMovilSonda, Error: r.Motivo + conReparado(r) + " (qdd doctor --fix aplica su reparación; qdd remoto sonda la repite ahora)"})
		return g
	}
	g.Items = append(g.Items, &CheckItem{Name: itemMovilSonda, Success: true})
	return g
}

func conReparado(r remoto.ResultadoSonda) string {
	if r.Reparado == remoto.ReparaReiniciar {
		return "; el servicio ya se reinició para reconectar"
	}
	if r.Reparado == remoto.ReparaDirectorio {
		return "; ya se comparó el relay con su directorio"
	}
	return ""
}

// repararSonda aplica la reparación de la última sonda del servicio sin sondear: doctor no usa la red salvo el
// directorio firmado (invariante de la ficha remoto), y el servicio repite la sonda al minuto de arrancar. Dice lo que
// hizo y si reinició el servicio, para que nada más lo reinicie en la misma pasada.
func repararSonda(now time.Time) ([]string, bool, error) {
	s, err := remoto.LeerSonda()
	if err != nil {
		return nil, false, err
	}
	r := s.Ultima
	if now.Sub(r.En) > 3*remoto.CadaSonda {
		return reiniciarPorSonda(now, "el servicio no corría la prueba del canal: se reinició y la corre al minuto de arrancar")
	}
	reparacion := remoto.ReparacionDe(r)
	if reparacion == remoto.ReparaDirectorio {
		hecho, err := repararRelayMovil()
		return append([]string{r.Motivo}, hecho...), false, err
	}
	if reparacion == remoto.ReparaReiniciar {
		return reiniciarPorSonda(now, r.Motivo+": el servicio se reinició para reconectar y repite la prueba al arrancar")
	}
	return nil, false, errors.New(r.Motivo + ": no tiene una reparación segura; el detalle está en el registro de qdd remoto")
}

// reiniciarPorSonda reinicia el servicio y lo anota para el tope común de reinicios por el canal.
func reiniciarPorSonda(now time.Time, hecho string) ([]string, bool, error) {
	if !remoto.Corriendo() {
		return nil, false, errors.New("el servicio no corre: REM-06 lo arranca")
	}
	err := reiniciarServicioMovil()
	anotarReinicioCanal(err == nil, now)
	return []string{hecho}, err == nil, err
}

// anotarReinicioCanal: un reinicio de doctor por el canal (REM-07, REM-18) cuenta para el tope del servicio, así la
// sonda no lo vuelve a reiniciar por la misma caída.
func anotarReinicioCanal(reiniciado bool, now time.Time) {
	if !reiniciado {
		return
	}
	if err := remoto.AnotarReinicioCanal(now); err != nil {
		fmt.Printf("      [i] no se anotó el reinicio: %v\n", err)
	}
}

func textoSondaBuena(r remoto.ResultadoSonda) string {
	if r.Eco {
		return fmt.Sprintf("el canal va y vuelve en %d ms con %d teléfono(s) mirando", r.MS, r.Telefonos)
	}
	return fmt.Sprintf("el canal acepta a este PC (%d ms); sin un teléfono mirando no se publica la prueba de ida y vuelta", r.MS)
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
	if r := remoto.SolicitudesApartadas(); r != "" {
		apartadas = append(apartadas, r)
	}
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
	g.Items = append(g.Items, itemDeFlujosValidos(rotos), itemDeHistoriales(root, lista))
	return g
}

func itemDeFlujosValidos(rotos []string) *CheckItem {
	if len(rotos) == 0 {
		return &CheckItem{Name: itemFlujosValidos, Success: true}
	}
	return &CheckItem{Name: itemFlujosValidos, Error: strings.Join(rotos, "; ")}
}

func itemDeHistoriales(root string, lista []flujo.Resumen) *CheckItem {
	var desfases []string
	for _, r := range lista {
		if p := flujo.RevisarHistorial(root, r.ID); p != "" {
			desfases = append(desfases, p)
		}
	}
	if len(desfases) == 0 {
		return &CheckItem{Name: itemFlujosHistorial, Success: true}
	}
	return &CheckItem{Name: itemFlujosHistorial, Error: strings.Join(desfases, "; ") + " (--fix la anota como edición sin registro)"}
}

// repairFlujos anota en el historial la revisión que le falta; un flujo roto no se toca (FLU-01 no se adivina).
func repairFlujos(root string, g *CheckGroup) {
	for _, it := range g.Items {
		if it.Name != itemFlujosHistorial || it.Success {
			continue
		}
		for _, r := range flujo.Listar(root, func(string) string { return "" }) {
			if err := flujo.RepararHistorial(root, r.ID, time.Now()); err != nil {
				fmt.Printf("[!] historial de %s: %v\n", r.ID, err)
			}
		}
	}
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
	// Un servicio reescrito o reiniciado ya arrancó: REM-06, REM-07 y REM-18 no lo vuelven a reiniciar.
	reiniciado := false
	for _, item := range g.Items {
		if item.Success {
			continue
		}
		if item.Name == itemMovilArchivos {
			fmt.Println("  [~] Apartando el archivo roto de QDD Móvil...")
			imprimirReparado(repararArchivosMovil())
			continue
		}
		if item.Name == itemMovilDirectorio {
			fmt.Println("  [~] Comparando el relay con su directorio firmado...")
			imprimirReparado(repararRelayMovil())
			continue
		}
		if item.Name == itemMovilBinario {
			fmt.Println("  [~] Reescribiendo el servicio con este qdd...")
			err := reescribirServicioMovil()
			reiniciado = err == nil
			imprimirReparado([]string{"el servicio lanza este qdd y quedó arrancado"}, err)
			continue
		}
		if item.Name == itemMovilSonda && reiniciado {
			fmt.Println("  [~] Reparando el canal con el teléfono (última prueba del servicio)...")
			imprimirReparado([]string{"el servicio se acaba de reiniciar en esta pasada: repite la prueba al minuto de arrancar"}, nil)
			continue
		}
		if item.Name == itemMovilSonda {
			fmt.Println("  [~] Reparando el canal con el teléfono (última prueba del servicio)...")
			hecho, reinicio, err := repararSonda(now)
			reiniciado = reinicio
			imprimirReparado(hecho, err)
			continue
		}
		if item.Name == itemMovilCorre && !reiniciado {
			fmt.Println("  [~] Arrancando el servicio de qdd remoto...")
			err := arrancarServicioMovil()
			reiniciado = err == nil
			imprimirReparado([]string{"el servicio quedó arrancado"}, err)
			continue
		}
		if (item.Name == itemMovilRelay || item.Name == itemMovilCanal) && !reiniciado {
			fmt.Println("  [~] Reiniciando el servicio de qdd remoto...")
			err := reiniciarServicioMovil()
			reiniciado = err == nil
			anotarReinicioCanal(reiniciado, now)
			imprimirReparado([]string{"el servicio se reinició: en un minuto vuelve a latir y a unirse al canal"}, err)
			continue
		}
		if item.Name == itemMovilProyecto && strings.HasPrefix(item.Error, movilSinAnotar) {
			fmt.Println("  [~] Anotando este proyecto en el registro...")
			imprimirReparado([]string{"el proyecto se publica desde el próximo cambio"}, registro.Anotar(root, now))
		}
	}
}

// repararArchivosMovil: unas credenciales rotas no se reconstruyen (el secreto solo lo tenían ellas); un relay roto o
// perdido se pide al directorio firmado, como al activar.
func repararArchivosMovil() ([]string, error) {
	if remoto.CredencialesIlegibles() {
		ruta, err := remoto.ApartarCredenciales()
		if err != nil {
			return nil, err
		}
		return []string{"quedaron en " + ruta + "; empareja de nuevo con qdd login"}, nil
	}
	c, err := remoto.RecuperarRelay(func(actual remoto.ConfigRelay) (remoto.ConfigRelay, bool, error) {
		return remoto.ActualizarRelay(context.Background(), remotoConsultarRelay, actual)
	})
	if err != nil {
		return nil, fmt.Errorf("el directorio no respondió: %w; desde esta red no se puede, prueba desde otra", err)
	}
	return []string{fmt.Sprintf("quedó el relay del directorio (serie %d)", c.Serie)}, nil
}

// reescribirServicioMovil escribe el servicio con el qdd que corre doctor y lo enciende: el de antes ya no existe.
func reescribirServicioMovil() error {
	s, err := remotoServicio()
	if err != nil {
		return err
	}
	_, err = s.Instalar(remotoEjecutor)
	return err
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
	// Una tarea de Windows escrita sin usuario no se registra sin administrador (FND-195): primero se pone al día.
	if err := s.RenovarTarea(remotoEjecutor); err != nil {
		return err
	}
	for _, c := range s.Comandos(true) {
		if err := remotoEjecutor(c[0], c[1:]...); err != nil {
			return err
		}
	}
	return nil
}
