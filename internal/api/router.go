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
	// Outros handlers podem ser adicionados aqui conforme necessário
}

// PostContents implementa o endpoint POST /contents, delegando para o ContentHandler
func (s *apiServer) PostContents(c *gin.Context) {
	s.contentHandler.CreateContent(c)
}

// Métodos stub para os outros endpoints da ServerInterface que ainda não estão implementados
func (s *apiServer) GetContents(c *gin.Context, params GetContentsParams) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Endpoint GET /contents não implementado"})
}
func (s *apiServer) DeleteContentsId(c *gin.Context, id string) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Endpoint DELETE /contents/{id} não implementado"})
}
func (s *apiServer) GetContentsId(c *gin.Context, id string) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Endpoint GET /contents/{id} não implementado"})
}
func (s *apiServer) UpdateNote(c *gin.Context, id string) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Endpoint PATCH /contents/{id} não implementado"})
}
func (s *apiServer) GetEvents(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Endpoint GET /events não implementado"})
}
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

	// Rota de saúde simples (mantida separada pois geralmente não está na especificação OpenAPI)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	// Inicializa o servidor da API com os handlers
	server := &apiServer{
		contentHandler: contentHandler,
	}

	// Registra todas as rotas da API usando a função gerada pelo oapi-codegen
	// Isso garante que todas as rotas da especificação OpenAPI sejam mapeadas
	RegisterHandlers(router, server)

	return router
}
