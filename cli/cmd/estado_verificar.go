package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/qdd-framework/qdd/pkg/mejoras"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'
// Esperas que se responden solas (pedido de la persona, 2026-11-08): los verificadores reales de cada tipo. Todos son
// de solo lectura y con tope de tiempo; el mod qdd-tareas llama 'qdd estado verificar --si-toca' cada minuto y qdd
// decide si pasaron los 10 min.

const topeVerificacion = 20 * time.Second

var (
	estadoSiToca bool
	// correrVerificacion corre un programa de solo lectura en la carpeta del proyecto; en las pruebas se cambia.
	correrVerificacion = func(dir, programa string, args ...string) (string, error) {
		ctx, cancel := context.WithTimeout(context.Background(), topeVerificacion)
		defer cancel()
		c := exec.CommandContext(ctx, programa, args...)
		c.Dir = dir
		salida, err := c.Output()
		return strings.TrimSpace(string(salida)), err
	}
)

func verificadoresReales() estado.Verificadores {
	return estado.Verificadores{
		"npm":     verificarNpm,
		"pr":      verificarPR,
		"tag":     verificarTag,
		"pedido":  verificarPedido,
		"archivo": verificarArchivo,
	}
}

// verificarNpm: «npm view <paquete>@<versión> version» devuelve esa versión.
func verificarNpm(root, arg string) (bool, string, error) {
	salida, err := correrVerificacion(root, "npm", "view", arg, "version")
	if err != nil && salida == "" {
		return false, "npm no tiene " + arg, nil
	}
	version := arg[strings.LastIndex(arg, "@")+1:]
	return salida == version, "npm tiene " + arg, nil
}

// verificarPR: el PR del repositorio del proyecto está mergeado.
func verificarPR(root, arg string) (bool, string, error) {
	repo, ok := mejoras.RepoDeProyecto(root)
	if !ok {
		return false, "", fmt.Errorf("el proyecto no tiene un remote de GitHub para ver el PR #%s", arg)
	}
	salida, err := correrVerificacion(root, "gh", "pr", "view", arg, "-R", repo, "--json", "state", "-q", ".state")
	if err != nil {
		return false, "", fmt.Errorf("gh no pudo leer el PR #%s: %v", arg, err)
	}
	return salida == "MERGED", "PR #" + arg + " " + strings.ToLower(salida), nil
}

// verificarTag: el tag está en origin.
func verificarTag(root, arg string) (bool, string, error) {
	salida, err := correrVerificacion(root, "git", "ls-remote", "--tags", "origin", "refs/tags/"+arg)
	if err != nil {
		return false, "", fmt.Errorf("git no pudo leer los tags de origin: %v", err)
	}
	return salida != "", "tag " + arg + " en origin", nil
}

// verificarPedido: el pedido del panel quedó hecho.
func verificarPedido(root, arg string) (bool, string, error) {
	for _, p := range consola.PedidosA(root, time.Now()) {
		if p.ID != arg {
			continue
		}
		return p.Estado == consola.EstadoHecho, "pedido #" + arg + " " + p.Estado, nil
	}
	return false, "", fmt.Errorf("no hay un pedido #%s en el panel", arg)
}

// verificarArchivo: la ruta existe dentro del proyecto (nunca fuera).
func verificarArchivo(root, arg string) (bool, string, error) {
	ruta := filepath.Join(root, filepath.Clean("/"+arg))
	if _, err := os.Stat(ruta); err != nil {
		return false, arg + " todavía no existe", nil
	}
	return true, arg + " existe", nil
}

var estadoVerificarCmd = &cobra.Command{
	Use:   "verificar [--si-toca] [--json]",
	Short: "Revisa los ⏳ que se responden solos (npm:, pr:, tag:, pedido:, archivo:) y pasa a ✅ los cumplidos",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		r, err := estado.Verificar(estadoRaiz(), verificadoresReales(), estadoSiToca, time.Now())
		if err != nil {
			salirCon("%v", err)
		}
		if estadoJSON {
			json.NewEncoder(os.Stdout).Encode(r)
			return
		}
		fmt.Printf("[+] %d revisadas, %d cumplidas.\n", len(r.Revisadas), len(r.Cumplidas))
		for _, c := range r.Cumplidas {
			fmt.Printf("[✓] %s\n", c)
		}
		for _, a := range r.Avisos {
			fmt.Printf("[i] %s\n", a)
		}
	},
}

func init() {
	estadoSetCmd.Flags().StringVar(&estadoCambio.Verifica, "verifica", "", "el ⏳ se responde solo: npm:<paquete>@<versión>, pr:<n>, tag:<tag>, pedido:<n> o archivo:<ruta>")
	estadoVerificarCmd.Flags().BoolVar(&estadoSiToca, "si-toca", false, "solo los que no se revisaron en los últimos 10 min")
	estadoVerificarCmd.Flags().BoolVar(&estadoJSON, "json", false, "salida JSON")
	estadoCmd.AddCommand(estadoVerificarCmd)
}
