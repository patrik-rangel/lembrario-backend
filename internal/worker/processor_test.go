package worker_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"lembrario-backend/internal/db"
	"lembrario-backend/internal/worker"
)

// ─── Helpers & Mocks ─────────────────────────────────────────────────────────

type MockQueries struct {
	mock.Mock
}

type MockSearch struct {
	mock.Mock
}

// Implementando os métodos necessários para o Worker
func (m *MockQueries) UpsertMetadata(ctx context.Context, arg db.UpsertMetadataParams) (db.Metadata, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.Metadata), args.Error(1)
}

func (m *MockQueries) UpdateContentStatus(ctx context.Context, arg db.UpdateContentStatusParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

// Métodos stub para satisfazer a interface Querier (sqlc)
func (m *MockQueries) CreateContent(ctx context.Context, arg db.CreateContentParams) (db.Content, error) { return db.Content{}, nil }
func (m *MockQueries) DeleteContent(ctx context.Context, id string) error { return nil }
func (m *MockQueries) GetContentByID(ctx context.Context, id string) (db.GetContentByIDRow, error) { return db.GetContentByIDRow{}, nil }
func (m *MockQueries) GetContents(ctx context.Context, arg db.GetContentsParams) ([]db.GetContentsRow, error) { return nil, nil }
func (m *MockQueries) UpsertNote(ctx context.Context, arg db.UpsertNoteParams) (db.Note, error) { return db.Note{}, nil }

// ─── Test Table: Processamento ───────────────────────────────────────────────

func TestProcessContent(t *testing.T) {
	// Salva as funções originais e restaura ao final do teste
	oldScrape := worker.ScrapeURLFunc
	oldThumb := worker.DownloadThumbnailFunc
	defer func() {
		worker.ScrapeURLFunc = oldScrape
		worker.DownloadThumbnailFunc = oldThumb
	}()

	tests := []struct {
		name          string
		payload       worker.EnrichmentPayload
		scrapeResult  *worker.ScrapedData
		scrapeErr     error
		expectedStatus string
	}{
		{
			name: "Sucesso no processamento completo",
			payload: worker.EnrichmentPayload{ID: "01ABC", URL: "https://youtube.com/v123", Type: "video"},
			scrapeResult: &worker.ScrapedData{
				Title:    "Vídeo de Go",
				Provider: "youtube.com",
			},
			scrapeErr:     nil,
			expectedStatus: "COMPLETED",
		},
		{
			name: "Falha no scraping deve marcar como ERROR",
			payload: worker.EnrichmentPayload{ID: "01ERR", URL: "https://link-quebrado.com"},
			scrapeErr:     fmt.Errorf("timeout"),
			expectedStatus: "ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			
			// Setup Redis & Mocks
			mr := miniredis.RunT(t)
			rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
			mockQueries := new(MockQueries)
			
			// Mock do Scraper
			worker.ScrapeURLFunc = func(ctx context.Context, url string) (*worker.ScrapedData, error) {
				return tt.scrapeResult, tt.scrapeErr
			}
			worker.DownloadThumbnailFunc = func(ctx context.Context, id, url string) (string, error) {
				return "/path/to/thumb.jpg", nil
			}

			// Expectativas do Banco
			if tt.scrapeErr == nil {
				mockQueries.On("UpsertMetadata", mock.Anything, mock.Anything).Return(db.Metadata{}, nil)
			}
			mockQueries.On("UpdateContentStatus", mock.Anything, db.UpdateContentStatusParams{
				ID:     tt.payload.ID,
				Status: tt.expectedStatus,
			}).Return(nil)

			// Escuta o canal de notificações do Redis para validar o Pub/Sub
			pubsub := rdb.Subscribe(context.Background(), "content_updates")
			ch := pubsub.Channel()

			// Execução (usamos o processContent diretamente para evitar o loop do StartWorker)
			// Nota: Passamos nil para o SearchClient por enquanto ou um mock se preferir
			worker.ProcessContent(context.Background(), rdb, mockQueries, tt.payload, nil)

			// Validações
			mockQueries.AssertExpectations(t)

			// Valida se a notificação Pub/Sub foi enviada
			select {
			case msg := <-ch:
				var event worker.ContentUpdateEvent
				json.Unmarshal([]byte(msg.Payload), &event)
				assert.Equal(t, tt.payload.ID, event.ID)
				assert.Equal(t, tt.expectedStatus, event.Status)
			case <-time.After(500 * time.Millisecond):
				t.Fatal("Timeout: Notificação Pub/Sub não recebida")
			}
		})
	}
}