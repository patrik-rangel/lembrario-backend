package main

import (
	"log"

	"lembrario-backend/internal/api"
)

func main() {
	router := api.SetupRouter()

	log.Println("API Server starting on :8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to run API server: %v", err)
	}
}
