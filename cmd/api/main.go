package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"lembrario-backend/internal/api"
	"lembrario-backend/internal/config"
	"lembrario-backend/internal/db"
	"lembrario-backend/internal/service"
	"lembrario-backend/internal/search"
)

func main() {
	// 1. Carregar variáveis de ambiente
	if err := godotenv.Load(); err != nil {
		log.Println("Aviso: arquivo .env não encontrado, usando variáveis de ambiente do sistema")
	}

	// 2. Carregar configuração
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Erro ao carregar configuração: %v", err)
	}

	searchClient, err := search.New()
	if err != nil {
		log.Fatalf("Erro ao conectar Meilisearch: %v", err)
	}
	log.Println("Conexão com Meilisearch estabelecida")


	// 3. Conectar ao banco de dados
	dbPool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Erro ao conectar ao banco de dados: %v", err)
	}
	defer dbPool.Close()

	// Testar conexão
	if err := dbPool.Ping(context.Background()); err != nil {
		log.Fatalf("Erro ao fazer ping no banco de dados: %v", err)
	}
	log.Println("Conexão com banco de dados estabelecida")

	// 4. Conectar ao Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})
	defer redisClient.Close()

	// Testar conexão Redis
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Erro ao conectar ao Redis: %v", err)
	}
	log.Println("Conexão com Redis estabelecida")

	// 5. Criar instância das queries SQLC
	queries := db.New(dbPool)

	// 6. Criar services
	contentService := service.NewContentService(queries, redisClient, searchClient)

	// 7. Criar handlers
	contentHandler := api.NewContentHandler(contentService)
	sseHandler := api.NewSSEHandler(redisClient)

	// 8. Criar router com handlers injetados
	router := api.SetupRouter(contentHandler, sseHandler)

	// 9. Iniciar servidor
	port := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("API Server starting on %s", port)
	if err := router.Run(port); err != nil {
		log.Fatalf("Failed to run API server: %v", err)
	}
}
