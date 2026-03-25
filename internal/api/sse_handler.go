package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type SSEHandler struct {
	redis *redis.Client
}

func NewSSEHandler(redisClient *redis.Client) *SSEHandler {
	return &SSEHandler{
		redis: redisClient,
	}
}

// ContentUpdateEvent representa um evento de atualização de conteúdo
type ContentUpdateEvent struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Type   string `json:"type"`
}

// GetEvents implementa o endpoint SSE para atualizações em tempo real
// @Summary Stream de eventos de atualização de conteúdo
// @Description Estabelece uma conexão SSE para receber atualizações em tempo real sobre o status dos conteúdos
// @Tags Events
// @Produce text/event-stream
// @Success 200 {string} string "Stream de eventos"
// @Router /events [get]
func (h *SSEHandler) GetEvents(c *gin.Context) {
	// Configura headers para SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Cache-Control")

	// Cria um contexto que será cancelado quando a conexão for fechada
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// Cria um subscriber Redis para o canal de atualizações
	pubsub := h.redis.Subscribe(ctx, "content_updates")
	defer pubsub.Close()

	// Canal para receber mensagens
	ch := pubsub.Channel()

	// Envia um evento inicial de conexão estabelecida
	fmt.Fprintf(c.Writer, "event: connected\n")
	fmt.Fprintf(c.Writer, "data: {\"message\": \"Conexão SSE estabelecida\"}\n\n")
	c.Writer.Flush()

	// Loop principal para processar eventos
	for {
		select {
		case <-ctx.Done():
			// Cliente desconectou
			log.Println("Cliente SSE desconectado")
			return

		case msg := <-ch:
			if msg == nil {
				continue
			}

			// Tenta fazer parse da mensagem como ContentUpdateEvent
			var event ContentUpdateEvent
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				log.Printf("Erro ao fazer parse do evento SSE: %v", err)
				continue
			}

			// Envia o evento para o cliente
			fmt.Fprintf(c.Writer, "event: content_update\n")
			fmt.Fprintf(c.Writer, "data: %s\n\n", msg.Payload)
			c.Writer.Flush()

			log.Printf("Evento SSE enviado: ID=%s, Status=%s", event.ID, event.Status)

		case <-time.After(30 * time.Second):
			// Envia um ping periódico para manter a conexão viva
			fmt.Fprintf(c.Writer, "event: ping\n")
			fmt.Fprintf(c.Writer, "data: {\"timestamp\": \"%s\"}\n\n", time.Now().Format(time.RFC3339))
			c.Writer.Flush()
		}
	}
}

// PublishContentUpdate publica uma atualização de conteúdo no canal Redis
// Esta função será chamada pelo worker de enriquecimento quando o status mudar
func (h *SSEHandler) PublishContentUpdate(ctx context.Context, contentID, status, eventType string) error {
	event := ContentUpdateEvent{
		ID:     contentID,
		Status: status,
		Type:   eventType,
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("erro ao serializar evento: %w", err)
	}

	err = h.redis.Publish(ctx, "content_updates", eventJSON).Err()
	if err != nil {
		return fmt.Errorf("erro ao publicar evento no Redis: %w", err)
	}

	log.Printf("Evento publicado: ID=%s, Status=%s, Type=%s", contentID, status, eventType)
	return nil
}
