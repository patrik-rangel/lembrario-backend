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
)

const (
	enrichmentQueue       = "enrichment_queue"
	contentUpdatesChannel = "content_updates"
)

// EnrichmentPayload representa a estrutura do payload esperado na fila de enriquecimento.
type EnrichmentPayload struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// ContentUpdateEvent representa o evento de atualização de conteúdo
type ContentUpdateEvent struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Type   string `json:"type"`
}

// ScrapedData representa os dados extraídos de uma URL
type ScrapedData struct {
	Title       string
	Description string
	Provider    string
}

// StartWorker inicia o consumidor da fila de enriquecimento com graceful shutdown.
func StartWorker(ctx context.Context, redisClient *redis.Client, queries *db.Queries) {
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
			processContent(ctx, redisClient, queries, payload)
		}
	}
}

// processContent processa um conteúdo individual com tratamento completo de erro
func processContent(ctx context.Context, redisClient *redis.Client, queries *db.Queries, payload EnrichmentPayload) {
	log.Printf("🔄 Iniciando processamento do conteúdo ID: %s", payload.ID)

	// 1. Fazer scraping da URL
	scrapedData, err := ScrapeURL(ctx, payload.URL)
	if err != nil {
		log.Printf("❌ Erro no scraping para ID %s: %v", payload.ID, err)
		// Marcar como ERROR e notificar
		handleError(ctx, redisClient, queries, payload.ID, "Erro no scraping")
		return
	}

	log.Printf("📄 Dados extraídos - Título: %s, Provider: %s", scrapedData.Title, scrapedData.Provider)

	// 2. Salvar metadados no banco
	err = saveMetadata(ctx, queries, payload.ID, scrapedData)
	if err != nil {
		log.Printf("❌ Erro ao salvar metadados para ID %s: %v", payload.ID, err)
		// Marcar como ERROR e notificar
		handleError(ctx, redisClient, queries, payload.ID, "Erro ao salvar metadados")
		return
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
func handleError(ctx context.Context, redisClient *redis.Client, queries *db.Queries, contentID, reason string) {
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
func saveMetadata(ctx context.Context, queries *db.Queries, contentID string, data *ScrapedData) error {
	params := db.UpsertMetadataParams{
		ContentID:     contentID,
		Title:         pgtype.Text{String: data.Title, Valid: data.Title != ""},
		Description:   pgtype.Text{String: data.Description, Valid: data.Description != ""},
		ThumbnailPath: pgtype.Text{Valid: false},
		Transcript:    pgtype.Text{Valid: false},
		Provider:      pgtype.Text{String: data.Provider, Valid: data.Provider != ""},
		ReadingTime:   pgtype.Int4{Valid: false},
		RawData:       []byte{},
	}

	_, err := queries.UpsertMetadata(ctx, params)
	if err != nil {
		return fmt.Errorf("falha ao salvar metadados: %w", err)
	}

	log.Printf("💾 Metadados salvos para conteúdo ID: %s", contentID)
	return nil
}

// updateContentStatus atualiza o status de um conteúdo
func updateContentStatus(ctx context.Context, queries *db.Queries, contentID, status string) error {
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
