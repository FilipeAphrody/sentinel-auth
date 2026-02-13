# Atualizado para golang:1.25-alpine para corresponder ao go.mod
FROM golang:1.25-alpine AS builder

# Instalar dependências de build (git é necessário para baixar módulos)
RUN apk add --no-cache git

WORKDIR /app

# Copiar arquivos de dependência
COPY go.mod go.sum ./

# Baixar dependências
RUN go mod download

# Copiar o código fonte
COPY . .

# Compilar o binário
# -ldflags="-s -w" reduz o tamanho do binário removendo informações de debug
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o sentinel cmd/api/main.go

# --- Estágio Final (Imagem leve) ---
FROM alpine:latest

# Instalar certificados CA para chamadas HTTPS externas
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copiar o binário do estágio de build
COPY --from=builder /app/sentinel .

# Expor a porta da API
EXPOSE 8080

# Comando de inicialização
CMD ["./sentinel"]