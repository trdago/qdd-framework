package consola

import (
	"encoding/json"
	"reflect"
	"strings"
)

// Rule: Prohibido usar 'else'
// FND-188: la cola la reescriben qdd de versiones distintas a la vez (un mcp-server de hace días y el panel recién
// actualizado). Lo que uno no conoce no se pierde al reescribirla: se lee aparte y se devuelve igual.

// camposDePedido son las claves JSON que este qdd conoce.
var camposDePedido = func() map[string]bool {
	campos := map[string]bool{}
	t := reflect.TypeOf(Pedido{})
	for i := 0; i < t.NumField(); i++ {
		nombre, _, _ := strings.Cut(t.Field(i).Tag.Get("json"), ",")
		if nombre == "" || nombre == "-" {
			continue
		}
		campos[nombre] = true
	}
	return campos
}()

// pedidoPlano tiene los campos de Pedido sin sus métodos JSON.
type pedidoPlano Pedido

func (p *Pedido) UnmarshalJSON(b []byte) error {
	var plano pedidoPlano
	if err := json.Unmarshal(b, &plano); err != nil {
		return err
	}
	var todos map[string]json.RawMessage
	if err := json.Unmarshal(b, &todos); err != nil {
		return err
	}
	for k := range todos {
		if camposDePedido[k] {
			delete(todos, k)
		}
	}
	plano.otros = nil
	if len(todos) > 0 {
		plano.otros = todos
	}
	*p = Pedido(plano)
	return nil
}

func (p Pedido) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(pedidoPlano(p))
	if err != nil || len(p.otros) == 0 {
		return b, err
	}
	var todos map[string]json.RawMessage
	if err := json.Unmarshal(b, &todos); err != nil {
		return nil, err
	}
	for k, v := range p.otros {
		if _, propio := todos[k]; propio {
			continue
		}
		todos[k] = v
	}
	return json.Marshal(todos)
}
