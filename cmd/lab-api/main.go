package main

import (
	"log"
	"os"
	"lab-devops/internal/api"
	"lab-devops/internal/executor"
	"lab-devops/internal/repository"
	"lab-devops/internal/service"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func main() {
	// Configurações via Variáveis de Ambiente
	sqliteDBPath := getEnv("DB_PATH", "./data/lab.db")
	migrationsPath := getEnv("MIGRATIONS_PATH", "./db/migrations/001_init_schema.sql")
	dockerNetwork := getEnv("DOCKER_NETWORK", "minha-rede-lab")
	tempDirRoot := getEnv("TEMP_DIR_ROOT", "/app/data/temp-exec")
	serverPort := getEnv("SERVER_PORT", ":8080")

	// 1. Camada de Infraestrutura (Implementações)
	repo, err := repository.NewSQLiteRepository(sqliteDBPath, migrationsPath)
	if err != nil {
		log.Fatalf("Falha ao iniciar o repositório SQLite: %v", err)
	}

	exec, err := executor.NewDockerExecutor(dockerNetwork, tempDirRoot)
	if err != nil {
		log.Fatalf("Falha ao iniciar o Docker executor: %v", err)
	}

	// 2. Camada de Lógica de Negócios (Serviço)
	// (Injeta as implementações nas interfaces)
	labSvc := service.NewLabService(repo, exec)

	// 3. Camada de Apresentação (API/Handlers)
	handler := api.NewHandler(labSvc)

	// 4. Configuração do Servidor Web (Echo)
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	
	// Regista as rotas
	api.RegisterRoutes(e, handler)
	
	log.Printf("🚀 Servidor da API do Laboratório rodando na porta %s", serverPort)
	if err := e.Start(serverPort); err != nil {
		log.Fatalf("Falha ao iniciar o servidor Echo: %v", err)
	}
}
