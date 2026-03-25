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

const enrichmentQueue = "enrichment_queue"

// EnrichmentPayload representa a estrutura do payload esperado na fila de enriquecimento.
type EnrichmentPayload struct {
	ID  string `json:"id"`
	URL string `json:"url"`
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

			// Por enquanto, apenas simula processamento
			log.Printf("🔄 Processando conteúdo ID: %s...", payload.ID)
			time.Sleep(2 * time.Second)
			log.Printf("✅ Processamento do conteúdo ID: %s concluído (simulado)", payload.ID)
		}
	}
}


func processContent(ctx context.Context, redisClient *redis.Client, queries *db.Queries, payload EnrichmentPayload) error {                                  
    log.Printf("🔄 Iniciando processamento do conteúdo ID: %s", payload.ID)                                                                                  
                                                                                                                                                             
    // 1. Fazer scraping da URL                                                                                                                              
    scrapedData, err := ScrapeURL(ctx, payload.URL)                                                                                                          
    if err != nil {                                                                                                                                          
        return fmt.Errorf("erro no scraping: %w", err)                                                                                                       
    }                                                                                                                                                        
                                                                                                                                                             
    log.Printf("📄 Dados extraídos - Título: %s, Provider: %s", scrapedData.Title, scrapedData.Provider)                                                     
                                                                                                                                                             
    // 2. Salvar metadados no banco                                                                                                                          
    err = saveMetadata(ctx, queries, payload.ID, scrapedData)                                                                                                
    if err != nil {                                                                                                                                          
        return fmt.Errorf("erro ao salvar metadados: %w", err)                                                                                               
    }                                                                                                                                                        
                                                                                                                                                             
    // 3. Atualizar status para COMPLETED                                                                                                                    
    err = updateContentStatus(ctx, queries, payload.ID, "COMPLETED")                                                                                         
    if err != nil {                                                                                                                                          
        return fmt.Errorf("erro ao atualizar status: %w", err)                                                                                               
    }                                                                                                                                                        
                                                                                                                                                             
    // 4. Notificar via Pub/Sub                                                                                                                              
    err = notifyContentUpdate(ctx, redisClient, payload.ID, "COMPLETED")                                                                                     
    if err != nil {                                                                                                                                          
        log.Printf("⚠️ Erro ao notificar atualização via Pub/Sub: %v", err)                                                                                  
        // Não retornamos erro aqui pois o processamento foi bem-sucedido                                                                                    
    }                                                                                                                                                        
                                                                                                                                                             
    log.Printf("✅ Processamento do conteúdo ID: %s concluído com sucesso", payload.ID)                                                                      
    return nil                                                                                                                                               
}                                                                                                                                                            
                                                                                                                                                             
// saveMetadata salva os metadados extraídos no banco de dados                                                                                               
func saveMetadata(ctx context.Context, queries *db.Queries, contentID string, data *ScrapedData) error {                                                     
    params := db.UpsertMetadataParams{                                                                                                                       
        ContentID:     contentID,                                                                                                                            
        Title:         pgtype.Text{String: data.Title, Valid: data.Title != ""},                                                                             
        Description:   pgtype.Text{String: data.Description, Valid: data.Description != ""},                                                                 
        ThumbnailPath: pgtype.Text{Valid: false}, // Por enquanto não extraímos thumbnail                                                                    
        Transcript:    pgtype.Text{Valid: false}, // Por enquanto não extraímos transcript                                                                   
        Provider:      pgtype.Text{String: data.Provider, Valid: data.Provider != ""},                                                                       
        ReadingTime:   pgtype.Int4{Valid: false}, // Por enquanto não calculamos tempo de leitura                                                            
        RawData:       []byte{},                  // Por enquanto não salvamos dados brutos                                                                  
    }                                                                                                                                                        
                                                                                                                                                             
    _, err := queries.UpsertMetadata(ctx, params)                                                                                                            
    return err                                                                                                                                               
}                                                                                                                                                            
                                                                                                                                                             
// updateContentStatus atualiza o status de um conteúdo                                                                                                      
func updateContentStatus(ctx context.Context, queries *db.Queries, contentID, status string) error {                                                         
    params := db.UpdateContentStatusParams{                                                                                                                  
        ID:     contentID,                                                                                                                                   
        Status: status,                                                                                                                                      
    }                                                                                                                                                        
    return queries.UpdateContentStatus(ctx, params)                                                                                                          
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
        return err                                                                                                                                           
    }                                                                                                                                                        
                                                                                                                                                             
    return redisClient.Publish(ctx, contentUpdatesChannel, eventBytes).Err()                                                                                 
}                                                                                                                                                            
           