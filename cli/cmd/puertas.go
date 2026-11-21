package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/qdd-framework/qdd/pkg/blueprints"
	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/qdd-framework/qdd/pkg/puertas"
	"github.com/spf13/cobra"
)

// codigoRechazo es la salida de 'qdd puertas revisar' cuando una puerta rechaza: lo distingue de una falla propia,
// que deja pasar la acción.
const codigoRechazo = 2

var puertasCmd = &cobra.Command{
	Use:   "puertas",
	Short: "Puertas de QDD: no commitear sin qdd audit al día y publicar solo con la persona",
	Long: `Las puertas encaminan a la IA por lo que hace, no por lo que dice: un commit sin un qdd audit aprobado para ese
árbol se rechaza, y publicar (gh pr merge, git push de un tag, npm publish, gh release create) va como pedido.
Se activan por proyecto con 'puertas:' en .qdd/policies.yaml ('publicar: ia' deja publicar a la IA).`,
}

var puertasRevisarCmd = &cobra.Command{
	Use:   "revisar -- <comando>",
	Short: "Decide si un comando de shell pasa las puertas (sale con 2 y dice qué falta si no)",
	Long: `Lo llama el panel de Claude Code antes de cada comando de la IA. Si pasa, no imprime nada y sale con 0; si una
puerta lo rechaza, imprime qué falta y el comando para resolverlo, y sale con 2. Si no puede decidir, deja pasar.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dir, _ := cmd.Flags().GetString("dir")
		sesion, _ := cmd.Flags().GetString("sesion")
		if sesion == "" {
			sesion = estado.SesionActual()
		}
		codigo, mensaje := revisarPuertas(dir, sesion, args[0])
		if mensaje != "" {
			fmt.Fprintln(cmd.OutOrStdout(), mensaje)
		}
		os.Exit(codigo)
	},
}

func init() {
	puertasRevisarCmd.Flags().String("dir", ".", "Carpeta donde corre el comando")
	puertasRevisarCmd.Flags().String("sesion", "", "Sesión de Claude Code que lo corre (por defecto CLAUDE_CODE_SESSION_ID)")
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
		d := puertas.Decidir(h)
		if d.Resultado == puertas.Inactiva {
			continue
		}
		anotarPuerta(proyecto, puertas.Registro{Fecha: inicio, Puerta: d.Puerta, Resultado: d.Resultado,
			MS: time.Since(inicio).Milliseconds(), Sesion: sesion, Largo: len([]rune(d.Mensaje))})
		if d.Resultado == puertas.Rechaza {
			return codigoRechazo, d.Mensaje
		}
	}
	return 0, ""
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
	return h, puertas.Proyecto(dir), true
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
