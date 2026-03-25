package api

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"lembrario-backend/internal/db"                   // Para o pool do DB e sqlc queries
	"lembrario-backend/internal/queue"                // Para o cliente Redis
	"lembrario-backend/internal/service"              // Para o serviço de conteúdo
)

// apiServer implementa a interface ServerInterface gerada pelo oapi-codegen
type apiServer struct {
	contentHandler *ContentHandler
	sseHandler     *SSEHandler
	// Outros handlers podem ser adicionados aqui conforme necessário
}

// PostContents implementa o endpoint POST /contents, delegando para o ContentHandler
func (s *apiServer) PostContents(c *gin.Context) {
	s.contentHandler.CreateContent(c)
}

// GetEvents implementa o endpoint GET /events, delegando para o SSEHandler
func (s *apiServer) GetEvents(c *gin.Context) {
	s.sseHandler.GetEvents(c)
}

// GetContents implementa o endpoint GET /contents
func (s *apiServer) GetContents(c *gin.Context, params GetContentsParams) {
	s.contentHandler.GetContents(c, params)
}

// GetContentsId implementa o endpoint GET /contents/{id}
func (s *apiServer) GetContentsId(c *gin.Context, id string) {
	s.contentHandler.GetContentByID(c, id)
}

// UpdateNote implementa o endpoint PATCH /contents/{id}/note
func (s *apiServer) UpdateNote(c *gin.Context, id string) {
	s.contentHandler.UpdateNote(c, id)
}

// DeleteContentsId implementa o endpoint DELETE /contents/{id}
func (s *apiServer) DeleteContentsId(c *gin.Context, id string) {
	s.contentHandler.DeleteContent(c, id)
}

// Método stub para o endpoint de busca que ainda não está implementado
func (s *apiServer) GetSearch(c *gin.Context, params GetSearchParams) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Endpoint GET /search não implementado"})
}

// SetupRouter configura as rotas da API
func SetupRouter() *gin.Engine {
	router := gin.Default()

	// Inicializa o pool do banco de dados
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dbPool, err := db.NewDBPool(ctx)
	if err != nil {
		log.Fatalf("Falha ao inicializar o pool do banco de dados: %v", err)
	}

	// Inicializa o cliente Redis
	redisClient := queue.ConnectRedis()

	// Inicializa as queries do sqlc
	queries := db.New(dbPool)

	// Inicializa o ContentService
	contentService := service.NewContentService(queries, redisClient)

	// Inicializa os Handlers
	contentHandler := NewContentHandler(contentService)
	sseHandler := NewSSEHandler(redisClient)

	// Rota de saúde simples (mantida separada pois geralmente não está na especificação OpenAPI)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	// Inicializa o servidor da API com os handlers
	server := &apiServer{
		contentHandler: contentHandler,
		sseHandler:     sseHandler,
	}

	// Registra todas as rotas da API usando a função gerada pelo oapi-codegen
	// Isso garante que todas as rotas da especificação OpenAPI sejam mapeadas
	RegisterHandlers(router, server)

	return router
}
