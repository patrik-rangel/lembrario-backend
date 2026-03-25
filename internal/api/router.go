package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
func SetupRouter(contentHandler *ContentHandler, sseHandler *SSEHandler) *gin.Engine {
	router := gin.Default()

	// Rota de saúde
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	// Criar servidor da API
	server := &apiServer{
		contentHandler: contentHandler,
		sseHandler:     sseHandler,
	}

	// Registrar rotas
	RegisterHandlers(router, server)

	return router
}
