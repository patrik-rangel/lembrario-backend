package search

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/meilisearch/meilisearch-go"
)

const indexName = "contents"

// ContentDocument é o schema do documento indexado no Meilisearch
type ContentDocument struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	Type        string    `json:"type"`
	Provider    string    `json:"provider"`
	AuthorName  string    `json:"authorName,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

// Client encapsula o cliente do Meilisearch
type Client struct {
	inner meilisearch.ServiceManager
	index meilisearch.IndexManager
}

// New cria e configura o cliente. Deve ser chamado uma vez na inicialização.
func New() (*Client, error) {
	host := os.Getenv("MEILI_HOST")
	if host == "" {
		host = "http://meilisearch:7700"
	}
	apiKey := os.Getenv("MEILI_MASTER_KEY")

	inner := meilisearch.New(host, meilisearch.WithAPIKey(apiKey))

	// Verifica conectividade
	if _, err := inner.Health(); err != nil {
		return nil, fmt.Errorf("não foi possível conectar ao Meilisearch em %s: %w", host, err)
	}

	c := &Client{
		inner: inner,
		index: inner.Index(indexName),
	}

	if err := c.configure(); err != nil {
		return nil, fmt.Errorf("erro ao configurar índice: %w", err)
	}

	return c, nil
}

// configure define os atributos de busca, filtro e highlight do índice
func (c *Client) configure() error {
	_, err := c.index.UpdateSettings(&meilisearch.Settings{
		// Campos pesquisáveis — por ordem de relevância
		SearchableAttributes: []string{
			"title",
			"description",
			"provider",
			"authorName",
			"url",
		},
		// Campos usados em filtros: GET /search?q=go&filter=type=video
		FilterableAttributes: []string{
			"type",
			"provider",
			"createdAt",
		},
		// Campos usados em ordenação
		SortableAttributes: []string{
			"createdAt",
		},
		// Campos retornados com <em> ao redor do match para o frontend destacar
		DisplayedAttributes: []string{
			"id", "title", "description", "url",
			"type", "provider", "authorName", "createdAt",
		},
	})
	return err
}

func (c *Client) IndexContent(doc ContentDocument) error {
	pk := "id"
	_, err := c.index.AddDocuments([]ContentDocument{doc}, &meilisearch.DocumentOptions{
		PrimaryKey: &pk,
	})
	if err != nil {
		return fmt.Errorf("erro ao indexar documento %s: %w", doc.ID, err)
	}
	return nil
}

func (c *Client) DeleteContent(id string) error {
	pk := "id"
	_, err := c.index.DeleteDocument(id, &meilisearch.DocumentOptions{
		PrimaryKey: &pk,
	})
	if err != nil {
		return fmt.Errorf("erro ao remover documento %s do índice: %w", id, err)
	}
	return nil
}

// SearchResult é o retorno estruturado de uma busca
type SearchResult struct {
	Hits             []SearchHit `json:"hits"`
	TotalHits        int64       `json:"totalHits"`
	ProcessingTimeMs int64       `json:"processingTimeMs"`
	Query            string      `json:"query"`
}

// SearchHit representa um resultado individual com highlights
type SearchHit struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	URL         string                 `json:"url"`
	Type        string                 `json:"type"`
	Provider    string                 `json:"provider"`
	AuthorName  string                 `json:"authorName,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
	Highlights  map[string]interface{} `json:"_formatted,omitempty"`
}

type SearchOptions struct {
	Query  string
	Filter string // ex: "type = video", "provider = youtube.com"
	Limit  int64
	Offset int64
}

// Search executa uma busca no índice
func (c *Client) Search(opts SearchOptions) (*SearchResult, error) {
	if opts.Limit == 0 {
		opts.Limit = 20
	}

	params := &meilisearch.SearchRequest{
		Query:  opts.Query,
		Limit:  opts.Limit,
		Offset: opts.Offset,
		// Envolve os matches com <em> para highlight no frontend
		AttributesToHighlight: []string{"title", "description"},
		HighlightPreTag:       "<mark>",
		HighlightPostTag:      "</mark>",
	}

	if opts.Filter != "" {
		params.Filter = opts.Filter
	}

	resp, err := c.index.Search(opts.Query, params)
	if err != nil {
		return nil, fmt.Errorf("erro na busca por '%s': %w", opts.Query, err)
	}

	result := &SearchResult{
		TotalHits:        resp.TotalHits,
		ProcessingTimeMs: resp.ProcessingTimeMs,
		Query:            opts.Query,
		Hits:             make([]SearchHit, 0, len(resp.Hits)),
	}

	for _, hit := range resp.Hits {
		result.Hits = append(result.Hits, mapToSearchHit(hit))
	}

	return result, nil
}

func mapToSearchHit(m meilisearch.Hit) SearchHit {
	str := func(key string) string {
		raw, ok := m[key]
		if !ok {
			return ""
		}
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return ""
		}
		return s
	}

	hit := SearchHit{
		ID:          str("id"),
		Title:       str("title"),
		Description: str("description"),
		URL:         str("url"),
		Type:        str("type"),
		Provider:    str("provider"),
		AuthorName:  str("authorName"),
	}

	if raw, ok := m["_formatted"]; ok {
		var formatted map[string]interface{}
		if err := json.Unmarshal(raw, &formatted); err == nil {
			hit.Highlights = formatted
		}
	}

	return hit
}
