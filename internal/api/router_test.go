package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"lembrario-backend/internal/api"
	"lembrario-backend/internal/db"
	"lembrario-backend/internal/search"
	"lembrario-backend/internal/service"
)

var (
	testToken   string
	authService *api.AuthService
)

// ─── Helpers ──────────────────────────────────────────────────────────────────

func setupTestRouter(t *testing.T, svc service.ContentServiceInterface) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	// Configura credenciais de teste
	os.Setenv("JWT_SECRET", "secret-muito-secreto-para-testes-123")
	os.Setenv("ADMIN_USER", "admin")
	os.Setenv("ADMIN_PASSWORD", "pass")

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	// Gera o token que os testes vão usar
	testService := api.NewAuthService()
	token, _, _ := testService.GenerateToken("admin")
	testToken = token

	contentHandler := api.NewContentHandler(svc)
	sseHandler := api.NewSSEHandler(rdb)

	return api.SetupRouter(contentHandler, sseHandler)
}

// ─── Health ───────────────────────────────────────────────────────────────────

func TestHealthEndpoint(t *testing.T) {
	r := setupTestRouter(t, &mockContentService{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	parseJSON(t, w.Body.Bytes(), &body)
	assert.Equal(t, "UP", body["status"])
}

// ─── Roteamento ───────────────────────────────────────────────────────────────

func TestRouteRegistration(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		wantStatusCode int
	}{
		{
			name:           "POST /contents existe",
			method:         http.MethodPost,
			path:           "/contents",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "GET /contents existe",
			method:         http.MethodGet,
			path:           "/contents",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "GET /contents/:id existe",
			method:         http.MethodGet,
			path:           "/contents/01ABC",
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:           "PATCH /contents/:id existe",
			method:         http.MethodPatch,
			path:           "/contents/01ABC",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "DELETE /contents/:id existe",
			method:         http.MethodDelete,
			path:           "/contents/01ABC",
			wantStatusCode: http.StatusNoContent,
		},
		{
			name:           "GET /search existe",
			method:         http.MethodGet,
			path:           "/search?q=teste",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "rota inexistente retorna 404",
			method:         http.MethodGet,
			path:           "/nao-existe",
			wantStatusCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockContentService{
				getContentsFn: func(_ context.Context, _ service.GetContentsParams) ([]db.GetContentsRow, error) {
					return []db.GetContentsRow{}, nil
				},
				getByIDFn: func(_ context.Context, _ string) (db.GetContentByIDRow, error) {
					return db.GetContentByIDRow{}, errors.New("not found")
				},
				searchFn: func(_, _ string, _, _ int64) (*service.SearchResponse, error) {
					return &service.SearchResponse{Hits: []search.SearchHit{}}, nil
				},
			}

			r := setupTestRouter(t, svc)

			w := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, nil)

			req.Header.Set("Authorization", "Bearer "+testToken)

			r.ServeHTTP(w, req)
			assert.Equal(t, tt.wantStatusCode, w.Code, "rota: %s %s", tt.method, tt.path)
		})
	}
}

// ─── SSE ──────────────────────────────────────────────────────────────────────

func TestEventsEndpointHeaders(t *testing.T) {
	r := setupTestRouter(t, &mockContentService{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancela imediatamente para não bloquear

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+testToken)
	r.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", w.Header().Get("Connection"))
}

// ─── Util ─────────────────────────────────────────────────────────────────────

func parseJSON(t *testing.T, data []byte, v any) {
	t.Helper()
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("falha ao fazer parse do JSON: %v\nbody: %s", err, string(data))
	}
}
