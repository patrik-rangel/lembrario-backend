package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"lembrario-backend/internal/api"
	"lembrario-backend/internal/db"
	"lembrario-backend/internal/search"
	"lembrario-backend/internal/service"

	"github.com/jackc/pgx/v5/pgtype"
)

// ─── Mock ────────────────────────────────────────────────────────────────────

type mockContentService struct {
	createFn      func(ctx context.Context, params service.CreateContentParams) (db.Content, error)
	getContentsFn func(ctx context.Context, params service.GetContentsParams) ([]db.GetContentsRow, error)
	getByIDFn     func(ctx context.Context, id string) (db.GetContentByIDRow, error)
	upsertNoteFn  func(ctx context.Context, params service.UpdateNoteParams) (db.Note, error)
	deleteFn      func(ctx context.Context, id string) error
	searchFn      func(query, filter string, limit, offset int64) (*service.SearchResponse, error)
}

func (m *mockContentService) Create(ctx context.Context, params service.CreateContentParams) (db.Content, error) {
	if m.createFn == nil {
		return db.Content{}, nil
	}
	return m.createFn(ctx, params)
}

func (m *mockContentService) GetContents(ctx context.Context, params service.GetContentsParams) ([]db.GetContentsRow, error) {
	if m.getContentsFn == nil {
		return nil, nil
	}
	return m.getContentsFn(ctx, params)
}

func (m *mockContentService) GetByID(ctx context.Context, id string) (db.GetContentByIDRow, error) {
	if m.getByIDFn == nil {
		return db.GetContentByIDRow{}, nil
	}
	return m.getByIDFn(ctx, id)
}

func (m *mockContentService) UpsertNote(ctx context.Context, params service.UpdateNoteParams) (db.Note, error) {
	if m.upsertNoteFn == nil {
		return db.Note{}, nil
	}
	return m.upsertNoteFn(ctx, params)
}

func (m *mockContentService) Delete(ctx context.Context, id string) error {
	if m.deleteFn == nil {
		return nil
	}
	return m.deleteFn(ctx, id)
}

func (m *mockContentService) Search(query, filter string, limit, offset int64) (*service.SearchResponse, error) {
	if m.searchFn == nil {
		return nil, nil
	}
	return m.searchFn(query, filter, limit, offset)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func setupRouter(svc service.ContentServiceInterface) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := api.NewContentHandler(svc)

	r.POST("/contents", h.CreateContent)
	r.GET("/contents", func(c *gin.Context) {
		limit := 20
		offset := 0
		h.GetContents(c, api.GetContentsParams{Limit: &limit, Offset: &offset})
	})
	r.GET("/contents/:id", func(c *gin.Context) {
		h.GetContentByID(c, c.Param("id"))
	})
	r.PATCH("/contents/:id", func(c *gin.Context) {
		h.UpdateNote(c, c.Param("id"))
	})
	r.DELETE("/contents/:id", func(c *gin.Context) {
		h.DeleteContent(c, c.Param("id"))
	})
	r.GET("/search", h.GetSearch)

	return r
}

func makeContentRow(id, url, status string, withMetadata bool) db.GetContentsRow {
	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
	row := db.GetContentsRow{
		ID:        id,
		Url:       url,
		Status:    status,
		Type:      pgtype.Text{String: "website", Valid: true},
		CreatedAt: now,
		Provider:  pgtype.Text{String: "medium.com", Valid: true},
	}
	if withMetadata {
		row.Title = pgtype.Text{String: "Título Teste", Valid: true}
		row.Description = pgtype.Text{String: "Descrição Teste", Valid: true}
	}
	return row
}

// ─── CreateContent ────────────────────────────────────────────────────────────

func TestCreateContent(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		body           map[string]any
		mockFn         func(ctx context.Context, params service.CreateContentParams) (db.Content, error)
		wantStatusCode int
		wantID         string
	}{
		{
			name: "sucesso — cria conteúdo e retorna 202",
			body: map[string]any{"url": "https://medium.com/artigo", "type": "website"},
			mockFn: func(_ context.Context, _ service.CreateContentParams) (db.Content, error) {
				return db.Content{ID: "01ABC", Url: "https://medium.com/artigo", CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}}, nil
			},
			wantStatusCode: http.StatusAccepted,
			wantID:         "01ABC",
		},
		{
			name:           "erro — body inválido (sem url)",
			body:           map[string]any{"type": "website"},
			mockFn:         nil,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "erro — falha no service",
			body: map[string]any{"url": "https://medium.com/artigo", "type": "website"},
			mockFn: func(_ context.Context, _ service.CreateContentParams) (db.Content, error) {
				return db.Content{}, errors.New("db connection error")
			},
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockContentService{createFn: tt.mockFn}
			r := setupRouter(svc)

			bodyBytes, _ := json.Marshal(tt.body)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/contents", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if tt.wantID != "" {
				var resp map[string]any
				json.Unmarshal(w.Body.Bytes(), &resp)
				assert.Equal(t, tt.wantID, resp["id"])
			}
		})
	}
}

