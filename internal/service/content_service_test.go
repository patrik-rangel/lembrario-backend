package service_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"lembrario-backend/internal/db"
	"lembrario-backend/internal/search"
	"lembrario-backend/internal/service"
)

// MockDB define os comportamentos esperados do banco
type MockDB struct {
	mock.Mock
}

// MockQueries implementa a interface db.Querier completa
type MockQueries struct {
	mock.Mock
}

func (m *MockQueries) CreateContent(ctx context.Context, arg db.CreateContentParams) (db.Content, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.Content), args.Error(1)
}

func (m *MockQueries) GetContentByID(ctx context.Context, id string) (db.GetContentByIDRow, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(db.GetContentByIDRow), args.Error(1)
}

func (m *MockQueries) UpsertNote(ctx context.Context, arg db.UpsertNoteParams) (db.Note, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.Note), args.Error(1)
}

func (m *MockQueries) DeleteContent(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockQueries) GetContents(ctx context.Context, arg db.GetContentsParams) ([]db.GetContentsRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]db.GetContentsRow), args.Error(1)
}

func (m *MockQueries) UpsertMetadata(ctx context.Context, arg db.UpsertMetadataParams) (db.Metadata, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.Metadata), args.Error(1)
}

func (m *MockQueries) UpdateContentStatus(ctx context.Context, arg db.UpdateContentStatusParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

// MockSearchClient simula o cliente Meilisearch
type MockSearchClient struct {
	mock.Mock
}

func (m *MockSearchClient) Search(options search.SearchOptions) (*search.SearchResult, error) {
	args := m.Called(options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*search.SearchResult), args.Error(1)
}

// (Aqui você implementaria os métodos do sqlc conforme a necessidade do teste)
// Exemplo simplificado para o Create:
func (m *MockDB) CreateContent(ctx context.Context, arg db.CreateContentParams) (db.Content, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.Content), args.Error(1)
}

// MockSearch define o comportamento do Meilisearch
type MockSearch struct {
	mock.Mock
}

func (m *MockSearch) Search(options any) (any, error) { // simplificado
	args := m.Called(options)
	return args.Get(0), args.Error(1)
}

func TestContentService_Create(t *testing.T) {
	tests := []struct {
		name       string
		params     service.CreateContentParams
		dbResult   db.Content
		dbError    error
		wantErr    bool
		checkRedis bool
	}{
		{
			name: "Sucesso ao criar e enfileirar",
			params: service.CreateContentParams{
				URL:  "https://google.com",
				Type: "website",
			},
			dbResult:   db.Content{ID: "01ABC", Url: "https://google.com"},
			dbError:    nil,
			wantErr:    false,
			checkRedis: true,
		},
		{
			name: "Erro no banco não deve enfileirar no Redis",
			params: service.CreateContentParams{
				URL: "https://erro.com",
			},
			dbResult:   db.Content{},
			dbError:    fmt.Errorf("falha no banco"),
			wantErr:    true,
			checkRedis: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mr := miniredis.RunT(t)
			rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
			mockQueries := new(MockQueries)

			// Injetamos o mock no service (precisará de cast ou interface no Service)
			s := service.NewContentService(mockQueries, rdb, nil)

			// Setup das expectativas do mock
			if tt.dbError != nil || tt.dbResult.ID != "" {
				mockQueries.On("CreateContent", mock.Anything, mock.Anything).Return(tt.dbResult, tt.dbError)
			}

			content, err := s.Create(context.Background(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.dbResult.ID, content.ID)
			}

			if tt.checkRedis {
				// Verifica se o item caiu na fila "enrichment_queue"
				len := rdb.LLen(context.Background(), "enrichment_queue").Val()
				assert.Equal(t, int64(1), len)
			}
		})
	}
}

// ─── Test Table: GetByID ─────────────────────────────────────────────────────

func TestContentService_GetByID(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		dbResult db.GetContentByIDRow
		dbError  error
		wantErr  bool
	}{
		{
			name: "Sucesso ao buscar conteúdo",
			id:   "01ABC",
			dbResult: db.GetContentByIDRow{
				ID:  "01ABC",
				Url: "https://teste.com",
			},
			wantErr: false,
		},
		{
			name:    "Erro quando ID não existe",
			id:      "XPT00",
			dbError: fmt.Errorf("not found"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueries := new(MockQueries)
			s := service.NewContentService(mockQueries, nil, nil)

			mockQueries.On("GetContentByID", mock.Anything, tt.id).Return(tt.dbResult, tt.dbError)

			res, err := s.GetByID(context.Background(), tt.id)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.id, res.ID)
			}
		})
	}
}

// ─── Test Table: UpsertNote ──────────────────────────────────────────────────

func TestContentService_UpsertNote(t *testing.T) {
	tests := []struct {
		name    string
		params  service.UpdateNoteParams
		dbError error
		wantErr bool
	}{
		{
			name: "Sucesso ao salvar nota",
			params: service.UpdateNoteParams{
				ContentID: "01ABC",
				Body:      "# Nova Nota",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueries := new(MockQueries)
			s := service.NewContentService(mockQueries, nil, nil)

			mockQueries.On("UpsertNote", mock.Anything, mock.MatchedBy(func(p db.UpsertNoteParams) bool {
				return p.ContentID == tt.params.ContentID && p.Body == tt.params.Body
			})).Return(db.Note{}, tt.dbError)

			_, err := s.UpsertNote(context.Background(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ─── Test Table: Delete ──────────────────────────────────────────────────────

func TestContentService_Delete(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		dbError error
		wantErr bool
	}{
		{
			name:    "Sucesso ao deletar",
			id:      "01ABC",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueries := new(MockQueries)
			s := service.NewContentService(mockQueries, nil, nil)

			mockQueries.On("DeleteContent", mock.Anything, tt.id).Return(tt.dbError)

			err := s.Delete(context.Background(), tt.id)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
