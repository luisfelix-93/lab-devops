package service

import (
	"context"
	"fmt"
	"lab-devops/internal/domain"
)

type LabService struct {
	repo     WorkspaceRepository
	executor Executor
}

func NewLabService(repo WorkspaceRepository, executor Executor) *LabService {
	return &LabService{
		repo:     repo,
		executor: executor,
	}
}

func (s *LabService) ExecuteLab(
	ctx context.Context,
	labID string,
	code  string,
) (<-chan ExecutionResult, <-chan ExecutionFinalState, error) {
	lab, err := s.repo.GetLabByID(ctx, labID)
	if err != nil {
		return nil, nil, fmt.Errorf("falha ao buscar workspace para o lab %s: %w", labID, err)
	}

	if lab == nil {
		return nil, nil, fmt.Errorf("lab com ID %s não encontrado", labID)
	}

	ws, err := s.repo.GetWorkspaceByLabID(ctx, labID)
	if err != nil {
		return nil, nil, fmt.Errorf("falha ao buscar workspace para o lab %s: %w", labID, err)
	}
	if ws == nil {
		return nil, nil, fmt.Errorf("workspace para o lab %s não encontrado", labID)
	}

	err = s.repo.UpdateWorkspaceCode(ctx, ws.ID, code)
	if err != nil {
		return nil, nil, fmt.Errorf("falha ao atualizar workspace para o lab %s: %w", labID, err)
	}

	execConfig := domain.ExecutionConfig{
		WorkspaceID: ws.ID,
		Code:        code,
		State:       ws.State,
		Type:		 domain.ExecutionType(lab.Type),
	}
	
	logStream, finalState, err := s.executor.Execute(ctx, execConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("falha ao executar lab %s: %w", labID, err)
	}

	return logStream, finalState, nil
}

func (s *LabService) SaveWorkspaceState(ctx context.Context, workspaceID string, state []byte) error {
	if err := s.repo.UpdateWorkspaceState(ctx, workspaceID, state); err != nil {
		return fmt.Errorf("falha ao salvar o estado final do workspace %s: %w", workspaceID, err)
	}
	return nil
}

func (s *LabService) GetLabDetails(ctx context.Context, labID string) (*domain.Lab, *domain.Workspace, error) {
	lab, err := s.repo.GetLabByID(ctx, labID)
	if err != nil {
		return nil, nil, err
	}
	if lab == nil {
		return nil, nil, fmt.Errorf("lab não encontrado")
	}

	ws, err := s.repo.GetWorkspaceByLabID(ctx, labID)
	if err != nil {
		return nil, nil, err
	}
	if ws == nil {
		ws, err = s.repo.CreateWorkspace(ctx, labID)
		if err != nil {
			return nil, nil, fmt.Errorf("falha ao criar workspace: %w", err)
		}
	}

	return lab, ws, nil
}
