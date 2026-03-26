package api

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AuthService struct {
	secretKey []byte
}

func NewAuthService() *AuthService {
	return &AuthService{
		secretKey: []byte(os.Getenv("JWT_SECRET")),
	}
}

// GenerateToken cria um JWT válido por 24 horas
func (s *AuthService) GenerateToken(username string) (string, int, error) {
	expiresIn := 86400 // 24 horas
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": username,
		"exp": time.Now().Add(time.Hour * 24).Unix(),
		"iat": time.Now().Unix(),
	})

	tokenString, err := token.SignedString(s.secretKey)
	return tokenString, expiresIn, err
}

// ValidateToken verifica se o token é legítimo e não expirou
func (s *AuthService) ValidateToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("método de assinatura inesperado: %v", token.Header["alg"])
		}
		return s.secretKey, nil
	})
}
