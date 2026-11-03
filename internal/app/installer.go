package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// InstallParams contiene los parámetros necesarios para la instalación.
type InstallParams struct {
	ArchivePath string
	ScriptDir   string
}

// ProductInfo mapea los metadatos contenidos en product-info.json de JetBrains.
type ProductInfo struct {
	Name                    string `json:"name"`
	Version                 string `json:"version"`
	BuildNumber             string `json:"buildNumber"`
	ProductCode             string `json:"productCode"`
	EnvVarBaseName          string `json:"envVarBaseName"`
	DataDirectoryName       string `json:"dataDirectoryName"`
	SvgIconPath             string `json:"svgIconPath"`
	ProductVendor           string `json:"productVendor"`
	MajorVersionReleaseDate string `json:"majorVersionReleaseDate"`
	MinRequiredJavaVersion  int    `json:"minRequiredJavaVersion"`
}

// readProductInfo lee y parsea el archivo product-info.json desde el directorio instalado.
func readProductInfo(targetDir string) (*ProductInfo, error) {
	path := filepath.Join(targetDir, "product-info.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var info ProductInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("error al decodificar product-info.json: %w", err)
	}

	return &info, nil
}

// RunInstaller ejecuta el flujo completo de instalación de un IDE de JetBrains.
func RunInstaller(params InstallParams) error {
	archivePath := params.ArchivePath

	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		return fmt.Errorf("el archivo '%s' no existe o no es válido", archivePath)
	}

	// 1. Garantizar que la categoría de menú exista en el sistema
	if err := EnsureCategoryExists(params.ScriptDir); err != nil {
		fmt.Printf("    [!] Advertencia al crear la categoría de menú: %v\n", err)
	}

	// 2. Crear directorio temporal para descompresión
	tempDir, err := os.MkdirTemp("", "jb_extract_*")
	if err != nil {
		return fmt.Errorf("no se pudo crear directorio temporal: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// 3. Extracción asíncrona con animación de spinner
	if err := ExtractArchiveWithSpinner(archivePath, tempDir); err != nil {
		return err
	}

	// 4. Identificar la carpeta raíz extraída
	extractedFolder, err := GetSingleSubdirectory(tempDir)
	if err != nil {
		return fmt.Errorf("error identificando carpeta extraída: %w", err)
	}

	folderName := filepath.Base(extractedFolder)
	targetDir := filepath.Join("/opt/jetbrains", folderName)

	if err := os.MkdirAll("/opt/jetbrains", 0755); err != nil {
		return fmt.Errorf("error al crear /opt/jetbrains: %w", err)
	}

	_ = os.RemoveAll(targetDir)

	// Instalar/Mover archivos con animación asíncrona
	if err := moveOrCopyDirWithSpinner(extractedFolder, targetDir); err != nil {
		return fmt.Errorf("error al mover a la ubicación final: %w", err)
	}

	// 5. Leer metadatos desde product-info.json
	pInfo, err := readProductInfo(targetDir)
	if err != nil {
		fmt.Printf("[!] No se pudo procesar product-info.json (%v). Usando métodos de búsqueda manual...\n", err)
	}

	// 6. Detectar ejecutable usando envVarBaseName como pista
	var baseNameHint string
	if pInfo != nil && pInfo.EnvVarBaseName != "" {
		baseNameHint = strings.ToLower(pInfo.EnvVarBaseName)
	}

	execPath, appBinName, err := resolveExecutable(targetDir, baseNameHint)
	if err != nil {
		return err
	}

	// 7. Localizar el ícono de la aplicación
	iconPath := findAppIcon(targetDir, pInfo)

	// 8. Determinar el nombre oficial de la aplicación
	appName := getAppName(pInfo, folderName)

	// 9. Crear el acceso directo .desktop en /usr/share/applications
	desktopFileName := strings.ToLower(strings.ReplaceAll(appName, " ", "-")) + ".desktop"
	desktopFilePath := filepath.Join("/usr/share/applications", desktopFileName)

	fmt.Printf("[+] Generando acceso directo: %s\n", desktopFilePath)
	if err := createDesktopShortcut(desktopFilePath, appName, execPath, iconPath, appBinName, pInfo); err != nil {
		return fmt.Errorf("error al crear el archivo .desktop: %w", err)
	}

	fmt.Println("==================================================")
	fmt.Println(" [✓] ¡Instalación completada con éxito!")
	fmt.Printf(" [*] Programa: %s\n", appName)
	if pInfo != nil && pInfo.Version != "" {
		fmt.Printf(" [*] Versión: %s (Build %s)\n", pInfo.Version, pInfo.BuildNumber)
	}
	fmt.Printf(" [*] Ejecutable: %s\n", execPath)
	fmt.Printf(" [*] Menú: JetBrains -> %s\n", appName)
	fmt.Println("==================================================")

	return nil
}

// resolveExecutable busca el ejecutable en bin/ usando envVarBaseName si está disponible.
func resolveExecutable(targetDir, baseNameHint string) (execPath string, binName string, err error) {
	binDir := filepath.Join(targetDir, "bin")

	if baseNameHint != "" {
		nativeExec := filepath.Join(binDir, baseNameHint)
		shScript := filepath.Join(binDir, baseNameHint+".sh")

		// Preferir binario nativo si existe
		if info, e := os.Stat(nativeExec); e == nil && !info.IsDir() {
			fmt.Printf("[+] Binario nativo detectado por product-info: %s\n", nativeExec)
			return nativeExec, baseNameHint, nil
		}

		// Si no hay binario nativo, probar script .sh (ej. pycharm.sh)
		if info, e := os.Stat(shScript); e == nil && !info.IsDir() {
			fmt.Printf("[+] Script de arranque detectado por product-info: %s\n", shScript)
			return shScript, baseNameHint, nil
		}
	}

	// Respaldar a la búsqueda heurística si la pista falla o product-info no existe
	shScript, err := findShScript(binDir)
	if err != nil {
		return "", "", fmt.Errorf("no se encontró script ejecutable: %w", err)
	}

	appBinName := strings.TrimSuffix(filepath.Base(shScript), ".sh")
	nativeExec := filepath.Join(binDir, appBinName)

	if info, e := os.Stat(nativeExec); e == nil && !info.IsDir() {
		fmt.Printf("[+] Binario nativo detectado (búsqueda): %s\n", nativeExec)
		return nativeExec, appBinName, nil
	}

	fmt.Printf("[!] Usando script de arranque (búsqueda): %s\n", shScript)
	return shScript, appBinName, nil
}

// moveOrCopyDirWithSpinner traslada los archivos ejecutando un spinner interactivo mientras copia.
func moveOrCopyDirWithSpinner(src, dst string) error {
	errChan := make(chan error, 1)
	doneChan := make(chan struct{})

	go func() {
		if err := moveOrCopyDir(src, dst); err != nil {
			errChan <- err
			return
		}
		close(doneChan)
	}()

	spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	i := 0
	fmt.Printf("[+] Instalando en %s ", dst)

	for {
		select {
		case err := <-errChan:
			fmt.Println("\n[X] Falló la copia de archivos.")
			return err
		case <-doneChan:
			fmt.Println("\r[✓] Archivos instalados correctamente en /opt/jetbrains.   ")
			return nil
		default:
			fmt.Printf("\r[+] Instalando en %s %s", dst, spinner[i%len(spinner)])
			i++
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// moveOrCopyDir intenta mover la carpeta con os.Rename y realiza un fallback a copia si están en distintas particiones.
func moveOrCopyDir(src, dst string) error {
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	if err := copyDir(src, dst); err != nil {
		return err
	}

	return os.RemoveAll(src)
}

// copyDir realiza la copia recursiva de un directorio a otro.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		return copyFile(path, targetPath, info.Mode())
	})
}

// copyFile copia un archivo preservando sus permisos.
func copyFile(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Sync()
}

// findShScript busca el script .sh de arranque dentro de la carpeta bin/.
func findShScript(binDir string) (string, error) {
	var candidateScript string

	ignoredScripts := map[string]bool{
		"inspect.sh":           true,
		"format.sh":            true,
		"ltedit.sh":            true,
		"remote-dev-server.sh": true,
		"install.sh":           true,
		"jetbrains_client.sh":  true,
	}

	err := filepath.Walk(binDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		name := info.Name()

		if strings.HasSuffix(name, ".sh") && !ignoredScripts[name] && !strings.HasPrefix(name, "fsnotifier") {
			candidateScript = path
			if !strings.Contains(name, "install") && !strings.Contains(name, "setup") {
				return filepath.SkipAll
			}
		}
		return nil
	})

	if candidateScript == "" {
		_ = filepath.Walk(binDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}

			if info.Mode()&0111 != 0 && !strings.HasPrefix(info.Name(), "fsnotifier") && filepath.Ext(path) == "" {
				candidateScript = path
				return filepath.SkipAll
			}
			return nil
		})
	}

	if candidateScript == "" {
		return "", fmt.Errorf("no se encontró ningún script de arranque o binario ejecutable en %s", binDir)
	}

	return candidateScript, err
}

