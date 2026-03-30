# --- Estágio de Build ---
FROM golang:1.25.0-alpine AS builder

# Instala certificados para chamadas HTTPS (necessário para o Scraper)
RUN apk add --no-cache ca-certificates git

WORKDIR /app

# Cache de dependências
COPY go.mod go.sum ./
RUN go mod download

# Copia o código fonte
COPY . .

# Compila os dois binários de forma estática
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/api ./cmd/api/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/worker ./cmd/worker/main.go

# --- Estágio Final (Runtime) ---
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata
WORKDIR /root/

# Copia os binários do builder
COPY --from=builder /app/bin/api .
COPY --from=builder /app/bin/worker .

# Criar pasta de thumbnails para persistência
RUN mkdir -p uploads/thumbnails

# A porta que a API expõe
EXPOSE 8080

# Por padrão, inicia a API (sobrescrevemos isso no compose para o worker)
CMD ["./api"]