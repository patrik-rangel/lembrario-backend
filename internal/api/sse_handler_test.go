package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"lembrario-backend/internal/api"
)

// ─── Helpers ──────────────────────────────────────────────────────────────────

// setupSSETest retorna o roteador e o cliente redis para podermos publicar eventos
func setupSSETest(t *testing.T) (*gin.Engine, *redis.Client) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	
	// Mock simples pois o SSEHandler não depende do service
	contentHandler := api.NewContentHandler(&mockContentService{})
	sseHandler := api.NewSSEHandler(rdb)
	
	return api.SetupRouter(contentHandler, sseHandler), rdb
}

// ─── SSE Headers ──────────────────────────────────────────────────────────────

func TestEventsEndpointHeadersSse(t *testing.T) {
	r, _ := setupSSETest(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancela imediatamente para o handler não travar o teste

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	req = req.WithContext(ctx)
	
	r.ServeHTTP(w, req)

	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", w.Header().Get("Connection"))
}

// ─── SSE Streaming ────────────────────────────────────────────────────────────

func TestEventsStreamingFlow(t *testing.T) {
	r, rdb := setupSSETest(t)

	// Contexto para fechar a conexão após recebermos o que queremos
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	req = req.WithContext(ctx)

	// Canal para avisar que a requisição terminou
	done := make(chan bool)

	go func() {
		r.ServeHTTP(w, req)
		done <- true
	}()

	// 1. Aguarda um instante para a subscrição no Redis ocorrer
	time.Sleep(50 * time.Millisecond)

	// 2. Publica um evento simulando o Worker
	expectedEvent := api.ContentUpdateEvent{
		ID:     "01KMK",
		Status: "COMPLETED",
		Type:   "video",
	}
	payload, _ := json.Marshal(expectedEvent)
	rdb.Publish(context.Background(), "content_updates", string(payload))

	// 3. Pequena pausa para o processamento e encerramos a conexão
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	resp := w.Body.String()

	// Valida se os eventos foram escritos no stream
	assert.Contains(t, resp, "event: connected", "Deve enviar evento de conexão inicial")
	assert.Contains(t, resp, "event: content_update", "Deve enviar evento de atualização")
	assert.Contains(t, resp, expectedEvent.ID, "O payload deve conter o ID correto")
	assert.Contains(t, resp, expectedEvent.Status, "O payload deve conter o status correto")
}

// ─── SSE Publish Helper ───────────────────────────────────────────────────────

func TestSSEHandler_PublishContentUpdate(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	handler := api.NewSSEHandler(rdb)

	// Se inscreve no canal para validar o que foi publicado
	pubsub := rdb.Subscribe(context.Background(), "content_updates")
	ch := pubsub.Channel()

	contentID := "01TEST"
	status := "ERROR"
	eventType := "article"

	err := handler.PublishContentUpdate(context.Background(), contentID, status, eventType)
	assert.NoError(t, err)

	// Valida se a mensagem chegou no Redis formatada corretamente
	select {
	case msg := <-ch:
		var event api.ContentUpdateEvent
		err := json.Unmarshal([]byte(msg.Payload), &event)
		assert.NoError(t, err)
		assert.Equal(t, contentID, event.ID)
		assert.Equal(t, status, event.Status)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout esperando mensagem no Redis")
	}
}