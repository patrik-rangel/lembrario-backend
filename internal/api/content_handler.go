package api

import (
	"net/http"
	"time"
	"strconv"
	"fmt"

	"github.com/gin-gonic/gin"
	"lembrario-backend/internal/service"
)

type ContentHandler struct {
	contentService service.ContentServiceInterface
}

type ContentWithMetadata struct {
	Id        *string    `json:"id,omitempty"`
	Url       *string    `json:"url,omitempty"`
	Status    *string    `json:"status,omitempty"`
	Type      string     `json:"type"`
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
	Metadata  *Metadata  `json:"metadata,omitempty"`
	Note      *Note      `json:"note,omitempty"`
}

type Metadata struct {
	Title         string `json:"title"`
	Description   string `json:"description"`
	ThumbnailPath string `json:"thumbnailPath"`
	Transcript    string `json:"transcript"`
	Provider      string `json:"provider"`
	ReadingTime   int    `json:"readingTime"`
}

type Note struct {
	Id        *string    `json:"id,omitempty"`
	Body      *string    `json:"body,omitempty"`
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

// type CreateContentRequest struct {
// 	Url  string `json:"url" binding:"required"`
// 	Type string `json:"type" binding:"required"`
// }

// type CreateContentResponse struct {
// 	Id  *string `json:"id,omitempty"`
// 	Url *string `json:"url,omitempty"`
// }

// type GetContentsParams struct {
// 	Limit  *int `form:"limit"`
// 	Offset *int `form:"offset"`
// }

// type UpdateNoteRequest struct {
// 	Body string `json:"body" binding:"required"`
// }

func NewContentHandler(contentService service.ContentServiceInterface) *ContentHandler {
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
		URL:  req.Url,
		Type: req.Type,
	}

	content, err := h.contentService.Create(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao criar conteúdo", "details": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, CreateContentResponse{
		Id:  &content.ID,
		Url: &content.Url,
	})
}

// GetContents lista conteúdos com paginação.
func (h *ContentHandler) GetContents(c *gin.Context, params GetContentsParams) {
	// Valores padrão para paginação
	limit := int32(20)
	offset := int32(0)

	// Parse dos parâmetros de query se fornecidos
	if params.Limit != nil {
		limit = int32(*params.Limit)
	}
	if params.Offset != nil {
		offset = int32(*params.Offset)
	}

	serviceParams := service.GetContentsParams{
		Limit:  limit,
		Offset: offset,
	}

	contents, err := h.contentService.GetContents(c.Request.Context(), serviceParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao buscar conteúdos", "details": err.Error()})
		return
	}

	// Converter para o formato da API
	var response []ContentWithMetadata
	for _, content := range contents {
		item := ContentWithMetadata{
			Id:        &content.ID,
			Url:       &content.Url,
			Status:    &content.Status,
			Type:      content.Type.String,
			CreatedAt: &content.CreatedAt.Time,
			UpdatedAt: &content.UpdatedAt.Time,
		}

		// Adicionar metadados se existirem
		if content.Title.Valid {
			item.Metadata = &Metadata{
				Title:         content.Title.String,
				Description:   content.Description.String,
				ThumbnailPath: content.ThumbnailPath.String,
				Transcript:    content.Transcript.String,
				Provider:      content.Provider.String,
				ReadingTime:   int(content.ReadingTime.Int32),
			}
		}

		// Adicionar nota se existir
		if content.NoteID.Valid {
			item.Note = &Note{
				Id:        &content.NoteID.String,
				Body:      &content.NoteBody.String,
				CreatedAt: &content.NoteCreatedAt.Time,
				UpdatedAt: &content.NoteUpdatedAt.Time,
			}
		}

		response = append(response, item)
	}

	c.JSON(http.StatusOK, response)
}

// GetContentByID busca um conteúdo específico por ID.
func (h *ContentHandler) GetContentByID(c *gin.Context, id string) {
	content, err := h.contentService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Conteúdo não encontrado"})
		return
	}

	response := ContentWithMetadata{
		Id:        &content.ID,
		Url:       &content.Url,
		Status:    &content.Status,
		Type:      content.Type.String,
		CreatedAt: &content.CreatedAt.Time,
		UpdatedAt: &content.UpdatedAt.Time,
	}

	// Adicionar metadados se existirem
	if content.Title.Valid {
		response.Metadata = &Metadata{
			Title:         content.Title.String,
			Description:   content.Description.String,
			ThumbnailPath: content.ThumbnailPath.String,
			Transcript:    content.Transcript.String,
			Provider:      content.Provider.String,
			ReadingTime:   int(content.ReadingTime.Int32),
		}
	}

	// Adicionar nota se existir
	if content.NoteID.Valid {
		response.Note = &Note{
			Id:        &content.NoteID.String,
			Body:      &content.NoteBody.String,
			CreatedAt: &content.NoteCreatedAt.Time,
			UpdatedAt: &content.NoteUpdatedAt.Time,
		}
	}

	c.JSON(http.StatusOK, response)
}

// UpdateNote cria ou atualiza uma nota associada a um conteúdo.
func (h *ContentHandler) UpdateNote(c *gin.Context, id string) {
	var req UpdateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	params := service.UpdateNoteParams{
		ContentID: id,
		Body:      req.Body,
	}

	note, err := h.contentService.UpsertNote(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao atualizar nota", "details": err.Error()})
		return
	}

	response := Note{
		Id:        &note.ID,
		Body:      &note.Body,
		CreatedAt: &note.CreatedAt.Time,
		UpdatedAt: &note.UpdatedAt.Time,
	}

	c.JSON(http.StatusOK, response)
}

// DeleteContent remove um conteúdo e todos os dados associados.
func (h *ContentHandler) DeleteContent(c *gin.Context, id string) {
	err := h.contentService.Delete(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao deletar conteúdo", "details": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *ContentHandler) GetSearch(c *gin.Context) {
    query := c.Query("q")

    limit := int64(20)
    if l := c.Query("limit"); l != "" {
        if parsed, err := strconv.ParseInt(l, 10, 64); err == nil && parsed > 0 && parsed <= 100 {
            limit = parsed
        }
    }

    offset := int64(0)
    if o := c.Query("offset"); o != "" {
        if parsed, err := strconv.ParseInt(o, 10, 64); err == nil && parsed >= 0 {
            offset = parsed
        }
    }

    // Filtro opcional: ?filter=type = video
    filter := c.Query("filter")

    result, err := h.contentService.Search(query, filter, limit, offset)
    if err != nil {
        fmt.Printf("Erro na busca: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao realizar busca"})
        return
    }

    c.JSON(http.StatusOK, result)
}
