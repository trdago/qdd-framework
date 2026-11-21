package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/qdd-framework/qdd/pkg/audit"
	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/qdd-framework/qdd/pkg/puertas"
	"github.com/spf13/cobra"
)

var puertasInstalarCmd = &cobra.Command{
	Use:   "instalar",
	Short: "Instala los hooks de git de las puertas (pre-commit y commit-msg) sin tocar lo que ya tienen",
	Long: `Agrega a pre-commit y commit-msg un bloque marcado que corre 'qdd puertas hook': así las puertas llegan a cualquier
IA, no solo a Claude Code. Va tras el shebang, convive con lo que ya hay y con husky va en .husky/. La persona puede
saltarlas en su terminal con git commit --no-verify. Lo corren qdd init y qdd doctor --fix.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		dir, _ := cmd.Flags().GetString("dir")
		cambios, err := puertas.InstalarHooks(dir)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "[!] %v\n", err)
			os.Exit(1)
		}
		imprimirHooks(cambios)
	},
}

var puertasHookCmd = &cobra.Command{
	Use:    "hook <pre-commit|commit-msg> [archivo]",
	Short:  "Lo corren los hooks de git: pre-commit (puerta audit) y commit-msg (puerta finding)",
	Hidden: true,
	Args:   cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		archivo := ""
		if len(args) == 2 {
			archivo = args[1]
		}
		dir, _ := os.Getwd()
		codigo, mensaje := hookPuertas(dir, args[0], archivo)
		if mensaje != "" {
			fmt.Fprintln(cmd.ErrOrStderr(), mensaje)
		}
		os.Exit(codigo)
	},
}

func init() {
	puertasInstalarCmd.Flags().String("dir", ".", "Carpeta del repositorio")
	puertasCmd.AddCommand(puertasInstalarCmd, puertasHookCmd)
}

func imprimirHooks(cambios []string) {
	if len(cambios) == 0 {
		fmt.Println("[✓] Los hooks de las puertas ya están al día.")
		return
	}
	for _, c := range cambios {
		fmt.Printf("[+] Puertas en %s\n", c)
	}
}

// hookPuertas decide la puerta del hook. Sale con 2 si rechaza: el hook corta el commit solo con ese código. Dentro de Claude Code el panel ya
// anotó la decisión del mismo commit: el hook decide igual pero no la anota otra vez.
func hookPuertas(dir, hook, archivo string) (int, string) {
	a := puertas.Accion{Puerta: puertas.PuertaAudit}
	if hook == "commit-msg" {
		a = puertas.Accion{Puerta: puertas.PuertaFinding, ArchivoMensaje: archivo}
	}
	inicio := time.Now()
	h, proyecto, ok := hechosDe(dir, a)
	if !ok {
		return 0, ""
	}
	codigo, mensaje, _ := aplicarPuerta(h, proyecto, estado.SesionActual(), inicio, os.Getenv("CLAUDECODE") == "")
	return codigo, mensaje
}

// grupoPuertas revisa las puertas del proyecto (ficha puertas): activas en la política y con sus hooks al día.
const grupoPuertas = "Puertas de QDD (qdd puertas)"

const itemPuertasActivas = "Puertas activas en .qdd/policies.yaml"

func buildPuertasChecks(projectPath string) *CheckGroup {
	g := &CheckGroup{Name: grupoPuertas}
	if _, err := puertas.Raiz(projectPath); err != nil {
		return g
	}
	datos, _ := os.ReadFile(filepath.Join(projectPath, ".qdd", "policies.yaml"))
	if !puertas.PoliticaDe(datos).Activa {
		g.Items = append(g.Items, &CheckItem{Name: itemPuertasActivas, Error: "desactivadas: qdd doctor --fix las activa (no commitear sin qdd audit, fix con finding, leer la ficha antes de editar, publicar como pedido)"})
		return g
	}
	g.Items = append(g.Items, &CheckItem{Name: itemPuertasActivas, Success: true})
	for _, e := range puertas.EstadoHooks(projectPath) {
		g.Items = append(g.Items, itemDeHook(e))
	}
	return g
}

func itemDeHook(e puertas.EstadoHook) *CheckItem {
	item := &CheckItem{Name: "Hook " + e.Hook + " de las puertas", Success: e.Estado == puertas.HookAlDia}
	if !item.Success {
		item.Error = fmt.Sprintf("%s en %s: qdd doctor --fix lo deja al día sin tocar lo demás", e.Estado, e.Ruta)
	}
	return item
}

// repairPuertas activa las puertas en un proyecto que ya tiene políticas (lo dice) e instala sus hooks. Sin
// policies.yaml no inventa uno: con solo puertas, las demás políticas quedarían apagadas.
func repairPuertas(projectPath string, g *CheckGroup) {
	if !groupHasFailures(g) {
		return
	}
	politicas := filepath.Join(projectPath, ".qdd", "policies.yaml")
	if _, err := os.Stat(politicas); err != nil {
		fmt.Println("  [!] Sin .qdd/policies.yaml no se activan las puertas: corre qdd init.")
		return
	}
	p := audit.LoadPolicies(projectPath)
	if p.Puertas == nil {
		p.Puertas = &audit.PoliticaPuertas{}
		if err := audit.SavePolicies(projectPath, p); err != nil {
			fmt.Printf("  [!] %v\n", err)
			return
		}
		fmt.Println("  [~] Activé las puertas de QDD en .qdd/policies.yaml: commit solo con qdd audit al día, fix con su finding, leer la ficha antes de editar y publicar como pedido. Se apagan quitando 'puertas:'.")
	}
	if !puertas.EsRaiz(projectPath) {
		fmt.Println("  [!] Los hooks son de todo el repositorio: instálalos desde su raíz con qdd puertas instalar.")
		return
	}
	cambios, err := puertas.InstalarHooks(projectPath)
	if err != nil {
		fmt.Printf("  [!] %v\n", err)
		return
	}
	for _, c := range cambios {
		fmt.Printf("  [~] Puertas en %s\n", c)
	}
}
