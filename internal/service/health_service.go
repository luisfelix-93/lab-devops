package service

import (
	"context"
	"os"
	"time"
)

type HealthStatus string

const (
	StatusOK          HealthStatus = "ok"
	StatusDegraded    HealthStatus = "degraded"
	StatusUnavailable HealthStatus = "unavailable"
)

type HealthCheckResponse struct {
	Status    HealthStatus      `json:"status"`
	Checks    map[string]string `json:"checks"`
	Timestamp time.Time         `json:"timestamp"`
}

type HealthService struct {
	repo WorkspaceRepository
}

func NewHealthService(repo WorkspaceRepository) *HealthService {
	return &HealthService{
		repo: repo,
	}
}

func (s *HealthService) CheckHealth(ctx context.Context) HealthCheckResponse {
	checks := make(map[string]string)
	aggregatedStatus := StatusOK

	// 1. Database Check
	if err := s.repo.Ping(ctx); err != nil {
		checks["database"] = "error: " + err.Error()
		aggregatedStatus = StatusUnavailable // DB is critical
	} else {
		checks["database"] = "ok"
	}

	// 2. Disk Check (Simplificado: verifica se consegue escrever no /tmp ou diretório atual)
	// Para verificar espaço em disco real de forma cross-platform sem cgo seria necessário libs extras.
	// Aqui vamos verificar se o diretório de trabalho é gravável como proxy de "disco ok".
    // TODO: Implementar verificação de espaço livre real quando possível.
	if err := s.checkDiskWritable(); err != nil {
		checks["disk"] = "error: " + err.Error()
		if aggregatedStatus == StatusOK {
			aggregatedStatus = StatusDegraded
		}
	} else {
		checks["disk"] = "ok"
	}

	return HealthCheckResponse{
		Status:    aggregatedStatus,
		Checks:    checks,
		Timestamp: time.Now(),
	}
}

func (s *HealthService) checkDiskWritable() error {
	// Tenta criar um arquivo temporário para verificar se o disco está gravável
	f, err := os.CreateTemp("", "healthcheck")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	return f.Close()
}
