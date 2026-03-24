package worker

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

const enrichmentQueue = "enrichment_queue" // Deve ser o mesmo nome da fila no serviço

// EnrichmentPayload representa a estrutura do payload esperado na fila de enriquecimento.
type EnrichmentPayload struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// StartWorker inicia o consumidor da fila de enriquecimento.
func StartWorker(redisClient *redis.Client) {
	log.Println("Worker iniciado, escutando a fila:", enrichmentQueue)
	ctx := context.Background()

	for {
		// BLPOP bloqueia até que um item esteja disponível na fila
		// O 0 significa timeout infinito
		result, err := redisClient.BLPop(ctx, 0, enrichmentQueue).Result()
		if err != nil {
			log.Printf("Erro ao ler da fila Redis '%s': %v", enrichmentQueue, err)
			time.Sleep(5 * time.Second) // Espera antes de tentar novamente
			continue
		}

		// result[0] é o nome da fila, result[1] é o valor
		payloadStr := result[1]
		var payload EnrichmentPayload
		if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
			log.Printf("Erro ao decodificar payload da fila: %v. Payload: %s", err, payloadStr)
			continue // Pula para o próximo item
		}

		log.Printf("Processando conteúdo ID: %s para a URL: %s", payload.ID, payload.URL)

		// Simula um processamento demorado
		time.Sleep(2 * time.Second)

		log.Printf("Processamento do conteúdo ID: %s concluído.", payload.ID)
	}
}
