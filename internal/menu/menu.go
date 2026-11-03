package menu

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// MenuOption representa las acciones posibles seleccionadas por el usuario.
type MenuOption int

const (
	OptionInvalid MenuOption = iota
	OptionInstall
	OptionUninstall
	OptionCancel
)

// ShowMainMenu despliega la interfaz de opciones en la consola y retorna la opción elegida.
func ShowMainMenu() (MenuOption, error) {
	reader := bufio.NewReader(os.Stdin)
	hasInstallations := HasInstalledPrograms()

	fmt.Println("==========================================")
	fmt.Println("    Gestor de JetBrains para Linux        ")
	fmt.Println("==========================================")
	fmt.Println("Selecciona una opción:")
	fmt.Println("  1) Instalar programa (.tar.gz)")

	if hasInstallations {
		fmt.Println("  2) Desinstalar programa")
		fmt.Println("  3) Cancelar")
	} else {
		fmt.Println("  2) Cancelar")
	}
	fmt.Println()

	fmt.Print("Opción: ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return OptionInvalid, fmt.Errorf("error al leer la entrada: %w", err)
	}

	choice := strings.TrimSpace(input)

	switch choice {
	case "1":
		return OptionInstall, nil
	case "2":
		if hasInstallations {
			return OptionUninstall, nil
		}
		fmt.Println("\n[*] Operación cancelada por el usuario.")
		return OptionCancel, nil
	case "3":
		if hasInstallations {
			fmt.Println("\n[*] Operación cancelada por el usuario.")
			return OptionCancel, nil
		}
		return OptionInvalid, fmt.Errorf("opción no válida")
	default:
		return OptionInvalid, fmt.Errorf("opción no válida")
	}
}

// RequestArchivePath solicita interactivamente la ruta del archivo .tar.gz si no se pasó por argumento.
func RequestArchivePath() (string, error) {
	fmt.Print("Arrastra el archivo .tar.gz aquí y presiona Enter: ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("error al leer la ruta: %w", err)
	}

	path := CleanPath(input)
	if path == "" {
		return "", fmt.Errorf("la ruta ingresada está vacía")
	}

	return path, nil
}

// CleanPath remueve comillas y caracteres de escape al arrastrar un archivo a la terminal.
func CleanPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.Trim(path, "'\"")
	path = strings.ReplaceAll(path, "\\ ", " ")
	return path
}

// HasInstalledPrograms comprueba si existen subdirectorios dentro de /opt/jetbrains.
func HasInstalledPrograms() bool {
	baseDir := "/opt/jetbrains"
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if entry.IsDir() {
			return true
		}
	}

	return false
}
