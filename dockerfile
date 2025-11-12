# 
# STAGE 1: O Construtor (Builder)
# 
# Usamos a imagem oficial do Go baseada no Alpine Linux
FROM golang:1.24-alpine AS builder

# 1. Instalar as ferramentas de compilação C (gcc)
# O "build-base" é um pacote do Alpine que inclui o gcc e outras ferramentas
RUN apk add --no-cache build-base gcc

# 2. Definir a variável de ambiente para habilitar o CGO
ENV CGO_ENABLED=1

# 3. Criar um diretório de trabalho
WORKDIR /app

# 4. Copiar os arquivos de módulo e baixar dependências
COPY go.mod go.sum ./
RUN go mod download

# 5. Copiar TODO o resto do código-fonte
COPY . .

# 6. Construir o binário
# -o /app/lab-api : Salva o binário compilado como 'lab-api'
# -ldflags="-s -w" : Deixa o binário menor (remove símbolos de debug)
# ./cmd/lab-api/main.go : O ponto de entrada
RUN go build -o /app/lab-api -ldflags="-s -w" ./cmd/lab-api/main.go

# 
# STAGE 2: A Imagem Final (Final)
# 
# Começamos do zero, com uma imagem Alpine limpa e minúscula
FROM alpine:latest

# 1. Instalar as dependências de RUNTIME
# A nossa API precisa de DUAS coisas para rodar:
#   a) O socket do Docker (que vamos montar)
#   b) O binário 'docker-cli' para o nosso executor (os/exec) chamar
RUN apk add --no-cache docker-cli

# 2. Criar um diretório de trabalho
WORKDIR /app

# 3. Copiar o binário construído no Stage 1
COPY --from=builder /app/lab-api /app/lab-api

# 4. Copiar os arquivos de migração (o binário precisa deles)
COPY ./db/migrations /app/db/migrations

# 5. Criar os diretórios de dados que a app espera
RUN mkdir -p /app/data/temp-exec

# 6. Expor a porta (documentação)
EXPOSE 8080

# 7. O comando para rodar a aplicação
CMD ["./lab-api"]