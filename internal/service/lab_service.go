package service

import (
	"context"
	"fmt"
	"lab-devops/internal/domain"
	"log"

	"github.com/google/uuid"
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
	code string,
) (<-chan ExecutionResult, <-chan ExecutionFinalState, string, error) {
	lab, err := s.repo.GetLabByID(ctx, labID)
	if err != nil {
		return nil, nil, "", fmt.Errorf("falha ao buscar workspace para o lab %s: %w", labID, err)
	}

	if lab == nil {
		return nil, nil, "", fmt.Errorf("lab com ID %s não encontrado", labID)
	}

	ws, err := s.repo.GetWorkspaceByLabID(ctx, labID)
	if err != nil {
		return nil, nil, "", fmt.Errorf("falha ao buscar workspace para o lab %s: %w", labID, err)
	}
	if ws == nil {
		return nil, nil, "", fmt.Errorf("workspace para o lab %s não encontrado", labID)
	}

	err = s.repo.UpdateWorkspaceCode(ctx, ws.ID, code)
	if err != nil {
		return nil, nil, "", fmt.Errorf("falha ao atualizar workspace para o lab %s: %w", labID, err)
	}

	execConfig := domain.ExecutionConfig{
		WorkspaceID: ws.ID,
		Code:        code,
		State:       ws.State,
		Type:        domain.ExecutionType(lab.Type),
	}

	logStream, finalState, err := s.executor.Execute(ctx, execConfig)
	if err != nil {
		return nil, nil, "", fmt.Errorf("falha ao executar lab %s: %w", labID, err)
	}

	return logStream, finalState, ws.ID, nil
}

func (s *LabService) ValidateLab(
	ctx context.Context,
	labID string,
) (<-chan ExecutionResult, <-chan ExecutionFinalState, string, error) {
	lab, err := s.repo.GetLabByID(ctx, labID)
	if err != nil {
		return nil, nil, "", fmt.Errorf("falha ao buscar lab para o lab %s: %w", labID, err)
	}

	ws, err := s.repo.GetWorkspaceByLabID(ctx, labID)
	if err != nil {
		return nil, nil, "", fmt.Errorf("falha ao buscar workspace para o lab %s: %w", labID, err)
	}

	if lab.ValidationCode == "" {
		return nil, nil, "", fmt.Errorf("lab %s não possui código de validação", labID)
	}

	execConfig := domain.ExecutionConfig{
		WorkspaceID: ws.ID,
		Code:        lab.ValidationCode,
		State:       ws.State,
		Type:        domain.ExecutionType(lab.Type),
	}

	logStream, finalState, err := s.executor.Execute(ctx, execConfig)
	return logStream, finalState, ws.ID, err
}

func (s *LabService) SaveWorkspaceStatus(ctx context.Context, workspaceId string, status string) error {
	if err := s.repo.UpdateWorkspaceStatus(ctx, workspaceId, status); err != nil {
		return fmt.Errorf("falha ao salvar o status do wokspace %s: %w", workspaceId, err)
	}
	return nil
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

func (s *LabService) CreateLab(
	ctx context.Context,
	title, labType, instructions, initialCode string,
	trackID string,
	labOrder int,
	validationCode string, // NOVO PARAMETRO
) (*domain.Lab, error) {
	if title == "" || labType == "" {
		return nil, fmt.Errorf("titulo e tipo são obrigatórios")
	}

	newLab := &domain.Lab{
		ID:             uuid.New().String(),
		Title:          title,
		Type:           labType,
		Instructions:   instructions,
		InitialCode:    initialCode,
		TrackID:        trackID,
		LabOrder:       labOrder,
		ValidationCode: validationCode,
	}

	if err := s.repo.CreateLab(ctx, newLab); err != nil {
		return nil, fmt.Errorf("falha ao criar lab: %w", err)
	}

	return newLab, nil
}

func (s *LabService) ListLabs(ctx context.Context) ([]*domain.Lab, error) {
	labs, err := s.repo.ListLabs(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao listar labs: %w", err)
	}
	return labs, nil
}

func (s *LabService) GetWorkspaceState(ctx context.Context, workspaceID string) ([]byte, error) {
	state, err := s.repo.GetWorkspaceState(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter estado do workspace %s: %w", workspaceID, err)
	}
	return state, nil
}

func (s *LabService) CleanLab(ctx context.Context, labId string) error {
	err := s.repo.CleanLab(ctx, labId)
	if err != nil {
		return fmt.Errorf("erro ao apagar laboratório: %w", err)
	}
	return err
}

func (s *LabService) CreateTrack(ctx context.Context, title, description string) (*domain.Track, error) {
	if title == "" {
		return nil, fmt.Errorf("titulo é obrigatório")
	}

	newTrack := &domain.Track{
		ID:          uuid.New().String(),
		Title:       title,
		Description: description,
	}

	if err := s.repo.CreateTrack(ctx, newTrack); err != nil {
		return nil, fmt.Errorf("falha ao criar trilha: %w", err)
	}

	return newTrack, nil
}

// ListTracks lista todas as trilhas e seus labs
func (s *LabService) ListTracks(ctx context.Context) ([]*domain.Track, error) {
	tracks, err := s.repo.ListTracks(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao listar trilhas: %w", err)
	}

	for _, track := range tracks {
		labs, err := s.repo.ListLabsByTrackID(ctx, track.ID)
		if err != nil {
			log.Printf("AVISO: Falha ao carregar labs para trilha %s: %v", track.ID, err)
			continue
		}
		track.Labs = labs
	}

	return tracks, nil
}
