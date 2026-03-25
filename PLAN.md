
````markdown
# 📋 Plano de Implementação: Fluxo de Conteúdo (Lembrário)

Este documento descreve as etapas para implementar o fluxo completo de ingestão, enriquecimento assíncrono e gestão de notas, garantindo a paridade entre o OpenAPI, os Diagramas de Sequência e o código Go.

---

## 🏗️ Fase 1: Persistência e Contrato (SQLC & ULID)

**Objetivo:** Preparar o banco de dados para receber os novos registros.

- [ ] **Queries SQLC:** Adicionar em `internal/db/queries.sql` a instrução para criar conteúdo  
  - **Query:** `CreateContent` (`INSERT INTO contents ... RETURNING *`)  
  - **Campos:** `id` (ULID), `url`, `type`, `status` (default `'PENDING'`)

- [ ] **Geração de Código:** Rodar `sqlc generate` para atualizar as interfaces do banco

- [ ] **Validação de ID:** Garantir que o pacote `pkg/id` está pronto para gerar os ULIDs necessários para a chave primária

---

## 🚀 Fase 2: Ingestão Inicial (API First)

**Objetivo:** Receber a URL e responder rapidamente ao usuário (**Status 202**).

- [ ] **ContentHandler (PostContents):**
  - Realizar o `ShouldBindJSON` para a struct `CreateContentRequest`
  - Chamar o método `Create` do `ContentService`
  - Retornar `202 Accepted` com o ID gerado

- [ ] **ContentService (Create):**
  - Gerar o ULID
  - Persistir no banco via SQLC com status `PENDING`
  - Próximo passo: Integrar com a fila do Redis (Fase 3)

---

## 📨 Fase 3: Mensageria e Fila (Redis)

**Objetivo:** Despachar a tarefa para o processamento em background.

- [ ] **Redis Client:** Configurar o produtor no pacote `internal/queue`

- [ ] **Producer Logic:**  
  No `ContentService`, após salvar no banco, disparar um `LPUSH` para a chave `enrichment_queue` com o JSON:
  ```json
  { "id": "...", "url": "..." }
````

* [ ] **Fallback de Erro:**
  Se a fila falhar:

  * Atualizar o status no banco para `ERROR`, **ou**
  * Reverter a transação

---

## 📡 Fase 4: Feedback em Tempo Real (SSE)

**Objetivo:** Manter o Frontend atualizado sem refresh.

* [ ] **SSE Handler (GetEvents):**

  * Implementar o loop de stream (`text/event-stream`)
  * Realizar o `SUBSCRIBE` no canal Redis `content_updates`
  * Enviar eventos ao frontend quando o status mudar (`COMPLETED` / `ERROR`)

---

## 📝 Fase 5: Gestão de Notas (CRUD 1:1)

**Objetivo:** Permitir anotações em Markdown vinculadas ao conteúdo.

* [ ] **Upsert de Notas:**
  Implementar o handler `UpdateNote` (`PATCH`/`PUT`)

  * Usar `ON CONFLICT (content_id) DO UPDATE` no SQL para garantir o vínculo 1:1

* [ ] **Busca Enriquecida:**
  Atualizar o `GetContentsId` para retornar o conteúdo com o `JOIN` da nota associada

---

## 🧹 Fase 6: Exclusão (Cleanup Total)

**Objetivo:** Não deixar lixo no sistema.

* [ ] **Delete Flow:**

  * Remover do Postgres (Cascade)
  * Remover do índice do Meilisearch
  * (Futuro) Remover assets físicos do storage

```
