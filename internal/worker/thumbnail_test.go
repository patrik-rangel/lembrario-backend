package worker_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"lembrario-backend/internal/worker"
)

func createImgServer(status int, contentType string, body []byte) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(status)
		w.Write(body)
	}))
}

func TestInferExtension(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		url         string
		want        string
	}{
		{"Pelo Content-Type JPEG", "image/jpeg", "http://ex.com/img", ".jpg"},
		{"Pelo Content-Type PNG", "image/png", "http://ex.com/img", ".png"},
		{"Pela URL (fallback)", "text/plain", "http://ex.com/photo.webp?size=small", ".webp"},
		{"Default para JPG", "application/octet-stream", "http://ex.com/unknown", ".jpg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Nota: Como inferExtension não é exportada, você pode testar 
			// via downloadThumbnail ou torná-la exportada (InferExtension)
			got := worker.InferExtension(tt.contentType, tt.url)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDownloadThumbnail(t *testing.T) {
	// Setup: Criar diretório temporário para não sujar o sistema
	tempDir := t.TempDir()
	
	// Precisamos "injetar" esse diretório no worker. 
	// Como a variável thumbnailDir é global no pacote, vamos setar o ENV:
	os.Setenv("THUMBNAIL_DIR", tempDir)

	tests := []struct {
		name      string
		imageURL  string
		status    int
		body      []byte
		wantErr   bool
		checkFile bool
	}{
		{
			name:      "Download com sucesso",
			imageURL:  "/valid.png",
			status:    http.StatusOK,
			body:      []byte("fake-image-data"),
			wantErr:   false,
			checkFile: true,
		},
		{
			name:      "URL vazia retorna string vazia sem erro",
			imageURL:  "",
			wantErr:   false,
			checkFile: false,
		},
		{
			name:      "Erro 404 no servidor de imagem",
			imageURL:  "/404.jpg",
			status:    http.StatusNotFound,
			wantErr:   true,
			checkFile: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var finalURL string
			if tt.imageURL != "" {
				server := createImgServer(tt.status, "image/png", tt.body)
				defer server.Close()
				finalURL = server.URL + tt.imageURL
			}

			contentID := "test-id-" + tt.name
			path, err := worker.DownloadThumbnail(context.Background(), contentID, finalURL)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, path)
			} else {
				assert.NoError(t, err)
				if tt.checkFile {
					assert.Contains(t, path, contentID)
					// Verifica se o arquivo realmente existe no disco
					_, err := os.Stat(path)
					assert.NoError(t, err, "Arquivo deveria existir no path: %s", path)
				}
			}
		})
	}
}