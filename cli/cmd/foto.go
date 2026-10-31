package cmd

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/qdd-framework/qdd/pkg/foto"
	"github.com/qdd-framework/qdd/pkg/integration"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'
// qdd foto y la tool MCP qdd_foto (pedido del usuario, 2026-10-31): una captura pegada en la terminal llega como bytes
// de PNG. qdd trae la imagen del portapapeles o la última captura, en Linux, Windows y macOS, para que la IA la vea.

var fotoDe string

var fotoCmd = &cobra.Command{
	Use:   "foto",
	Short: "Trae la imagen del portapapeles (o la última captura de pantalla) para que la IA la vea",
	Long: `Lee la imagen del portapapeles con la herramienta del sistema (wl-paste o xclip en Linux, PowerShell en Windows,
osascript en macOS) y la deja en tmp/fotos (git no la ve). Sin imagen en el portapapeles, usa la última captura de
pantalla. Imprime la ruta para decirle a la IA «mira <ruta>»; por MCP, qdd_foto le entrega la imagen directamente.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
	PersistentPostRun: func(cmd *cobra.Command, args []string) {},
	Run: func(cmd *cobra.Command, args []string) {
		cwd, _ := os.Getwd()
		f, err := foto.Traer(cmd.Context(), integration.FindProjectRoot(cwd), fotoDe, foto.EntornoReal(runtime.GOOS), time.Now())
		if err != nil {
			salirCon("%v", err)
		}
		fmt.Printf("[+] %s: %s (%s, %s)\n", f.Fuente, rutaCorta(cwd, f.Ruta), f.Tipo, pesoFoto(f.Bytes))
		fmt.Printf("    Para la IA: «mira %s»\n", rutaCorta(cwd, f.Ruta))
	},
}

func init() {
	fotoCmd.Flags().StringVar(&fotoDe, "de", foto.DeAuto, "auto | portapapeles | captura")
	rootCmd.AddCommand(fotoCmd)
}

// rutaCorta es la ruta relativa a la carpeta actual si queda dentro, o la absoluta.
func rutaCorta(cwd, ruta string) string {
	rel, err := filepath.Rel(cwd, ruta)
	if err != nil || filepath.IsAbs(rel) || len(rel) > 1 && rel[:2] == ".." {
		return ruta
	}
	return rel
}

func pesoFoto(n int) string {
	if n < 1<<20 {
		return fmt.Sprintf("%d KB", (n+1023)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
}

func registerFotoTool(s *server.MCPServer) {
	tool := mcp.NewTool("qdd_foto",
		mcp.WithDescription("Trae la imagen copiada o la última captura (Linux, Windows, macOS) para que la veas. Úsala si la persona dice «mira la captura» / «te copié una imagen» o pega bytes de imagen como texto."),
		mcp.WithString("de", mcp.Description("auto (def.: portapapeles; si no, la última captura) | portapapeles | captura")),
	)
	RegisterToolWithTelemetry(s, tool, handleFotoTool)
}

func handleFotoTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	f, err := foto.Traer(ctx, projectRootForMCP(), stringArg(request, "de"), foto.EntornoReal(runtime.GOOS), time.Now())
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	texto := fmt.Sprintf("%s: %s (%s, %s)", f.Fuente, f.Ruta, f.Tipo, pesoFoto(f.Bytes))
	return &mcp.CallToolResult{Content: []mcp.Content{
		mcp.NewTextContent(texto),
		mcp.NewImageContent(base64.StdEncoding.EncodeToString(f.Datos), f.Tipo),
	}}, nil
}