// findAppIcon busca la imagen visual utilizando svgIconPath o realizando un escaneo.
func findAppIcon(targetDir string, pInfo *ProductInfo) string {
	if pInfo != nil && pInfo.SvgIconPath != "" {
		iconPath := filepath.Join(targetDir, pInfo.SvgIconPath)
		if _, err := os.Stat(iconPath); err == nil {
			return iconPath
		}
	}

	var icon string
	binDir := filepath.Join(targetDir, "bin")

	_ = filepath.Walk(binDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if (ext == ".svg" || ext == ".png") && !strings.Contains(info.Name(), "_small") {
			icon = path
			return filepath.SkipAll
		}
		return nil
	})

	if icon != "" {
		return icon
	}

	_ = filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext == ".svg" || ext == ".png" {
			icon = path
			return filepath.SkipAll
		}
		return nil
	})

	return icon
}

// getAppName obtiene el nombre desde ProductInfo o genera uno legible basado en la carpeta.
func getAppName(pInfo *ProductInfo, fallbackFolder string) string {
	if pInfo != nil && pInfo.Name != "" {
		return pInfo.Name
	}

	clean := strings.ReplaceAll(fallbackFolder, "-", " ")
	words := strings.Fields(clean)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// createDesktopShortcut genera el acceso directo .desktop aprovechando la información obtenida.
func createDesktopShortcut(filePath, name, execPath, iconPath, wmClass string, pInfo *ProductInfo) error {
	comment := "IDE de JetBrains / Desarrollo"
	if pInfo != nil && pInfo.Version != "" {
		comment = fmt.Sprintf("%s %s - %s (Build %s)", pInfo.ProductVendor, pInfo.Name, pInfo.Version, pInfo.BuildNumber)
	}

	content := fmt.Sprintf(`[Desktop Entry]
Version=1.0
Type=Application
Name=%s
Icon=%s
Exec="%s" %%f
Comment=%s
Categories=JETBRAINS;Development;IDE;
Terminal=false
StartupWMClass=%s
`, name, iconPath, execPath, comment, wmClass)

	return os.WriteFile(filePath, []byte(content), 0755)
}
