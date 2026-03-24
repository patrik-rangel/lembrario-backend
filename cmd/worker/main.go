package main

import (
	"log"

	"github.com/go-dev-api-doc/internal/queue"
	"github.com/go-dev-api-doc/internal/worker"
)

func main() {
	log.Println("Worker starting...")

	// Conecta ao Redis
	redisClient := queue.ConnectRedis()

	// Inicia o processador do worker
	worker.StartWorker(redisClient)

	// O worker.StartWorker() já é um loop infinito, então não precisamos de select {}.
	// O código após StartWorker() só seria alcançado se o worker parasse.
}
