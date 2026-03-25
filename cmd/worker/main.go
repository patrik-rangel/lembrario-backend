package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"lembrario-backend/internal/config"
	"lembrario-backend/internal/db"
	"lembrario-backend/internal/worker"
)

func main() {
	log.Println("Worker starting...")

	// 1. Carregar variáveis de ambiente
	if err := godotenv.Load(); err != nil {
		log.Println("Aviso: arquivo .env não encontrado, usando variáveis de ambiente do sistema")
	}

	// 2. Carregar configuração
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Erro ao carregar configuração: %v", err)
	}

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

	// 6. Configurar graceful shutdown
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	// Canal para capturar sinais do sistema
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Goroutine para lidar com shutdown
	go func() {
		sig := <-sigChan
		log.Printf("Recebido sinal %v, iniciando shutdown graceful...", sig)
		cancel()
	}()

	// 7. Iniciar o worker
	log.Println("Worker iniciado, aguardando mensagens...")
	worker.StartWorker(ctx, redisClient, queries)

	log.Println("Worker finalizado")
}
