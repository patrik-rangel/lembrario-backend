package worker

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// ScrapeURL extrai metadados de uma URL
func ScrapeURL(ctx context.Context, targetURL string) (*ScrapedData, error) {
	// Validar URL
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("URL inválida: %w", err)
	}

	// Criar cliente HTTP com timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Criar request com contexto
	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar request: %w", err)
	}

	// Adicionar User-Agent para evitar bloqueios
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; LembrarioBot/1.0)")

	// Fazer request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status HTTP não OK: %d", resp.StatusCode)
	}

	// Ler body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler response body: %w", err)
	}

	// Parse HTML
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer parse do HTML: %w", err)
	}

	// Extrair metadados
	data := &ScrapedData{
		Provider: parsedURL.Host,
	}

	extractMetadata(doc, data)

	// Se não conseguiu extrair título, usar a URL como fallback
	if data.Title == "" {
		data.Title = targetURL
	}

	return data, nil
}

// extractMetadata extrai metadados do documento HTML
func extractMetadata(n *html.Node, data *ScrapedData) {
	if n.Type == html.ElementNode {
		switch n.Data {
		case "title":
			if data.Title == "" && n.FirstChild != nil {
				data.Title = strings.TrimSpace(n.FirstChild.Data)
			}
		case "meta":
			extractMetaTag(n, data)
		}
	}

	// Recursivamente processar filhos
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractMetadata(c, data)
	}
}

// extractMetaTag extrai informações de tags meta
func extractMetaTag(n *html.Node, data *ScrapedData) {
	var name, property, content string

	for _, attr := range n.Attr {
		switch attr.Key {
		case "name":
			name = attr.Val
		case "property":
			property = attr.Val
		case "content":
			content = attr.Val
		}
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return
	}

	// Open Graph tags
	switch property {
	case "og:title":
		if data.Title == "" {
			data.Title = content
		}
	case "og:description":
		if data.Description == "" {
			data.Description = content
		}
	}

	// Meta tags padrão
	switch name {
	case "description":
		if data.Description == "" {
			data.Description = content
		}
	case "twitter:title":
		if data.Title == "" {
			data.Title = content
		}
	case "twitter:description":
		if data.Description == "" {
			data.Description = content
		}
	}
}
