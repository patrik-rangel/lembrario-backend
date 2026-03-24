# 📋 Plano de Execução - Fase 1: Infraestrutura e Ingestão

**Objetivo:** Estabelecer a camada de dados (Postgres + sqlc), mensageria (Redis) e o primeiro endpoint de ingestão assíncrona.

---

## 3. Fluxo de Ingestão (API)
- [ ] **Service (`internal/service/content_service.go`):**
  - Método `Create(url)`: 
    1. Gera ULID via `pkg/id`.
    2. Salva no DB via `sqlc`.
    3. Envia `{id, url}` para a fila Redis `enrichment_queue`.
- [ ] **Handler (`internal/api/content_handler.go`):**
  - Implementar `POST /contents` que recebe a URL no body.
  - Chamar o service e retornar `202 Accepted` com o ULID.
- [ ] **Router (`internal/api/router.go`):**
  - Registrar a rota `POST /contents`.

---

## 4. Bootstrap do Worker
- [ ] **Consumidor (`internal/worker/processor.go`):**
  - Criar um loop que escuta a `enrichment_queue` usando `BRPOP`.
  - Por enquanto, apenas logar: "Processando conteúdo ID: {id} para a URL: {url}".
  - Adicionar um `time.Sleep(2 * time.Second)` para simular processamento.

---
