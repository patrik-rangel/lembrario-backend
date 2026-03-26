package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
	"lembrario-backend/internal/db"
	"lembrario-backend/internal/search"
)

var (
	ScrapeURLFunc         = ScrapeURL
	DownloadThumbnailFunc = DownloadThumbnail
)

const (
	enrichmentQueue       = "enrichment_queue"
	contentUpdatesChannel = "content_updates"
)

// EnrichmentPayload representa a estrutura do payload esperado na fila de enriquecimento.
type EnrichmentPayload struct {
	ID   string `json:"id"`
	URL  string `json:"url"`
	Type string `json:"type"`
}

// ContentUpdateEvent representa o evento de atualização de conteúdo
type ContentUpdateEvent struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Type   string `json:"type"`
}

// ScrapedData representa os dados extraídos de uma URL
type ScrapedData struct {
	Title        string
	Description  string
	Provider     string
	ThumbnailURL string // novo
	AuthorName   string // novo — canal do YouTube, autor do artigo etc.
	Duration     string // novo — para vídeos/podcasts
}

// StartWorker inicia o consumidor da fila de enriquecimento com graceful shutdown.
func StartWorker(ctx context.Context, redisClient *redis.Client, queries db.Querier, searchClient *search.Client) {
	log.Printf("Worker iniciado, escutando a fila: %s", enrichmentQueue)

	for {
		select {
		case <-ctx.Done():
			log.Println("Contexto cancelado, finalizando worker...")
			return
		default:
			// BRPOP bloqueia até que um item esteja disponível na fila
			// Timeout de 5 segundos para permitir verificação do contexto
			result, err := redisClient.BRPop(ctx, 5*time.Second, enrichmentQueue).Result()
			if err != nil {
				// Se for timeout, continua o loop para verificar o contexto
				if err.Error() == "redis: nil" {
					continue
				}
				log.Printf("Erro ao ler da fila Redis '%s': %v", enrichmentQueue, err)
				time.Sleep(5 * time.Second)
				continue
			}

			// result[0] é o nome da fila, result[1] é o valor
			payloadStr := result[1]
			var payload EnrichmentPayload
			if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
				log.Printf("Erro ao decodificar payload da fila: %v. Payload: %s", err, payloadStr)
				continue
			}

			log.Printf("📥 Mensagem recebida - ID: %s, URL: %s", payload.ID, payload.URL)

			// Processar o conteúdo (não retornamos erro aqui pois já tratamos internamente)
			ProcessContent(ctx, redisClient, queries, payload, searchClient)
		}
	}
}

// processContent processa um conteúdo individual com tratamento completo de erro
func ProcessContent(ctx context.Context, redisClient *redis.Client, queries db.Querier, payload EnrichmentPayload, searchClient *search.Client) {
	log.Printf("🔄 Iniciando processamento do conteúdo ID: %s", payload.ID)

	// 1. Fazer scraping da URL
	scrapedData, err := ScrapeURLFunc(ctx, payload.URL)
	if err != nil {
		log.Printf("❌ Erro no scraping para ID %s: %v", payload.ID, err)
		// Marcar como ERROR e notificar
		handleError(ctx, redisClient, queries, payload.ID, "Erro no scraping")
		return
	}

	log.Printf("📄 Dados extraídos - Título: %s, Provider: %s", scrapedData.Title, scrapedData.Provider)

	thumbnailPath, err := DownloadThumbnailFunc(ctx, payload.ID, scrapedData.ThumbnailURL)
	if err != nil {
		log.Printf("⚠️ Erro ao baixar thumbnail para ID %s: %v (continuando)", payload.ID, err)
		thumbnailPath = ""
	}
	scrapedData.ThumbnailURL = thumbnailPath

	// 2. Salvar metadados no banco
	err = saveMetadata(ctx, queries, payload.ID, scrapedData)
	if err != nil {
		log.Printf("❌ Erro ao salvar metadados para ID %s: %v", payload.ID, err)
		// Marcar como ERROR e notificar
		handleError(ctx, redisClient, queries, payload.ID, "Erro ao salvar metadados")
		return
	}

	if err := indexContent(searchClient, payload, scrapedData); err != nil {
		// Não é fatal — o conteúdo já foi salvo no banco
		log.Printf("⚠️ Erro ao indexar conteúdo %s no Meilisearch: %v", payload.ID, err)
	} else {
		log.Printf("🔍 Conteúdo %s indexado no Meilisearch", payload.ID)
	}

	// 3. Atualizar status para COMPLETED
	err = updateContentStatus(ctx, queries, payload.ID, "COMPLETED")
	if err != nil {
		log.Printf("❌ Erro ao atualizar status para ID %s: %v", payload.ID, err)
		// Marcar como ERROR e notificar
		handleError(ctx, redisClient, queries, payload.ID, "Erro ao atualizar status")
		return
	}

	// 4. Notificar via Pub/Sub
	err = notifyContentUpdate(ctx, redisClient, payload.ID, "COMPLETED")
	if err != nil {
		log.Printf("⚠️ Erro ao notificar atualização via Pub/Sub para ID %s: %v", payload.ID, err)
		// Não tratamos como erro fatal aqui pois o processamento foi bem-sucedido
	}

	log.Printf("✅ Processamento do conteúdo ID: %s concluído com sucesso", payload.ID)
}

