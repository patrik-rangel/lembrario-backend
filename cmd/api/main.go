package main

import (
	"log"

	"github.com/joho/godotenv"
	"lembrario-backend/internal/api"
)

func main() {
	// Carregar variáveis de ambiente do arquivo .env
	if err := godotenv.Load(); err != nil {
		log.Println("Aviso: arquivo .env não encontrado, usando variáveis de ambiente do sistema")
	}

	router := api.SetupRouter()

	log.Println("API Server starting on :8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to run API server: %v", err)
	}
}
