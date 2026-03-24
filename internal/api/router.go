package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetupRouter configura as rotas da API
func SetupRouter() *gin.Engine {
	router := gin.Default()

	// Rota de saúde simples
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	// Outras rotas serão adicionadas aqui

	return router
}
