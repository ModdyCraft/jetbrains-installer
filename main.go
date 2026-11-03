package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"jetbrains-installer/internal/app"
	"jetbrains-installer/internal/menu"
)

// ensureRoot verifica si la aplicación se ejecuta con privilegios de superusuario.
func ensureRoot() {
	if os.Geteuid() != 0 {
		fmt.Println("[!] Este programa requiere permisos de administrador.")
		fmt.Println("    Reejecutando con sudo...")

		cmd := exec.Command("sudo", os.Args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			fmt.Printf("[X] Error al elevar privilegios: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}
}

func main() {
	ensureRoot()

	exePath, err := os.Executable()
	if err != nil {
		exePath = "."
	}
	scriptDir := filepath.Dir(exePath)

	option, err := menu.ShowMainMenu()
	if err != nil {
		fmt.Printf("\n[X] Error: %v\n", err)
		os.Exit(1)
	}

	switch option {
	case menu.OptionInstall:
		var archivePath string

		// Si se pasó una ruta por argumento en la terminal (ej: sudo ./installer /ruta/archivo.tar.gz)
		if len(os.Args) > 1 {
			archivePath = menu.CleanPath(os.Args[1])
		}

		// Si no se proporcionó por CLI, solicitarla interactivamente al usuario
		if archivePath == "" {
			fmt.Println()
			path, err := menu.RequestArchivePath()
			if err != nil {
				fmt.Printf("\n[X] Error: %v\n", err)
				os.Exit(1)
			}
			archivePath = path
		}

		params := app.InstallParams{
			ArchivePath: archivePath,
			ScriptDir:   scriptDir,
		}

		fmt.Println()
		if err := app.RunInstaller(params); err != nil {
			fmt.Printf("\n[X] Error en la instalación: %v\n", err)
			os.Exit(1)
		}

	case menu.OptionUninstall:
		fmt.Println()
		if err := app.RunUninstaller(); err != nil {
			fmt.Printf("\n[X] Error en la desinstalación: %v\n", err)
			os.Exit(1)
		}

	case menu.OptionCancel:
		os.Exit(0)

	default:
		fmt.Println("\n[*] Opción no válida.")
		os.Exit(1)
	}
}
