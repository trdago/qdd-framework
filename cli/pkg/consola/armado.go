package consola

import (
	"fmt"
	"io"
	"os"
	"time"
)

// Rule: Prohibido usar 'else'
// Pedido del usuario (2026-11-01): «al ya tener mi intención no debería tener que apretar yo de nuevo… quedar
// escuchando para que cuando esté listo se ejecute mi acción». Confirmar un pedido es la intención de la persona: si
// espera a otro pedido o su comando dice «todavía no» (CodigoTodaviaNo), queda armado y el mismo proceso de qdd que lo
// corre («qdd consola correr», aparte del panel, FND-089) vuelve a intentarlo solo, sin la IA y sin otro «s  Sí», hasta
// que sale bien o falla de verdad. «x Dejar de esperar» (Desarmar) lo vuelve a pendiente; pasado TopeArmado, o si el
// proceso que esperaba murió, también, diciendo por qué.

var (
	// CadaArmado es cada cuánto se reintenta un comando que dijo «todavía no».
	CadaArmado = 30 * time.Second
	// CadaEspera es cada cuánto se mira si terminó el pedido que se espera.
	CadaEspera = 3 * time.Second
	// TopeArmado es cuánto se espera con la intención de la persona antes de devolverle el pedido.
	TopeArmado = 2 * time.Hour
)

// aceptar toma el «s  Sí» de la persona con la cola bloqueada: ejecutando si puede correr ya, armado si espera a otro
// pedido. Uno hecho, en curso o ya armado no se acepta dos veces (FND-070).
func aceptar(root, id, sesion string, now time.Time) (string, error) {
	var estado string
	err := conCola(root, now, func(cola []Pedido) ([]Pedido, error) {
		i := indiceID(cola, id)
		if i < 0 {
			return nil, fmt.Errorf("no hay un pedido '%s' en la cola", id)
		}
		if !cola[i].Ejecutable() {
			return nil, errNoEjecutable(id, cola[i].Estado)
		}
		estado = EstadoEjecutando
		if len(esperaA(cola[i], cola)) > 0 {
			estado = EstadoArmado
		}
		cola[i].Estado, cola[i].Fin, cola[i].Salida, cola[i].PID, cola[i].Hijo = estado, nil, "", os.Getpid(), 0
		cola[i].Inicio, cola[i].Sesion, cola[i].Avisado, cola[i].Confirmado = &now, sesionValida(sesion), false, &now
		// Armado, lo toma el proceso que lo va a esperar (anotarLanzado, tomarTurno), no quien lo lanzó.
		if estado == EstadoArmado {
			cola[i].ArmadoEn, cola[i].Proximo, cola[i].Inicio, cola[i].PID = &now, &now, nil, 0
		}
		return cola, nil
	})
	return estado, err
}

func indiceID(cola []Pedido, id string) int {
	for i, p := range cola {
		if p.ID == id {
			return i
		}
	}
	return -1
}

// armadoTras deja armado un pedido que acaba de decir «todavía no»: lo reintenta pid pasado CadaArmado.
func armadoTras(p Pedido, pid int, aviso string, fin time.Time) Pedido {
	desde := fin
	if p.ArmadoEn != nil {
		desde = *p.ArmadoEn
	}
	proximo := fin.Add(CadaArmado)
	p.Estado, p.Fin, p.PID, p.Aviso, p.AvisoEn = EstadoArmado, nil, pid, aviso, &fin
	p.ArmadoEn, p.Proximo = &desde, &proximo
	return p
}

// desarmado vuelve un pedido armado a pendiente, con el motivo como aviso.
func desarmado(p Pedido, motivo string) Pedido {
	ahora := time.Now()
	if p.Aviso != "" {
		motivo += " Lo último que dijo: " + lineaFinal(p.Aviso)
	}
	p.Estado, p.PID, p.Hijo, p.Proximo, p.ArmadoEn, p.Aviso, p.AvisoEn = EstadoPendiente, 0, 0, nil, nil, motivo, &ahora
	return p
}

func lineaFinal(s string) string {
	return ultimasLineas(s, 1)
}

// Desarmar es «x Dejar de esperar»: el pedido armado vuelve a pendiente y el proceso que esperaba termina sin correrlo.
func Desarmar(root, idOTecla string) error {
	return conCola(root, time.Now(), func(cola []Pedido) ([]Pedido, error) {
		i := indiceDe(cola, idOTecla)
		if i < 0 {
			i = indiceID(cola, idOTecla)
		}
		if i < 0 || cola[i].Estado != EstadoArmado {
			return nil, fmt.Errorf("no hay un pedido armado '%s' en la cola", idOTecla)
		}
		p := desarmado(cola[i], "Dejaste de esperar.")
		p.Aviso = "Dejaste de esperar."
		cola[i] = p
		return cola, nil
	})
}

