package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerHumanCertifierTool(s *server.MCPServer) {
	tool := mcp.NewTool("qdd_certify_human",
		mcp.WithDescription("Ejecuta la certificación de comportamiento humano, accesibilidad (WCAG 2.2) y usabilidad (DAG Engine) sobre la aplicación web."),
		mcp.WithString("target", mcp.Description("Objetivo a certificar: 'all', una categoría ('usability', 'accessibility', 'resilience', 'auth') o un nodo específico ('auth:login_nominal'). Por defecto: 'all'.")),
	)

	RegisterToolWithTelemetry(s, tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		target := extractHumanTargetArg(request)

		cwd, _ := os.Getwd()
		runnerPath, err := findHumanCertifierRunner(cwd)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Motor de certificación humana no disponible: %v", err)), nil
		}

		output, err := runHumanCertifierBuffered(runnerPath, target)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Fallo en la certificación humana:\n%s\nError: %v", output, err)), nil
		}

		return mcp.NewToolResultText(output), nil
	})
}

func extractHumanTargetArg(request mcp.CallToolRequest) string {
	argsMap, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return "all"
	}
	target, ok := argsMap["target"].(string)
	if !ok || target == "" {
		return "all"
	}
	return target
}

func runHumanCertifierBuffered(runnerPath, target string) (string, error) {
	nodePath, err := exec.LookPath("node")
	if err != nil {
		return "", fmt.Errorf("Node.js ('node') no está instalado o no se encuentra en el PATH")
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd := exec.Command(nodePath, runnerPath, target)
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	runErr := cmd.Run()
	combined := stdoutBuf.String()
	if stderrBuf.Len() > 0 {
		combined += "\n" + stderrBuf.String()
	}

	return combined, runErr
}
