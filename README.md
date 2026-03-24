# 📚 Lembrário - Backend (Go)

O **Lembrário** é um serviço de API e processamento assíncrono para ingestão, enriquecimento e organização de links.
O sistema transforma URLs brutas em documentos ricos com metadados e notas estruturadas.

---

## 🚀 Tech Stack

* **Linguagem:** Go 1.26.0
* **Framework Web:** Gin Gonic (ou similar, focado em performance)
* **Banco de Dados:** PostgreSQL
* **Gerenciador de SQL:** sqlc (Type-safe SQL generator)
* **Migrações:** golang-migrate (ou similar)
* **Mensageria:** Redis (Fila de enriquecimento + Pub/Sub para SSE)
* **Busca:** Meilisearch
* **Identificadores:** ULID
## 🔌 Comunicação
* **API:** RESTful para operações de CRUD e busca.
* **Real-time:** SSE (Server-Sent Events) exclusivamente para notificações de progresso do Worker (status updates).

---

## 🏗️ Arquitetura de Software

O projeto é estruturado como um **monorepo**, contendo os principais processos do backend:

### 🔹 API Server

Responsável por:

* Autenticação (futuro)
* CRUD de notas
* Interface de busca

### 🔹 Worker

Responsável por:

* Processamento assíncrono
* Scraping de conteúdo
* Transcrições e enriquecimento

---

## 📁 Estrutura de Diretórios

```plaintext
.
├── cmd/
│   ├── api/            # Ponto de entrada da API
│   └── worker/         # Ponto de entrada do Processador
├── internal/
│   ├── api/            # Handlers e middlewares HTTP
│   ├── db/             # Código gerado pelo sqlc
│   ├── service/        # Lógica de negócio (Orquestração)
│   ├── worker/         # Implementação dos Jobs de enriquecimento
│   └── queue/          # Integração com Redis
├── sql/
│   ├── schema/         # Migrações e definição de tabelas (.sql)
│   └── queries/        # Queries SQL para o sqlc (.sql)
├── sqlc.yaml           # Configuração do gerador sqlc
└── docs/               # Diagramas de sequência e arquitetura
```

---

## 🛠️ Regras de Desenvolvimento

### 🧠 SQL First

Nenhuma query deve ser escrita manualmente no código Go.

* Defina queries em:
  `sql/queries/`
* Gere código com:

```bash
sqlc generate
```

---

### 🔑 IDs Imutáveis

* Utilize sempre **ULID** como chave primária para conteúdos

---

### 🔄 Status de Processamento

Estados válidos:

* `PENDING`
* `PROCESSING`
* `COMPLETED`
* `ERROR`

---

### ♻️ Resiliência do Worker

O Worker deve:

* Implementar **retentativas com backoff**
* Evitar falhas imediatas
* Enviar tarefas problemáticas para uma **Dead Letter Queue (DLQ)**

---

## ⚡ Objetivo do Projeto

Criar um sistema leve e escalável para:

* Armazenar conhecimento
* Organizar conteúdos consumidos
* Automatizar enriquecimento de dados
* Servir como um "Second Brain" pessoal

---

## 🚧 Roadmap
* [ ] Autenticação de usuários
* [ ] Indexação com Meilisearch
* [ ] Sistema de tags e categorias
* [ ] UI frontend
* [ ] Integração com fontes externas (YouTube, artigos, etc.)

---