// Lo que toca hacer en cada vuelta de quien corre un pedido aceptado.
const (
	turnoCorrer  = "correr"
	turnoEsperar = "esperar"
	turnoFin     = "fin"
)

// tomarTurno decide con la cola bloqueada: correr (lo marca ejecutando a nombre de este proceso), esperar (sigue
// armado: falta lo que espera o aún no toca reintentar) o terminar (lo desarmaron, se descartó o pasó el tope). primero:
// la primera vuelta exige un pedido aceptado que nadie más esté corriendo.
func tomarTurno(root, id string, now time.Time, primero bool) (string, time.Duration, error) {
	turno, espera := turnoFin, time.Duration(0)
	err := conCola(root, now, func(cola []Pedido) ([]Pedido, error) {
		i := indiceID(cola, id)
		if i < 0 && primero {
			return nil, fmt.Errorf("no hay un pedido '%s' en la cola", id)
		}
		if i < 0 {
			return cola, nil
		}
		p := cola[i]
		if primero && !aceptadoPorNadie(p) {
			return nil, fmt.Errorf("el pedido %s no espera que lo corran (está %s)", id, p.Estado)
		}
		if !primero && deOtroDueno(p) {
			return cola, nil
		}
		if p.Estado == EstadoEjecutando {
			cola[i].PID, cola[i].Hijo, turno = os.Getpid(), -1, turnoCorrer
			return cola, nil
		}
		if p.Estado != EstadoArmado {
			return cola, nil
		}
		cola[i].PID = os.Getpid()
		if p.ArmadoEn != nil && now.Sub(*p.ArmadoEn) > TopeArmado {
			cola[i] = desarmado(p, "Dejó de esperar tras "+TopeArmado.String()+": confírmalo de nuevo cuando esté listo.")
			return cola, nil
		}
		if len(esperaA(p, cola)) > 0 {
			turno, espera = turnoEsperar, CadaEspera
			return cola, nil
		}
		if p.Proximo != nil && now.Before(*p.Proximo) {
			turno, espera = turnoEsperar, min(p.Proximo.Sub(now), CadaArmado)
			return cola, nil
		}
		cola[i].Estado, cola[i].Inicio, cola[i].Hijo, turno = EstadoEjecutando, &now, -1, turnoCorrer
		return cola, nil
	})
	return turno, espera, err
}

// deOtroDueno: pasada la primera vuelta, el pedido es de otro proceso vivo si lo está corriendo o esperando él. Quien ya
// no es su dueño termina sin correrlo (FND-125: dos procesos despertaban a la misma hora y lo corrían los dos).
func deOtroDueno(p Pedido) bool {
	if p.PID == os.Getpid() {
		return false
	}
	return p.Estado == EstadoEjecutando || (p.Estado == EstadoArmado && p.PID != 0 && procesoVivo(p.PID))
}

// aceptadoPorNadie: ejecutando sin comando en marcha, o armado sin otro proceso vivo que lo espere.
func aceptadoPorNadie(p Pedido) bool {
	if p.Estado == EstadoEjecutando {
		return p.Hijo == 0
	}
	return p.Estado == EstadoArmado && (p.PID == 0 || p.PID == os.Getpid() || !procesoVivo(p.PID))
}

// correrAceptado es el bucle de «qdd consola correr»: corre el pedido cuando le toca y, mientras diga «todavía no» o
// espere a otro, vuelve a intentarlo solo.
func correrAceptado(root, id string, salida io.Writer) int {
	for primero := true; ; primero = false {
		turno, espera, err := tomarTurno(root, id, time.Now(), primero)
		if err != nil {
			fmt.Fprintf(salida, "[!] %v\n", err)
			return 1
		}
		if turno == turnoFin {
			return 0
		}
		if turno == turnoEsperar {
			time.Sleep(espera)
			continue
		}
		codigo := correrUna(root, id, salida)
		if codigo != CodigoTodaviaNo {
			return codigo
		}
	}
}

// correrUna corre el comando del pedido una vez, contando el intento.
func correrUna(root, id string, salida io.Writer) int {
	conCola(root, time.Now(), func(cola []Pedido) ([]Pedido, error) {
		if i := indiceID(cola, id); i >= 0 {
			cola[i].Intentos++
		}
		return cola, nil
	})
	for _, a := range accionesDePedidos(root) {
		if a.pedido == id {
			return correrComando(root, a, salida, true)
		}
	}
	return 1
}
