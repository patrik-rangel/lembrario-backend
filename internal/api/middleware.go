package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

func JWTMiddleware(authService *AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Exceções: Rotas que NÃO precisam de token
		path := c.Request.URL.Path
		if path == "/login" || path == "/health" || strings.HasSuffix(path, "/health") {
			c.Next()
			return
		}

		// 2. Obter o header Authorization
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token de autenticação ausente"})
			c.Abort()
			return
		}

		// 3. Validar formato "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Formato de autorização inválido (use Bearer)"})
			c.Abort()
			return
		}

		// 4. Validar o token real
		token, err := authService.ValidateToken(parts[1])
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token inválido ou expirado"})
			c.Abort()
			return
		}

		// Opcional: Salvar o username no contexto para uso posterior
		// claims, ok := token.Claims.(jwt.MapClaims)
		// if ok {
		// 	c.Set("username", claims["sub"])
		// }

		c.Next()
	}
}

// CORSMiddleware configura as permissões de acesso para o frontend
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Em produção, você deve substituir o "*" pela URL real do seu app Angular
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")

		// Trata a requisição de preflight (quando o browser pergunta se pode fazer o POST)
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
