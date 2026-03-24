package main

import "log"

func main() {
	log.Println("Worker starting...")
	// Lógica do worker será implementada aqui
	select {} // Mantém o worker rodando indefinidamente
}
