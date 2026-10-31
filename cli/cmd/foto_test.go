package cmd

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// Rule: Prohibido usar 'else'
// qdd foto y qdd_foto (pedido del usuario, 2026-10-31): la IA ve la imagen que la persona copió o capturó, en vez de
// recibir los bytes del PNG pegados como texto.

var png1x1 = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
	0x89, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
	0x42, 0x60, 0x82,
}

// portapapelesDePrueba deja un wl-paste en el PATH que entrega png1x1.
func portapapelesDePrueba(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("lector de prueba de Linux")
	}
	bin := t.TempDir()
	png := filepath.Join(bin, "img.png")
	os.WriteFile(png, png1x1, 0o644)
	os.WriteFile(filepath.Join(bin, "wl-paste"), []byte("#!/bin/sh\ncat '"+png+"'\n"), 0o755)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("WAYLAND_DISPLAY", "wayland-1")
	t.Setenv("DISPLAY", "")
}

func TestFotoMCP_DevuelveLaImagenParaQueLaIALaVea(t *testing.T) {
	portapapelesDePrueba(t)
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".qdd"), 0o755)
	t.Chdir(root)
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{"de": "portapapeles"}
	res, err := handleFotoTool(context.Background(), req)
	if err != nil || res.IsError {
		t.Fatalf("trae la imagen: %v %+v", err, res)
	}
	var img *mcp.ImageContent
	var texto string
	for _, c := range res.Content {
		if i, ok := mcp.AsImageContent(c); ok {
			img = i
		}
		if tx, ok := mcp.AsTextContent(c); ok {
			texto = tx.Text
		}
	}
	if img == nil || img.MIMEType != "image/png" || img.Data != base64.StdEncoding.EncodeToString(png1x1) {
		t.Fatalf("la imagen va como imagen, no como texto: %+v", img)
	}
	if !strings.Contains(texto, "portapapeles (wl-paste)") || !strings.Contains(texto, filepath.Join("tmp", "fotos")) {
		t.Fatalf("dice de dónde salió y dónde quedó: %q", texto)
	}
}

func TestFotoMCP_SinImagenEsUnErrorQueLoDice(t *testing.T) {
	t.Setenv("WAYLAND_DISPLAY", "")
	t.Setenv("DISPLAY", "")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Chdir(t.TempDir())
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{"de": "portapapeles"}
	res, err := handleFotoTool(context.Background(), req)
	if err != nil || !res.IsError {
		t.Fatalf("sin lector es un error de la tool, no una caída: %v %+v", err, res)
	}
}

func TestDoctor_RevisaConQueSeLeeElPortapapeles(t *testing.T) {
	portapapelesDePrueba(t)
	item := checkFoto()
	if item.Name != nombreFoto || !item.Success || !strings.Contains(item.Error, "wl-paste") {
		t.Fatalf("doctor dice con qué se lee el portapapeles: %+v", item)
	}
}
