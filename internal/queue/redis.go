package queue

import (
	"context"
	"log"
	"os"
	"strconv" // Adicionado para converter REDIS_DB

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

// ConnectRedis estabelece uma conexão com o Redis.
func ConnectRedis() *redis.Client {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379" // Valor padrão se REDIS_ADDR não estiver definido
	}

	redisPassword := os.Getenv("REDIS_PASSWORD") // Adiciona suporte a senha via variável de ambiente
	redisDBStr := os.Getenv("REDIS_DB")
	redisDB := 0 // DB padrão

	if redisDBStr != "" {
		if db, err := strconv.Atoi(redisDBStr); err == nil {
			redisDB = db
		} else {
			log.Printf("AVISO: Valor de REDIS_DB inválido '%s', usando DB 0 padrão. Erro: %v", redisDBStr, err)
		}
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword, // Usa a senha da variável de ambiente
		DB:       redisDB,       // Usa o DB da variável de ambiente
	})

	// Testa a conexão
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Falha ao conectar ao Redis em %s: %v", redisAddr, err)
	}

	log.Println("Conectado ao Redis em", redisAddr)
	return rdb
}
