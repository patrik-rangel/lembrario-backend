🏗️ Plano de Implementação: Worker de Enriquecimento
Objetivo: Criar um binário independente que consome a enrichment_queue, extrai metadados da web e notifica o backend via Pub/Sub.

📂 Fase 1: Estrutura e Contrato (SQLC)
Objetivo: Preparar o banco para receber os dados extraídos pelo Worker.

[x] Queries SQLC: Adicionar em internal/db/queries.sql:

UpsertMetadata: Insere ou atualiza o título, descrição, provider, etc.

UpdateContentStatus: Muda de PENDING para COMPLETED ou ERROR.

[x] Geração: Rodar sqlc generate.

🔄 Fase 2: O Loop de Consumo (Consumer)
Objetivo: Tirar as mensagens do Redis e iniciar o processamento.

[x] Worker Entrypoint: Criar cmd/worker/main.go (reutilizando o db.NewDBPool e a fiação do main.go da API).

[x] Consumer Loop: Implementar um loop infinito usando BRPOP (bloqueante) na enrichment_queue.

[x] Graceful Shutdown: Garantir que o worker termine de processar a tarefa atual antes de fechar ao receber um SIGINT/SIGTERM.

🌐 Fase 3: Engine de Scraping (A "Mágica")
Objetivo: Ir até a internet e buscar as informações.

[ ] Scraper Service: Criar internal/worker/scraper.go.

[ ] Extração Básica: Usar net/http e goquery para extrair <title> e <meta name="description">.

[ ] Identificação de Provider: Lógica simples para detectar se é YouTube (para futura integração com API de Transcrições).

📢 Fase 4: Notificação e Persistência
Objetivo: Salvar os dados e avisar o Frontend via SSE.

[ ] Update DB: Salvar os metadados e atualizar o status do conteúdo para COMPLETED.

[ ] Notify Pub/Sub: Publicar no canal content_updates do Redis o payload: {"id": "ULID", "status": "COMPLETED"}.

[ ] Trigger SSE: Validar se o backend (que já está rodando) recebe essa mensagem e repassa para o curl -i -N.

⚠️ Fase 5: Tratamento de Erro (Sad Path)
Objetivo: Implementar a estratégia de retry conforme o diagrama.

[ ] Retry Logic: Se o scraping falhar, incrementar um contador e devolver para a fila (com delay) ou mover para uma dead_letter_queue e marcar como ERROR.