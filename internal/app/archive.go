package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// ExtractArchiveWithSpinner ejecuta la extracción del paquete .tar.gz en segundo plano
// y muestra una animación de spinner en la terminal mientras finaliza.
func ExtractArchiveWithSpinner(archivePath, destDir string) error {
	errChan := make(chan error, 1)
	doneChan := make(chan struct{})

	// 1. Ejecutar la descompresión pesada en una Goroutine (ASÍNCRONO)
	go func() {
		if err := extractArchive(archivePath, destDir); err != nil {
			errChan <- fmt.Errorf("error durante la extracción: %w", err)
			return
		}
		close(doneChan)
	}()

	// 2. Animación de Spinner en el hilo principal
	spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	i := 0
	fileName := filepath.Base(archivePath)
	fmt.Printf("[+] Extrayendo %s ", fileName)

	for {
		select {
		case err := <-errChan:
			fmt.Println("\n[X] Falló la extracción.")
			return err
		case <-doneChan:
			fmt.Println("\r[✓] Extracción finalizada con éxito.   ")
			return nil
		default:
			fmt.Printf("\r[+] Extrayendo %s %s", fileName, spinner[i%len(spinner)])
			i++
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// extractArchive realiza la descompresión física usando 'tar' o '7z' según estén disponibles en el sistema.
func extractArchive(archivePath, destDir string) error {
	// Intentar extracción directa con 'tar'
	if _, err := exec.LookPath("tar"); err == nil {
		cmd := exec.Command("tar", "-xzf", archivePath, "-C", destDir)
		return cmd.Run()
	}

	// Fallback usando '7z' concatenado con 'tar'
	if decompressor, err := exec.LookPath("7z"); err == nil {
		cmd1 := exec.Command(decompressor, "x", archivePath, "-so")
		cmd2 := exec.Command("tar", "-xf", "-", "-C", destDir)

		// Conectar la salida de 7z a la entrada de tar
		pipe, err := cmd1.StdoutPipe()
		if err != nil {
			return fmt.Errorf("error al crear tubería de descompresión: %w", err)
		}
		cmd2.Stdin = pipe

		if err := cmd1.Start(); err != nil {
			return fmt.Errorf("error al iniciar 7z: %w", err)
		}
		if err := cmd2.Start(); err != nil {
			return fmt.Errorf("error al iniciar tar: %w", err)
		}

		if err := cmd1.Wait(); err != nil {
			return fmt.Errorf("error durante descompresión con 7z: %w", err)
		}
		return cmd2.Wait()
	}

	return fmt.Errorf("no se encontró 'tar' ni '7z' en el sistema")
}

// GetSingleSubdirectory identifica y retorna la carpeta principal resultante de la descompresión.
func GetSingleSubdirectory(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("error al leer directorio temporal: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			return filepath.Join(dir, entry.Name()), nil
		}
	}
	return "", fmt.Errorf("no se encontró ningún directorio dentro de la descompresión")
}
