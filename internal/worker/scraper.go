package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

func ScrapeURL(ctx context.Context, targetURL string) (*ScrapedData, error) {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("URL inválida: %w", err)
	}

	host := strings.ToLower(strings.TrimPrefix(parsed.Host, "www."))

	switch {
	case host == "youtube.com" || host == "youtu.be":
		return scrapeYouTube(ctx, parsed)
	case host == "github.com":
		return scrapeGitHub(ctx, parsed)
	default:
		return scrapeGeneric(ctx, targetURL)
	}
}

func scrapeYouTube(ctx context.Context, parsed *url.URL) (*ScrapedData, error) {
	videoID := parsed.Query().Get("v")
	if videoID == "" {
		videoID = strings.TrimPrefix(parsed.Path, "/")
	}
	if videoID == "" {
		return nil, fmt.Errorf("não foi possível extrair video ID da URL")
	}

	apiKey := os.Getenv("YOUTUBE_API_KEY")
	if apiKey == "" {
		fmt.Printf("⚠️ YOUTUBE_API_KEY não configurada, usando fallback genérico para videoID=%s", videoID)
		return scrapeGeneric(ctx, parsed.String())
	}

	apiURL := fmt.Sprintf(
		"https://www.googleapis.com/youtube/v3/videos?id=%s&part=snippet,contentDetails&key=%s",
		videoID, apiKey,
	)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar request YouTube: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro na YouTube API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler resposta YouTube: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("❌ YouTube API status %d: %s", resp.StatusCode, string(body))
		// Tenta o fallback genérico antes de desistir
		fmt.Printf("⚠️ Tentando fallback genérico para %s", parsed.String())
		return scrapeGeneric(ctx, parsed.String())
	}

	var result struct {
		Items []struct {
			Snippet struct {
				Title        string `json:"title"`
				Description  string `json:"description"`
				ChannelTitle string `json:"channelTitle"`
				Thumbnails   struct {
					Maxres struct {
						URL string `json:"url"`
					} `json:"maxres"`
					High struct {
						URL string `json:"url"`
					} `json:"high"`
				} `json:"thumbnails"`
			} `json:"snippet"`
			ContentDetails struct {
				Duration string `json:"duration"` // formato ISO 8601: PT4M13S
			} `json:"contentDetails"`
		} `json:"items"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta YouTube: %w", err)
	}

	// API key inválida retorna 200 com campo "error"
	if result.Error != nil {
		fmt.Printf("❌ YouTube API erro %d: %s", result.Error.Code, result.Error.Message)
		return scrapeGeneric(ctx, parsed.String())
	}

	if len(result.Items) == 0 {
		fmt.Printf("⚠️ YouTube API: nenhum item para videoID=%s, tentando fallback", videoID)
		return scrapeGeneric(ctx, parsed.String())
	}

	item := result.Items[0]
	snippet := item.Snippet

	// Prefere maxres, cai para high se não disponível
	thumbnailURL := snippet.Thumbnails.Maxres.URL
	if thumbnailURL == "" {
		thumbnailURL = snippet.Thumbnails.High.URL
	}

	return &ScrapedData{
		Title:        snippet.Title,
		Description:  snippet.Description,
		Provider:     "youtube.com",
		ThumbnailURL: thumbnailURL,
		AuthorName:   snippet.ChannelTitle,
		Duration:     item.ContentDetails.Duration, // ex: "PT4M13S"
	}, nil
}

func scrapeGeneric(ctx context.Context, targetURL string) (*ScrapedData, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 0 {
				for k, v := range via[0].Header {
					req.Header[k] = v
				}
			}
			if len(via) >= 10 {
				return fmt.Errorf("muitos redirecionamentos")
			}
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar request: %w", err)
	}

	// Headers completos que browsers reais enviam
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "pt-BR,pt;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status HTTP não OK: %d para URL: %s", resp.StatusCode, targetURL)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer parse do HTML: %w", err)
	}

	data := &ScrapedData{Provider: resp.Request.URL.Host}

	// Prioridade: og:title > twitter:title > <title>
	if v := doc.Find(`meta[property="og:title"]`).AttrOr("content", ""); v != "" {
		data.Title = v
	} else if v := doc.Find(`meta[name="twitter:title"]`).AttrOr("content", ""); v != "" {
		data.Title = v
	} else {
		data.Title = strings.TrimSpace(doc.Find("title").First().Text())
	}

	// Prioridade: og:description > twitter:description > meta description
	if v := doc.Find(`meta[property="og:description"]`).AttrOr("content", ""); v != "" {
		data.Description = v
	} else if v := doc.Find(`meta[name="twitter:description"]`).AttrOr("content", ""); v != "" {
		data.Description = v
	} else {
		data.Description = doc.Find(`meta[name="description"]`).AttrOr("content", "")
	}

	// Thumbnail via og:image
	data.ThumbnailURL = doc.Find(`meta[property="og:image"]`).AttrOr("content", "")

	return data, nil
}

// scrapeGitHub usa a API pública do GitHub para evitar bloqueios
func scrapeGitHub(ctx context.Context, parsed *url.URL) (*ScrapedData, error) {
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	client := &http.Client{Timeout: 10 * time.Second}

	var apiURL string
	switch len(parts) {
	case 1:
		apiURL = fmt.Sprintf("https://api.github.com/users/%s", parts[0])
	case 2:
		apiURL = fmt.Sprintf("https://api.github.com/repos/%s/%s", parts[0], parts[1])
	default:
		return scrapeGeneric(ctx, parsed.String())
	}

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar request GitHub API: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Lembrario/1.0")

	// Injeta o token se disponível — eleva rate limit de 60 para 5000 req/hora
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	} else {
		fmt.Println("⚠️ GITHUB_TOKEN não configurado — rate limit reduzido (60 req/hora)")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro na GitHub API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler resposta GitHub: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("❌ GitHub API status %d para %s: %s", resp.StatusCode, apiURL, string(body))
		return nil, fmt.Errorf("GitHub API retornou status %d", resp.StatusCode)
	}

	switch len(parts) {
	case 1:
		var ghUser struct {
			Name      string `json:"name"`
			Login     string `json:"login"`
			Bio       string `json:"bio"`
			AvatarURL string `json:"avatar_url"`
		}
		if err := json.Unmarshal(body, &ghUser); err != nil {
			return nil, fmt.Errorf("erro ao decodificar usuário GitHub: %w", err)
		}
		name := ghUser.Name
		if name == "" {
			name = ghUser.Login
		}
		return &ScrapedData{
			Title:        fmt.Sprintf("%s - GitHub", name),
			Description:  ghUser.Bio,
			Provider:     "github.com",
			ThumbnailURL: ghUser.AvatarURL, // foto de perfil do usuário
		}, nil

	case 2:
		var ghRepo struct {
			FullName    string `json:"full_name"`
			Description string `json:"description"`
			Owner       struct {
				AvatarURL string `json:"avatar_url"`
			} `json:"owner"`
		}
		if err := json.Unmarshal(body, &ghRepo); err != nil {
			return nil, fmt.Errorf("erro ao decodificar repo GitHub: %w", err)
		}
		return &ScrapedData{
			Title:        fmt.Sprintf("%s - GitHub", ghRepo.FullName),
			Description:  ghRepo.Description,
			Provider:     "github.com",
			ThumbnailURL: ghRepo.Owner.AvatarURL,
		}, nil
	}

	return nil, fmt.Errorf("caso inesperado no scrapeGitHub")
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
