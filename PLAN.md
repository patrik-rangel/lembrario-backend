
````markdown
# 📋 Plano de Implementação: Fluxo de Conteúdo (Lembrário)

Este documento descreve as etapas para implementar o fluxo completo de ingestão, enriquecimento assíncrono e gestão de notas, garantindo a paridade entre o OpenAPI, os Diagramas de Sequência e o código Go.

---

## 🏗️ Fase 1: Persistência e Contrato (SQLC & ULID) ✅ **COMPLETA**

**Objetivo:** Preparar o banco de dados para receber os novos registros.

- [x] **Queries SQLC:** Adicionar em `internal/db/queries.sql` a instrução para criar conteúdo  
  - **Query:** `CreateContent` (`INSERT INTO contents ... RETURNING *`)  
  - **Campos:** `id` (ULID), `url`, `type`, `status` (default `'PENDING'`)

- [x] **Geração de Código:** Rodar `sqlc generate` para atualizar as interfaces do banco

- [x] **Validação de ID:** Garantir que o pacote `pkg/id` está pronto para gerar os ULIDs necessários para a chave primária

---

## 🚀 Fase 2: Ingestão Inicial (API First) ✅ **COMPLETA**

**Objetivo:** Receber a URL e responder rapidamente ao usuário (**Status 202**).

- [x] **ContentHandler (PostContents):**
  - Realizar o `ShouldBindJSON` para a struct `CreateContentRequest`
  - Chamar o método `Create` do `ContentService`
  - Retornar `202 Accepted` com o ID gerado

- [x] **ContentService (Create):**
  - Gerar o ULID
  - Persistir no banco via SQLC com status `PENDING`
  - Integração com a fila do Redis implementada

---

## 📨 Fase 3: Mensageria e Fila (Redis) ✅ **COMPLETA**

**Objetivo:** Despachar a tarefa para o processamento em background.

- [x] **Redis Client:** Configurado no pacote `internal/queue`

- [x] **Producer Logic:**  
  No `ContentService`, após salvar no banco, disparar um `LPUSH` para a chave `enrichment_queue` com o JSON:
  ```json
  { "id": "...", "url": "..." }
````

* [x] **Fallback de Erro:**
  Se a fila falhar, apenas loga o erro sem reverter a criação do conteúdo

---

## 📡 Fase 4: Feedback em Tempo Real (SSE) ✅ **COMPLETA**

**Objetivo:** Manter o Frontend atualizado sem refresh.

* [x] **SSE Handler (GetEvents):**

  * Implementar o loop de stream (`text/event-stream`)
  * Realizar o `SUBSCRIBE` no canal Redis `content_updates`
  * Enviar eventos ao frontend quando o status mudar (`COMPLETED` / `ERROR`)
  * Ping periódico para manter conexão viva
  * Tratamento adequado de desconexão do cliente

* [x] **Publisher Helper:**
  Função `PublishContentUpdate` para ser usada pelo worker de enriquecimento

---

## 📝 Fase 5: Gestão de Notas (CRUD 1:1) ✅ **COMPLETA**

**Objetivo:** Permitir anotações em Markdown vinculadas ao conteúdo.

* [x] **Upsert de Notas:**
  Implementar o handler `UpdateNote` (`PATCH`/`PUT`)

  * Usar `ON CONFLICT (content_id) DO UPDATE` no SQL para garantir o vínculo 1:1

* [x] **Busca Enriquecida:**
  Atualizar o `GetContentsId` para retornar o conteúdo com o `JOIN` da nota associada

* [x] **Listagem com Notas:**
  Atualizar o `GetContents` para incluir notas na listagem paginada

---

## 🧹 Fase 6: Exclusão (Cleanup Total) ✅ **COMPLETA**

**Objetivo:** Não deixar lixo no sistema.

* [x] **Delete Flow:**

  * Remover do Postgres (Cascade automático via FK constraints)
  * Handler `DeleteContent` implementado
  * Cleanup completo de conteúdo, metadados e notas

---

## 🎯 Status Atual: **Fases 5 e 6 Completas!**

O sistema agora possui:
1. ✅ Ingestão de conteúdo com resposta `202 Accepted`
2. ✅ Persistência no banco com status `PENDING`
3. ✅ Enfileiramento no Redis para processamento assíncrono
4. ✅ Stream SSE para atualizações em tempo real
5. ✅ **Gestão completa de notas (CRUD 1:1)**
6. ✅ **Exclusão com cleanup total**

**Próximos passos opcionais:**
- Implementar busca com Meilisearch (`GET /search`)
- Criar worker de enriquecimento que consome a fila Redis
- Adicionar validações mais robustas
- Implementar rate limiting
- Adicionar logs estruturados

## 📋 Checklist Pós-Implementação

**Antes de testar, execute:**

```bash
# 1. Gerar código SQLC para as novas queries
sqlc generate

# 2. Aplicar migrações do banco (se necessário)
# Certifique-se de que as tabelas 'notes' existem com FK para 'contents'

# 3. Reiniciar o servidor
go run cmd/server/main.go
```

**Endpoints agora funcionais:**
- ✅ `POST /contents` - Criar conteúdo
- ✅ `GET /contents` - Listar conteúdos (com paginação)
- ✅ `GET /contents/{id}` - Buscar conteúdo específico
- ✅ `PATCH /contents/{id}/note` - Criar/atualizar nota
- ✅ `DELETE /contents/{id}` - Deletar conteúdo
- ✅ `GET /events` - Stream SSE
- ⏳ `GET /search` - Busca (ainda não implementado)

```
