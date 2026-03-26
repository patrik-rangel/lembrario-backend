package api

import (
	"net/http"
	"os"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *AuthService
}

func NewAuthHandler(s *AuthService) *AuthHandler {
	return &AuthHandler{authService: s}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: ptr("Payload inválido")})
		return
	}

	// Validação simples via ENV para o seu uso pessoal na VM
	adminUser := os.Getenv("ADMIN_USER")
	adminPass := os.Getenv("ADMIN_PASSWORD")

	if req.Username != adminUser || req.Password != adminPass {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: ptr("Usuário ou senha inválidos")})
		return
	}

	token, expires, err := h.authService.GenerateToken(req.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: ptr("Erro ao gerar token")})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Token:     &token,
		ExpiresIn: &expires,
	})
}