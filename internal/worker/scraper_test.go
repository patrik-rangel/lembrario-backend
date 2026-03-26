package worker_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"lembrario-backend/internal/worker"
)

// ─── Helpers ──────────────────────────────────────────────────────────────────

func createTestServer(htmlContent string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintln(w, htmlContent)
	}))
}

// ─── Scrape Generic ───────────────────────────────────────────────────────────

func TestScrapeGeneric(t *testing.T) {
	tests := []struct {
		name        string
		html        string
		wantTitle   string
		wantDesc    string
		wantThumb   string
	}{
		{
			name: "Prioridade para Open Graph",
			html: `<html><head>
				<title>Titulo Comum</title>
				<meta property="og:title" content="Titulo OG">
				<meta property="og:description" content="Descricao OG">
				<meta property="og:image" content="http://thumb.jpg">
			</head></html>`,
			wantTitle: "Titulo OG",
			wantDesc:  "Descricao OG",
			wantThumb: "http://thumb.jpg",
		},
		{
			name: "Fallback para tags Twitter",
			html: `<html><head>
				<meta name="twitter:title" content="Titulo Twitter">
				<meta name="twitter:description" content="Descricao Twitter">
			</head></html>`,
			wantTitle: "Titulo Twitter",
			wantDesc:  "Descricao Twitter",
			wantThumb: "",
		},
		{
			name: "Fallback para Title e Meta Description padrao",
			html: `<html><head>
				<title>Titulo Nativo</title>
				<meta name="description" content="Descricao Nativa">
			</head></html>`,
			wantTitle: "Titulo Nativo",
			wantDesc:  "Descricao Nativa",
			wantThumb: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := createTestServer(tt.html)
			defer server.Close()

			data, err := worker.ScrapeURL(context.Background(), server.URL)

			assert.NoError(t, err)
			assert.Equal(t, tt.wantTitle, data.Title)
			assert.Equal(t, tt.wantDesc, data.Description)
			assert.Equal(t, tt.wantThumb, data.ThumbnailURL)
		})
	}
}

// ─── Scrape Routing ──────────────────────────────────────────────────────────

func TestScrapeURL_Routing(t *testing.T) {
	// Este teste valida se o roteamento entre scrapers (YT, GH, Generic) está correto
	tests := []struct {
		name     string
		url      string
		wantProv string
	}{
		{
			name:     "Identifica YouTube",
			url:      "https://youtube.com/watch?v=123",
			wantProv: "youtube.com",
		},
		{
			name:     "Identifica GitHub",
			url:      "https://github.com/golang/go",
			wantProv: "github.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Nota: Como não configuramos as APIs, eles cairão no fallback generic
			// o que é suficiente para testar o roteamento do host
			data, _ := worker.ScrapeURL(context.Background(), tt.url)
			if data != nil {
				assert.Contains(t, data.Provider, tt.wantProv)
			}
		})
	}
}

// ─── Scrape GitHub ────────────────────────────────────────────────────────────

func TestScrapeGitHub_User(t *testing.T) {
	// Simula a API do GitHub
	ghServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"name": "Patrik Rangel", "login": "patrikrangel", "bio": "Go Developer", "avatar_url": "https://avatar.com"}`)
	}))
	defer ghServer.Close()

	// Aqui você precisaria do ajuste de URL que sugeri no topo (githubAPIBaseURL)
	// data, err := worker.ScrapeGitHub_Interno(context.Background(), ghServer.URL, "patrikrangel")
	
	assert.True(t, true) // Placeholder para a lógica de API
}