// ─── GetContents ──────────────────────────────────────────────────────────────

func TestGetContents(t *testing.T) {
	tests := []struct {
		name           string
		mockFn         func(ctx context.Context, params service.GetContentsParams) ([]db.GetContentsRow, error)
		wantStatusCode int
		wantLen        int
	}{
		{
			name: "sucesso — retorna lista com metadados",
			mockFn: func(_ context.Context, _ service.GetContentsParams) ([]db.GetContentsRow, error) {
				return []db.GetContentsRow{
					makeContentRow("01A", "https://medium.com/a", "COMPLETED", true),
					makeContentRow("01B", "https://dev.to/b", "COMPLETED", true),
				}, nil
			},
			wantStatusCode: http.StatusOK,
			wantLen:        2,
		},
		{
			name: "sucesso — lista vazia",
			mockFn: func(_ context.Context, _ service.GetContentsParams) ([]db.GetContentsRow, error) {
				return []db.GetContentsRow{}, nil
			},
			wantStatusCode: http.StatusOK,
			wantLen:        0,
		},
		{
			name: "erro — falha no service",
			mockFn: func(_ context.Context, _ service.GetContentsParams) ([]db.GetContentsRow, error) {
				return nil, errors.New("db timeout")
			},
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockContentService{getContentsFn: tt.mockFn}
			r := setupRouter(svc)

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/contents", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if tt.wantStatusCode == http.StatusOK {
				var resp []map[string]any
				json.Unmarshal(w.Body.Bytes(), &resp)
				assert.Len(t, resp, tt.wantLen)
			}
		})
	}
}

// ─── GetContentByID ───────────────────────────────────────────────────────────

func TestGetContentByID(t *testing.T) {
	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}

	tests := []struct {
		name           string
		id             string
		mockFn         func(ctx context.Context, id string) (db.GetContentByIDRow, error)
		wantStatusCode int
		checkBody      func(t *testing.T, body []byte)
	}{
		{
			name: "sucesso — retorna conteúdo com metadados",
			id:   "01ABC",
			mockFn: func(_ context.Context, id string) (db.GetContentByIDRow, error) {
				return db.GetContentByIDRow{
					ID:          id,
					Url:         "https://medium.com/artigo",
					Status:      "COMPLETED",
					Type:        pgtype.Text{String: "website", Valid: true},
					CreatedAt:   now,
					UpdatedAt:   now,
					Title:       pgtype.Text{String: "Artigo Incrível", Valid: true},
					Description: pgtype.Text{String: "Uma descrição", Valid: true},
					Provider:    pgtype.Text{String: "medium.com", Valid: true},
				}, nil
			},
			wantStatusCode: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp map[string]any
				json.Unmarshal(body, &resp)
				assert.Equal(t, "01ABC", resp["id"])
				assert.NotNil(t, resp["metadata"])
				metadata := resp["metadata"].(map[string]any)
				assert.Equal(t, "Artigo Incrível", metadata["title"])
			},
		},
		{
			name: "sucesso — conteúdo sem metadados ainda (PENDING)",
			id:   "01DEF",
			mockFn: func(_ context.Context, id string) (db.GetContentByIDRow, error) {
				return db.GetContentByIDRow{
					ID:        id,
					Url:       "https://dev.to/artigo",
					Status:    "PENDING",
					Type:      pgtype.Text{String: "website", Valid: true},
					CreatedAt: now,
					UpdatedAt: now,
				}, nil
			},
			wantStatusCode: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp map[string]any
				json.Unmarshal(body, &resp)
				assert.Nil(t, resp["metadata"])
			},
		},
		{
			name: "erro — conteúdo não encontrado",
			id:   "nao-existe",
			mockFn: func(_ context.Context, _ string) (db.GetContentByIDRow, error) {
				return db.GetContentByIDRow{}, errors.New("not found")
			},
			wantStatusCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockContentService{getByIDFn: tt.mockFn}
			r := setupRouter(svc)

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/contents/"+tt.id, nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)
			if tt.checkBody != nil {
				tt.checkBody(t, w.Body.Bytes())
			}
		})
	}
}

// ─── UpdateNote ───────────────────────────────────────────────────────────────

