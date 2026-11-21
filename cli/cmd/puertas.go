package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/blueprints"
	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/qdd-framework/qdd/pkg/findings"
	"github.com/qdd-framework/qdd/pkg/modules"
	"github.com/qdd-framework/qdd/pkg/puertas"
	"github.com/spf13/cobra"
)

// codigoRechazo es la salida de 'qdd puertas revisar' cuando una puerta rechaza: lo distingue de una falla propia,
// que deja pasar la acción.
const codigoRechazo = 2

var puertasCmd = &cobra.Command{
	Use:   "puertas",
	Short: "Puertas de QDD: ficha antes de editar, commit con qdd audit al día, fix con finding y publicar con la persona",
	Long: `Las puertas encaminan a la IA por lo que hace, no por lo que dice: un commit sin un qdd audit aprobado para ese
árbol se rechaza; un fix sin su finding (FND-NNN con test_result) también; publicar (gh pr merge, git push de un tag,
npm publish, gh release create) va como pedido, y editar un archivo que cubre una ficha pide leerla antes.
Se activan por proyecto con 'puertas:' en .qdd/policies.yaml ('publicar: ia' deja publicar a la IA).`,
}

var puertasRevisarCmd = &cobra.Command{
	Use:   "revisar [-- <comando> | --editar <archivo> --leida <ruta>...]",
	Short: "Decide si un comando de shell o una edición pasa las puertas (sale con 2 y dice qué falta si no)",
	Long: `Lo llama el panel de Claude Code antes de cada comando de la IA que commitea o publica, y antes de cada edición
(--editar, con --leida por cada ruta cuyo contexto ya pidió la sesión). Si pasa, no imprime nada y sale con 0; si una
puerta lo rechaza, imprime qué falta y el comando para resolverlo, y sale con 2. Si no puede decidir, deja pasar.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dir, _ := cmd.Flags().GetString("dir")
		sesion, _ := cmd.Flags().GetString("sesion")
		editar, _ := cmd.Flags().GetString("editar")
		leidas, _ := cmd.Flags().GetStringArray("leida")
		if sesion == "" {
			sesion = estado.SesionActual()
		}
		codigo, mensaje := 0, ""
		if editar != "" {
			codigo, mensaje = revisarEdicion(dir, sesion, editar, leidas)
		}
		if editar == "" && len(args) == 1 {
			codigo, mensaje = revisarPuertas(dir, sesion, args[0])
		}
		if mensaje != "" {
			fmt.Fprintln(cmd.OutOrStdout(), mensaje)
		}
		os.Exit(codigo)
	},
}

func init() {
	puertasRevisarCmd.Flags().String("dir", ".", "Carpeta donde corre el comando")
	puertasRevisarCmd.Flags().String("sesion", "", "Sesión de Claude Code que lo corre (por defecto CLAUDE_CODE_SESSION_ID)")
	puertasRevisarCmd.Flags().String("editar", "", "Archivo que la IA va a editar (puerta ficha)")
	puertasRevisarCmd.Flags().StringArray("leida", nil, "Ruta cuyo contexto ya pidió la sesión (se repite)")
	puertasCmd.AddCommand(puertasRevisarCmd)
	rootCmd.AddCommand(puertasCmd)
}

// revisarPuertas decide cada acción del comando y anota la decisión; la primera que rechaza corta.
func revisarPuertas(base, sesion, comando string) (int, string) {
	for _, a := range puertas.Acciones(comando) {
		inicio := time.Now()
		h, proyecto, ok := hechosDe(base, a)
		if !ok {
			continue
		}
		if codigo, mensaje, corta := aplicarPuerta(h, proyecto, sesion, inicio); corta {
			return codigo, mensaje
		}
	}
	return 0, ""
}

// aplicarPuerta decide y anota lo que la puerta decidió (lo que no le tocaba no); corta si rechaza.
func aplicarPuerta(h puertas.Hechos, proyecto, sesion string, inicio time.Time) (int, string, bool) {
	d := puertas.Decidir(h)
	if d.Resultado == puertas.Inactiva || d.Resultado == puertas.NoAplica {
		return 0, "", false
	}
	anotarPuerta(proyecto, puertas.Registro{Fecha: inicio, Puerta: d.Puerta, Resultado: d.Resultado,
		MS: time.Since(inicio).Milliseconds(), Sesion: sesion, Largo: len([]rune(d.Mensaje))})
	if d.Resultado != puertas.Rechaza {
		return 0, "", false
	}
	return codigoRechazo, d.Mensaje, true
}

// revisarEdicion es la puerta ficha: el archivo que se edita y las rutas cuyo contexto ya pidió la sesión (relativas a
// la raíz, como las da qdd_project_context, o a la carpeta de la sesión, como en 'qdd modules context').
func revisarEdicion(base, sesion, archivo string, leidas []string) (int, string) {
	inicio := time.Now()
	abs := absDesde(base, archivo)
	raiz, err := puertas.Raiz(carpetaExistente(filepath.Dir(abs)))
	if err != nil {
		return 0, ""
	}
	politica, _ := os.ReadFile(filepath.Join(raiz, ".qdd", "policies.yaml"))
	h := puertas.Hechos{Accion: puertas.Accion{Puerta: puertas.PuertaFicha, Archivo: relativa(raiz, abs)}, Politica: puertas.PoliticaDe(politica)}
	if !h.Politica.Activa {
		return 0, ""
	}
	mods, _ := modules.Load(raiz)
	h.Ficha, h.Leida = fichaLeida(raiz, base, h.Accion.Archivo, mods, leidas)
	codigo, mensaje, _ := aplicarPuerta(h, puertas.Proyecto(raiz), sesion, inicio)
	return codigo, mensaje
}

// fichaLeida es la ficha que cubre el archivo y si alguna ruta leída la mostró.
func fichaLeida(raiz, base, archivo string, mods []modules.Module, leidas []string) (string, bool) {
	cubren := modules.Covering(mods, archivo)
	if len(cubren) == 0 {
		return "", false
	}
	cubiertas := func(ruta string) []string {
		var ids []string
		for _, m := range modules.Covering(mods, ruta) {
			ids = append(ids, m.ID)
		}
		return ids
	}
	for _, m := range cubren {
		for _, l := range leidas {
			if puertas.LeidaCubre(relativa(raiz, absDesde(raiz, l)), archivo, m.ID, cubiertas) ||
				puertas.LeidaCubre(relativa(raiz, absDesde(base, l)), archivo, m.ID, cubiertas) {
				return m.ID, true
			}
		}
	}
	return cubren[0].ID, false
}

func absDesde(base, ruta string) string {
	if filepath.IsAbs(ruta) {
		return filepath.Clean(ruta)
	}
	abs, err := filepath.Abs(filepath.Join(base, ruta))
	if err != nil {
		return ruta
	}
	return abs
}

// relativa es la ruta desde la raíz, con /; una de afuera queda como vino.
func relativa(raiz, abs string) string {
	rel, err := filepath.Rel(raiz, abs)
	if err != nil {
		return abs
	}
	return filepath.ToSlash(rel)
}

// carpetaExistente sube hasta una carpeta que exista: un archivo nuevo puede ir en una carpeta nueva.
func carpetaExistente(dir string) string {
	for {
		if _, err := os.Stat(dir); err == nil || filepath.Dir(dir) == dir {
			return dir
		}
		dir = filepath.Dir(dir)
	}
}

// hechosDe trae lo que la puerta necesita. ok es falso cuando la acción no es para una puerta: un git push que no
// empuja un tag, o una carpeta que no es un repositorio (no se puede decidir y pasa).
func hechosDe(base string, a puertas.Accion) (puertas.Hechos, string, bool) {
	dir := a.Dir
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(base, dir)
	}
	raiz, err := puertas.Raiz(dir)
	if err != nil {
		return puertas.Hechos{}, "", false
	}
	if len(a.Refs) > 0 && !empujaUnTag(dir, a.Refs) {
		return puertas.Hechos{}, "", false
	}
	politica, _ := os.ReadFile(filepath.Join(raiz, ".qdd", "policies.yaml"))
	h := puertas.Hechos{Accion: a, Politica: puertas.PoliticaDe(politica)}
	if a.Puerta == puertas.PuertaAudit && h.Politica.Activa {
		h.Marca, h.ErrHuella = puertas.LeerMarca(dir)
		h.Huella, h.ErrHuella = huellaSiHayMarca(dir, h.Marca, h.ErrHuella)
	}
	if a.Puerta == puertas.PuertaFinding && h.Politica.Activa {
		h.Mensaje, h.Findings = mensajeDe(dir, a), findingsConPrueba(raiz)
	}
	return h, puertas.Proyecto(dir), true
}

// mensajeDe es el mensaje del commit: el de -m o el archivo de -F (relativo a donde corre; - es la entrada y no se lee).
func mensajeDe(dir string, a puertas.Accion) string {
	if a.ArchivoMensaje == "" || a.ArchivoMensaje == "-" {
		return a.Mensaje
	}
	datos, err := os.ReadFile(absDesde(dir, a.ArchivoMensaje))
	if err != nil {
		return a.Mensaje
	}
	return string(datos)
}

// findingsConPrueba: id → si tiene test_result.
func findingsConPrueba(raiz string) map[string]bool {
	fs, _ := findings.Leer(raiz)
	out := map[string]bool{}
	for _, f := range fs {
		out[f.ID] = out[f.ID] || strings.TrimSpace(f.Prueba) != ""
	}
	return out
}

func huellaSiHayMarca(dir, marca string, err error) (string, error) {
	if err != nil || marca == "" {
		return "", err
	}
	return puertas.Huella(dir)
}

func empujaUnTag(dir string, refs []string) bool {
	for _, r := range refs {
		if puertas.EsTag(dir, r) {
			return true
		}
	}
	return false
}

// anotarPuerta: si no se puede anotar, la decisión igual vale.
func anotarPuerta(proyecto string, r puertas.Registro) {
	home, err := blueprints.Home()
	if err != nil {
		return
	}
	puertas.Anotar(home, proyecto, r)
}
