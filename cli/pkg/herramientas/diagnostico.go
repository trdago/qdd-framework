package herramientas

import (
	"fmt"
	"strings"
)

// Rule: Prohibido usar 'else'

// Diagnostico es una línea de qdd doctor.
type Diagnostico struct {
	Nombre  string
	OK      bool
	Detalle string
}

// Diagnosticar revisa cada herramienta en esta máquina: que su definición se lea y sea válida, que su ejecutable
// exista, que tenga su entorno y que su prueba de humo salga con 0. No repara nada: que falte el ejecutable puede ser
// que no se compiló o que ya no existe, y eso lo decide la persona (el detalle dice cómo, con el comando exacto).
func Diagnosticar(root string) []Diagnostico {
	hs, errs := Cargar(root)
	var out []Diagnostico
	for _, h := range hs {
		out = append(out, diagnosticar(root, h))
	}
	for _, err := range errs {
		nombre := strings.SplitN(err.Error(), ":", 2)[0]
		out = append(out, Diagnostico{Nombre: "Herramienta " + nombre, Detalle: err.Error() + ". Corrígela o quítala."})
	}
	return out
}

func diagnosticar(root string, h Herramienta) Diagnostico {
	d := Diagnostico{Nombre: fmt.Sprintf("Herramienta %s (%s)", h.Nombre, h.ID)}
	if p := Validar(h); len(p) > 0 {
		d.Detalle = strings.Join(p, "; ")
		return d
	}
	if p := validarComando(root, h.Comando); len(p) > 0 {
		d.Detalle = fmt.Sprintf("%s: compílalo, o quítala con qdd herramienta quitar %s --si", p[0], h.ID)
		return d
	}
	if err := PruebaDeHumo(root, h); err != nil {
		d.Detalle = err.Error()
		return d
	}
	d.OK = true
	return d
}
