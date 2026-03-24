package main

import (
	"log"

	"lembrario-backend/internal/queue"
	"lembrario-backend/internal/worker"
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
