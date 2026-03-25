package service

import (
	"context"
	"encoding/json"
	"log"

	"lembrario-backend/internal/db" // Assumindo que o sqlc gerou tipos e queries aqui
	"lembrario-backend/pkg/id"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
)

const enrichmentQueue = "enrichment_queue"

type ContentService struct {
	queries *db.Queries
	redis   *redis.Client
}

func NewContentService(queries *db.Queries, redisClient *redis.Client) *ContentService {
	return &ContentService{
		queries: queries,
		redis:   redisClient,
	}
}

// CreateContentParams define os parâmetros para a criação de um novo conteúdo.
type CreateContentParams struct {
	URL  string
	Type string
}

// UpdateNoteParams define os parâmetros para atualização de uma nota.
type UpdateNoteParams struct {
	ContentID string
	Body      string
}

// GetContentsParams define os parâmetros para listagem de conteúdos.
type GetContentsParams struct {
	Limit  int32
	Offset int32
}

// Create cria um novo conteúdo, salva no banco de dados e o enfileira para enriquecimento.
func (s *ContentService) Create(ctx context.Context, params CreateContentParams) (db.Content, error) {
	contentID := id.New()

	createArgs := db.CreateContentParams{
		ID:   contentID,
		Url:  params.URL,
		Type: pgtype.Text{String: params.Type, Valid: true},
	}

	content, err := s.queries.CreateContent(ctx, createArgs)
	if err != nil {
		return db.Content{}, err
	}

	// Envia para a fila Redis
	payload := map[string]string{
		"id":  content.ID,
		"url": content.Url,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Erro ao serializar payload para Redis: %v", err)
		// Não retornamos erro aqui, pois o conteúdo já foi salvo no DB.
		// Apenas logamos e continuamos. O processamento da fila pode ter retentativas.
	} else {
		err = s.redis.LPush(ctx, enrichmentQueue, payloadBytes).Err()
		if err != nil {
			log.Printf("Erro ao enviar para a fila Redis '%s': %v", enrichmentQueue, err)
			// Idem, logar mas não falhar a criação do conteúdo.
		} else {
			log.Printf("Conteúdo ID %s enfileirado para enriquecimento.", content.ID)
		}
	}

	return content, nil
}

// GetByID busca um conteúdo por ID, incluindo metadados e nota associada.
func (s *ContentService) GetByID(ctx context.Context, contentID string) (db.GetContentByIDRow, error) {
	return s.queries.GetContentByID(ctx, contentID)
}

// GetContents lista conteúdos com paginação, incluindo metadados e notas.
func (s *ContentService) GetContents(ctx context.Context, params GetContentsParams) ([]db.GetContentsRow, error) {
	args := db.GetContentsParams{
		Limit:  params.Limit,
		Offset: params.Offset,
	}
	return s.queries.GetContents(ctx, args)
}

// UpsertNote cria ou atualiza uma nota associada a um conteúdo.
func (s *ContentService) UpsertNote(ctx context.Context, params UpdateNoteParams) (db.Note, error) {
	noteID := id.New()

	args := db.UpsertNoteParams{
		ID:        noteID,
		ContentID: params.ContentID,
		Body:      params.Body,
	}

	return s.queries.UpsertNote(ctx, args)
}

// Delete remove um conteúdo e todos os dados associados (cascade).
func (s *ContentService) Delete(ctx context.Context, contentID string) error {
	// O DELETE CASCADE no banco cuidará de remover metadados e notas automaticamente
	return s.queries.DeleteContent(ctx, contentID)
}
