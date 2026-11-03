package app

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

//go:embed assets/jetbrains.svg
var assetsFS embed.FS

// EnsureCategoryExists garantiza que la categoría de menú "JetBrains" y su ícono
// estén registrados correctamente en el sistema.
func EnsureCategoryExists(scriptDir string) error {
	directoryName := "jetbrains.directory"
	targetDirectoryPath := filepath.Join("/usr/share/desktop-directories", directoryName)
	pixmapsIconPath := "/usr/share/pixmaps/jetbrains.svg"
	hicolorIconPath := "/usr/share/icons/hicolor/scalable/apps/jetbrains.svg"

	// 1. Leer el archivo embebido internamente desde el binario
	svgData, err := assetsFS.ReadFile("assets/jetbrains.svg")
	if err == nil {
		// Instalar en /usr/share/pixmaps/
		_ = os.MkdirAll("/usr/share/pixmaps", 0755)
		if err := os.WriteFile(pixmapsIconPath, svgData, 0644); err != nil {
			fmt.Printf("    [!] No se pudo escribir en pixmaps: %v\n", err)
		}

		// Instalar en hicolor scalable
		_ = os.MkdirAll(filepath.Dir(hicolorIconPath), 0755)
		_ = os.WriteFile(hicolorIconPath, svgData, 0644)
	} else {
		fmt.Printf("    [!] Error al leer SVG embebido: %v\n", err)
	}

	// 2. Definir la ruta del ícono para el archivo .directory
	iconValue := "jetbrains"
	if _, err := os.Stat(pixmapsIconPath); err == nil {
		iconValue = pixmapsIconPath
	}

	// 3. Crear o actualizar el archivo .directory
	content := fmt.Sprintf(`[Desktop Entry]
Version=1.0
Type=Directory
Name=JetBrains
Icon=%s
Comment=IDEs y herramientas de desarrollo de JetBrains
`, iconValue)

	_ = os.MkdirAll("/usr/share/desktop-directories", 0755)
	if err := os.WriteFile(targetDirectoryPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("no se pudo crear la categoría de menú en %s: %w", targetDirectoryPath, err)
	}

	// 4. Refrescar la caché de KDE Plasma
	refreshDesktopCache()

	return nil
}

func refreshDesktopCache() {
	if cacheBin, err := exec.LookPath("gtk-update-icon-cache"); err == nil {
		_ = exec.Command(cacheBin, "-f", "-t", "/usr/share/icons/hicolor").Run()
	}

	if kbuild, err := exec.LookPath("kbuildsycoca6"); err == nil {
		_ = exec.Command(kbuild).Run()
	} else if kbuild5, err := exec.LookPath("kbuildsycoca5"); err == nil {
		_ = exec.Command(kbuild5).Run()
	}
}
