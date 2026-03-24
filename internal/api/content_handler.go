package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"lembrario-backend/internal/service"
)

// CreateContentRequest representa o corpo da requisição POST /contents.
type CreateContentRequest struct {
	URL  string `json:"url" binding:"required"`
	Type string `json:"type" binding:"required"`
}

// CreateContentResponse representa o corpo da resposta para POST /contents.
type CreateContentResponse struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type ContentHandler struct {
	contentService *service.ContentService
}

func NewContentHandler(contentService *service.ContentService) *ContentHandler {
	return &ContentHandler{
		contentService: contentService,
	}
}

// CreateContent lida com a criação de um novo conteúdo.
// @Summary Cria um novo conteúdo para enriquecimento assíncrono
// @Description Recebe uma URL e o tipo de conteúdo, salva no banco de dados e enfileira para processamento.
// @Tags Contents
// @Accept json
// @Produce json
// @Param request body CreateContentRequest true "URL e Tipo do conteúdo"
// @Success 202 {object} CreateContentResponse
// @Failure 400 {object} gin.H "Requisição inválida"
// @Failure 500 {object} gin.H "Erro interno do servidor"
// @Router /contents [post]
func (h *ContentHandler) CreateContent(c *gin.Context) {
	var req CreateContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	params := service.CreateContentParams{
		URL:  req.URL,
		Type: req.Type,
	}

	content, err := h.contentService.Create(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao criar conteúdo", "details": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, CreateContentResponse{
		ID:  content.ID,
		URL: content.Url,
	})
}
