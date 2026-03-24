package service

import (
	"context"
	"encoding/json"
	"log"

	"github.com/go-dev-api-doc/internal/db" // Assumindo que o sqlc gerou tipos e queries aqui
	"github.com/go-dev-api-doc/pkg/id"
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

// Create cria um novo conteúdo, salva no banco de dados e o enfileira para enriquecimento.
func (s *ContentService) Create(ctx context.Context, params CreateContentParams) (db.Content, error) {
	contentID := id.New()

	createArgs := db.CreateContentParams{
		ID:   contentID,
		Url:  params.URL,
		Type: params.Type,
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
