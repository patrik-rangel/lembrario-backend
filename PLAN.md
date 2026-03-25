🏗️ Fase 1: Infraestrutura e Configuração
Objetivo: Subir o motor de busca e preparar as credenciais.

[x] Docker Compose: Adicionar o serviço meilisearch ao docker-compose.yml.

[x] Variáveis de Ambiente: Adicionar MEILI_HOST e MEILI_MASTER_KEY ao .env e .env.example.

[x] Client Go: Instalar o SDK oficial: go get github.com/meilisearch/meilisearch-go.

🔄 Fase 2: Indexação no Worker (O "Push")
Objetivo: Garantir que, assim que o Worker terminar o scraping, o dado vá para o índice.

[ ] Internal Package: Criar internal/search/meilisearch.go para centralizar a conexão e as operações de indexação.

[ ] Worker Integration: No final do processamento do Worker (após o UpsertMetadata), enviar o "Documento Rico" para o Meilisearch.

[ ] Document Schema: Definir a estrutura do documento (ID, Title, Description, URL, Type, CreatedAt).

🔍 Fase 3: Endpoint de Busca na API
Objetivo: Expor a funcionalidade para o Frontend.

[ ] Service Layer: Adicionar o método Search(query string) no ContentService.

[ ] Handler API: Implementar o GET /search?q=termo no api/handler.go.

[ ] Search Options: Configurar AttributesToHighlight para que o frontend possa mostrar onde o termo foi encontrado.

🧹 Fase 4: Sincronização e Ajustes (Opcional)
[ ] Backfill Script: (Opcional) Um script simples para indexar os links que você já salvou no banco antes de ter o Meilisearch.