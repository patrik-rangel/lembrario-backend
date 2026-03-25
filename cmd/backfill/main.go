package main

import (
    "context"
    "log"
    "os"

    "github.com/joho/godotenv"
    "lembrario-backend/internal/db"
    "lembrario-backend/internal/search"
	"lembrario-backend/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
    _ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Erro ao carregar configuração: %v", err)
	}

    // Inicializa conexões (mesmo padrão do seu main.go)
    // dbConn := mustConnectDB()
	dbConn, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Erro ao conectar ao banco de dados: %v", err)
	}
	defer dbConn.Close()
    queries := db.New(dbConn)
    searchClient, err := search.New()
    if err != nil {
        log.Fatalf("Meilisearch: %v", err)
    }

    ctx := context.Background()

    // Busca todos os conteúdos com status COMPLETED que têm metadados
    contents, err := queries.ListContentsWithMetadata(ctx)
    if err != nil {
        log.Fatalf("Erro ao buscar conteúdos: %v", err)
    }

    log.Printf("Indexando %d conteúdos...", len(contents))

    ok, failed := 0, 0
    for _, c := range contents {
        doc := search.ContentDocument{
            ID:          c.ID,
            Title:       c.Title.String,
            Description: c.Description.String,
            URL:         c.Url,
            Type:        c.Type.String,
            Provider:    c.Provider.String,
            CreatedAt:   c.CreatedAt.Time,
        }
        if err := searchClient.IndexContent(doc); err != nil {
            log.Printf("❌ Falha ao indexar %s: %v", c.ID, err)
            failed++
        } else {
            ok++
        }
    }

    log.Printf("✅ Concluído — %d indexados, %d falhas", ok, failed)
    os.Exit(0)
}