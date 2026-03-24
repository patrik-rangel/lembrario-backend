package api

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-dev-api-doc/internal/api/content_handler" // Novo handler
	"github.com/go-dev-api-doc/internal/db"                   // Para o pool do DB e sqlc queries
	"github.com/go-dev-api-doc/internal/queue"                // Para o cliente Redis
	"github.com/go-dev-api-doc/internal/service"              // Para o serviço de conteúdo
)

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
	contentHandler := content_handler.NewContentHandler(contentService)

	// Rota de saúde simples
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	// Rotas de Conteúdo
	contentRoutes := router.Group("/contents")
	{
		contentRoutes.POST("", contentHandler.CreateContent)
	}

	// Outras rotas serão adicionadas aqui

	return router
}