func TestUpdateNote(t *testing.T) {
	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}

	tests := []struct {
		name           string
		id             string
		body           map[string]any
		mockFn         func(ctx context.Context, params service.UpdateNoteParams) (db.Note, error)
		wantStatusCode int
		checkBody      func(t *testing.T, body []byte)
	}{
		{
			name: "sucesso — cria nota",
			id:   "01ABC",
			body: map[string]any{"body": "Minha anotação sobre o artigo"},
			mockFn: func(_ context.Context, params service.UpdateNoteParams) (db.Note, error) {
				return db.Note{
					ID:        "note-01",
					Body:      params.Body,
					CreatedAt: now,
					UpdatedAt: now,
				}, nil
			},
			wantStatusCode: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp map[string]any
				json.Unmarshal(body, &resp)
				assert.Equal(t, "note-01", resp["id"])
				assert.Equal(t, "Minha anotação sobre o artigo", resp["body"])
			},
		},
		{
			name:           "erro — body inválido (sem campo body)",
			id:             "01ABC",
			body:           map[string]any{},
			mockFn:         nil,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "erro — falha no service",
			id:   "01ABC",
			body: map[string]any{"body": "Anotação"},
			mockFn: func(_ context.Context, _ service.UpdateNoteParams) (db.Note, error) {
				return db.Note{}, errors.New("db error")
			},
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockContentService{upsertNoteFn: tt.mockFn}
			r := setupRouter(svc)

			bodyBytes, _ := json.Marshal(tt.body)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPatch, "/contents/"+tt.id, bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)
			if tt.checkBody != nil {
				tt.checkBody(t, w.Body.Bytes())
			}
		})
	}
}

// ─── DeleteContent ────────────────────────────────────────────────────────────

func TestDeleteContent(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		mockFn         func(ctx context.Context, id string) error
		wantStatusCode int
	}{
		{
			name:           "sucesso — deleta e retorna 204",
			id:             "01ABC",
			mockFn:         func(_ context.Context, _ string) error { return nil },
			wantStatusCode: http.StatusNoContent,
		},
		{
			name:           "erro — falha no service",
			id:             "01ABC",
			mockFn:         func(_ context.Context, _ string) error { return errors.New("db error") },
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockContentService{deleteFn: tt.mockFn}
			r := setupRouter(svc)

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodDelete, "/contents/"+tt.id, nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)
		})
	}
}

// ─── GetSearch ────────────────────────────────────────────────────────────────

func TestGetSearch(t *testing.T) {
	tests := []struct {
		name           string
		queryString    string
		mockFn         func(query, filter string, limit, offset int64) (*service.SearchResponse, error)
		wantStatusCode int
		checkBody      func(t *testing.T, body []byte)
	}{
		{
			name:        "sucesso — retorna resultados com highlight",
			queryString: "?q=golang",
			mockFn: func(query, filter string, limit, offset int64) (*service.SearchResponse, error) {
				assert.Equal(t, "golang", query)
				assert.Equal(t, int64(20), limit)
				return &service.SearchResponse{
					Query:     "golang",
					TotalHits: 1,
					Hits: []search.SearchHit{
						{
							ID:       "01ABC",
							Title:    "Aprendendo Golang",
							Provider: "dev.to",
							Highlights: map[string]interface{}{
								"title": "Aprendendo <mark>Golang</mark>",
							},
						},
					},
				}, nil
			},
			wantStatusCode: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp service.SearchResponse
				json.Unmarshal(body, &resp)
				assert.Equal(t, "golang", resp.Query)
				assert.Equal(t, int64(1), resp.TotalHits)
				assert.Len(t, resp.Hits, 1)
				assert.Equal(t, "Aprendendo Golang", resp.Hits[0].Title)
			},
		},
		{
			name:        "sucesso — query vazia retorna lista vazia",
			queryString: "?q=",
			mockFn: func(query, filter string, limit, offset int64) (*service.SearchResponse, error) {
				return &service.SearchResponse{Query: "", Hits: []search.SearchHit{}}, nil
			},
			wantStatusCode: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp service.SearchResponse
				json.Unmarshal(body, &resp)
				assert.Empty(t, resp.Hits)
			},
		},
		{
			name:        "sucesso — respeita limit e offset customizados",
			queryString: "?q=go&limit=5&offset=10",
			mockFn: func(query, filter string, limit, offset int64) (*service.SearchResponse, error) {
				assert.Equal(t, int64(5), limit)
				assert.Equal(t, int64(10), offset)
				return &service.SearchResponse{Query: query, Hits: []search.SearchHit{}}, nil
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:        "sucesso — passa filtro para o service",
			queryString: "?q=video&filter=type+%3D+video",
			mockFn: func(query, filter string, limit, offset int64) (*service.SearchResponse, error) {
				assert.Equal(t, "type = video", filter)
				return &service.SearchResponse{Query: query, Hits: []search.SearchHit{}}, nil
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:        "erro — falha no service de busca",
			queryString: "?q=golang",
			mockFn: func(query, filter string, limit, offset int64) (*service.SearchResponse, error) {
				return nil, errors.New("meilisearch unavailable")
			},
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockContentService{searchFn: tt.mockFn}
			r := setupRouter(svc)

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/search"+tt.queryString, nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)
			if tt.checkBody != nil {
				tt.checkBody(t, w.Body.Bytes())
			}
		})
	}
}
