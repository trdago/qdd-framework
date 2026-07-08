package auth

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GuardCoreWriteAccess implementa la regla fundacional de que solo el comando "doctor"
// está autorizado para modificar/reparar la carpeta core del framework.
func GuardCoreWriteAccess(targetPath string) error {
	corePath := filepath.Join(".qdd", "core")
	
	if !strings.Contains(targetPath, corePath) {
		return nil
	}

	cmdName := ""
	if len(os.Args) > 1 {
		cmdName = os.Args[1]
	}

	return validateCoreAccess(cmdName, targetPath)
}

func validateCoreAccess(cmdName, targetPath string) error {
	if cmdName == "init" {
		return validateInitAccess(targetPath)
	}

	if !isAuthorizedCommand(cmdName) {
		return fmt.Errorf("[Access Denied] Modificación del Core rechazada. Comando origen '%s' no autorizado. Usa 'qdd doctor' para reparar o modificar configuraciones internas del framework", cmdName)
	}

	return nil
}

func validateInitAccess(targetPath string) error {
	if _, err := os.Stat(targetPath); err == nil {
		return fmt.Errorf("[Access Denied] init no puede sobrescribir archivos del core existentes. Use 'qdd doctor --fix' para reparaciones")
	}
	return nil
}

func isAuthorizedCommand(cmdName string) bool {
	if cmdName == "doctor" || cmdName == "test" || strings.HasSuffix(cmdName, ".test") {
		return true
	}
	return false
}
