package worker

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var thumbnailDir = getEnvOrDefault("THUMBNAIL_DIR", "uploads/thumbnails")

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// downloadThumbnail baixa a imagem de uma URL e salva em disco.
// Retorna o path relativo salvo, ou string vazia se não houver URL.
func downloadThumbnail(ctx context.Context, contentID, imageURL string) (string, error) {
	if imageURL == "" {
		return "", nil
	}

	if err := os.MkdirAll(thumbnailDir, 0755); err != nil {
		return "", fmt.Errorf("erro ao criar diretório de thumbnails: %w", err)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", imageURL, nil)
	if err != nil {
		return "", fmt.Errorf("erro ao criar request de thumbnail: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("erro ao baixar thumbnail: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("thumbnail retornou status %d", resp.StatusCode)
	}

	ext := inferExtension(resp.Header.Get("Content-Type"), imageURL)
	filename := fmt.Sprintf("%s%s", contentID, ext)
	destPath := filepath.Join(thumbnailDir, filename)

	f, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("erro ao criar arquivo de thumbnail: %w", err)
	}
	defer f.Close()

	// Limita a 5MB para evitar imagens gigantes
	limited := io.LimitReader(resp.Body, 5<<20)
	if _, err := io.Copy(f, limited); err != nil {
		os.Remove(destPath) // cleanup em caso de falha
		return "", fmt.Errorf("erro ao salvar thumbnail: %w", err)
	}

	log.Printf("🖼️ Thumbnail salvo: %s", destPath)
	return destPath, nil
}

// inferExtension tenta descobrir a extensão pelo Content-Type ou pela URL.
func inferExtension(contentType, imageURL string) string {
	switch {
	case strings.Contains(contentType, "jpeg"), strings.Contains(contentType, "jpg"):
		return ".jpg"
	case strings.Contains(contentType, "png"):
		return ".png"
	case strings.Contains(contentType, "webp"):
		return ".webp"
	case strings.Contains(contentType, "gif"):
		return ".gif"
	}
	// Fallback: tenta extrair da URL
	url := strings.ToLower(strings.Split(imageURL, "?")[0])
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".webp", ".gif"} {
		if strings.HasSuffix(url, ext) {
			return ext
		}
	}
	return ".jpg" // default seguro
}
