package app

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// RunUninstaller maneja el flujo de desinstalación interactiva de programas JetBrains.
func RunUninstaller() error {
	baseDir := "/opt/jetbrains"

	entries, err := os.ReadDir(baseDir)
	if err != nil || len(entries) == 0 {
		return fmt.Errorf("no se encontraron instalaciones de JetBrains en %s", baseDir)
	}

	// 1. Filtrar solo los directorios válidos
	var installedFolders []os.DirEntry
	for _, entry := range entries {
		if entry.IsDir() {
			installedFolders = append(installedFolders, entry)
		}
	}

	if len(installedFolders) == 0 {
		return fmt.Errorf("no hay programas instalados para desinstalar")
	}

	// 2. Mostrar la lista de programas instalados
	fmt.Println("Programas instalados:")
	for i, folder := range installedFolders {
		fmt.Printf("  %d) %s\n", i+1, folder.Name())
	}
	fmt.Println()

	// 3. Solicitar la selección al usuario
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Selecciona el número del programa que deseas desinstalar: ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("error al leer la entrada: %w", err)
	}

	input = strings.TrimSpace(input)
	selectedIndex, err := strconv.Atoi(input)
	if err != nil || selectedIndex < 1 || selectedIndex > len(installedFolders) {
		return fmt.Errorf("selección no válida")
	}

	selectedFolder := installedFolders[selectedIndex-1].Name()
	targetDir := filepath.Join(baseDir, selectedFolder)

	pInfo, err := readProductInfo(targetDir)
	if err != nil {
		return fmt.Errorf("error al leer product-info.json: %w", err)
	}

	// 4. Obtener nombre oficial o deducido para eliminar su acceso directo .desktop
	appName := getAppName(pInfo, selectedFolder)
	desktopFileName := strings.ToLower(strings.ReplaceAll(appName, " ", "-")) + ".desktop"
	desktopFilePath := filepath.Join("/usr/share/applications", desktopFileName)

	// 5. Confirmación final
	fmt.Printf("\n[!] ¿Estás seguro de que deseas eliminar '%s'? (s/N): ", selectedFolder)
	confirmInput, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("error al leer la confirmación: %w", err)
	}

	confirm := strings.ToLower(strings.TrimSpace(confirmInput))
	if confirm != "s" && confirm != "si" && confirm != "sí" {
		fmt.Println("[*] Operación cancelada.")
		return nil
	}

	// 6. Eliminar el directorio del programa
	fmt.Printf("[+] Eliminando directorio: %s...\n", targetDir)
	if err := os.RemoveAll(targetDir); err != nil {
		return fmt.Errorf("error al eliminar la carpeta del programa: %w", err)
	}

	// 7. Limpiar el archivo .desktop asociado si existe
	if _, err := os.Stat(desktopFilePath); err == nil {
		fmt.Printf("[+] Eliminando acceso directo: %s...\n", desktopFilePath)
		if err := os.Remove(desktopFilePath); err != nil {
			fmt.Printf("    [!] Advertencia: no se pudo eliminar el archivo .desktop: %v\n", err)
		}
	}

	fmt.Println("==================================================")
	fmt.Println(" [✓] Desinstalación completada con éxito.")
	fmt.Printf(" [*] Se ha eliminado: %s\n", appName)
	fmt.Println("==================================================")

	return nil
}