// handleError trata erros marcando o conteúdo como ERROR e notificando
func handleError(ctx context.Context, redisClient *redis.Client, queries db.Querier, contentID, reason string) {
	// Tentar marcar como ERROR no banco
	if err := updateContentStatus(ctx, queries, contentID, "ERROR"); err != nil {
		log.Printf("❌ Falha ao marcar conteúdo %s como ERROR: %v", contentID, err)
	}

	// Tentar notificar via Pub/Sub
	if err := notifyContentUpdate(ctx, redisClient, contentID, "ERROR"); err != nil {
		log.Printf("❌ Falha ao notificar erro via Pub/Sub para conteúdo %s: %v", contentID, err)
	}

	log.Printf("💀 Conteúdo %s marcado como ERROR: %s", contentID, reason)
}

// saveMetadata salva os metadados extraídos no banco de dados
func saveMetadata(ctx context.Context, queries db.Querier, contentID string, data *ScrapedData) error {
	rawJson, _ := json.Marshal(map[string]string{
		"source":        "scraper_v1",
		"fetched_at":    time.Now().String(),
		"thumbnail_url": data.ThumbnailURL, // guarda a URL original no raw_data
	})
	params := db.UpsertMetadataParams{
		ContentID:     contentID,
		Title:         pgtype.Text{String: data.Title, Valid: data.Title != ""},
		Description:   pgtype.Text{String: data.Description, Valid: data.Description != ""},
		ThumbnailPath: pgtype.Text{String: data.ThumbnailURL, Valid: data.ThumbnailURL != ""},
		Transcript:    pgtype.Text{Valid: false},
		Provider:      pgtype.Text{String: data.Provider, Valid: data.Provider != ""},
		ReadingTime:   pgtype.Int4{Valid: false},
		RawData:       rawJson,
	}

	_, err := queries.UpsertMetadata(ctx, params)
	if err != nil {
		return fmt.Errorf("falha ao salvar metadados: %w", err)
	}

	log.Printf("💾 Metadados salvos para conteúdo ID: %s", contentID)
	return nil
}

// updateContentStatus atualiza o status de um conteúdo
func updateContentStatus(ctx context.Context, queries db.Querier, contentID, status string) error {
	params := db.UpdateContentStatusParams{
		ID:     contentID,
		Status: status,
	}

	err := queries.UpdateContentStatus(ctx, params)
	if err != nil {
		return fmt.Errorf("falha ao atualizar status: %w", err)
	}

	log.Printf("🔄 Status atualizado para '%s' - conteúdo ID: %s", status, contentID)
	return nil
}

// notifyContentUpdate publica uma notificação de atualização de conteúdo
func notifyContentUpdate(ctx context.Context, redisClient *redis.Client, contentID, status string) error {
	event := ContentUpdateEvent{
		ID:     contentID,
		Status: status,
		Type:   "content_update",
	}

	eventBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("falha ao serializar evento: %w", err)
	}

	err = redisClient.Publish(ctx, contentUpdatesChannel, eventBytes).Err()
	if err != nil {
		return fmt.Errorf("falha ao publicar no Redis: %w", err)
	}

	log.Printf("📢 Notificação enviada - ID: %s, Status: %s", contentID, status)
	return nil
}

func indexContent(searchClient *search.Client, payload EnrichmentPayload, data *ScrapedData) error {
	if searchClient == nil {
		return nil
	}

	return searchClient.IndexContent(search.ContentDocument{
		ID:          payload.ID,
		Title:       data.Title,
		Description: data.Description,
		URL:         payload.URL,
		Type:        payload.Type,
		Provider:    data.Provider,
		AuthorName:  data.AuthorName,
		CreatedAt:   time.Now(),
	})
}
