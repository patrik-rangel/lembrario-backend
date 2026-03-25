package worker

import (
	"context"
	"encoding/json"
	"log"
	"time"

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
