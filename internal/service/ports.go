package service

import (
	"context"
	"lab-devops/internal/domain"
)

type ExecutionResult struct {
	Line string
	Err  error
}

type ExecutionFinalState struct {
	WorkspaceID      string
	NewState         []byte
	Error            error
	ExecutionResult  domain.StepResult
	ValidationResult domain.StepResult
}

type Executor interface {
	Execute(ctx context.Context, config domain.ExecutionConfig) (<-chan ExecutionResult, <-chan ExecutionFinalState, error)
}

type WorkspaceRepository interface {
	GetLabByID(ctx context.Context, labID string) (*domain.Lab, error)
	ListLabs(ctx context.Context) ([]*domain.Lab, error)
	GetWorkspaceByLabID(ctx context.Context, labID string) (*domain.Workspace, error)
	UpdateWorkspaceCode(ctx context.Context, workspaceId string, code string) error
	UpdateWorkspaceState(ctx context.Context, workspaceId string, state []byte) error
	GetWorkspaceState(ctx context.Context, workspaceId string) ([]byte, error)
	CreateWorkspace(ctx context.Context, labId string) (*domain.Workspace, error)
	CreateLab(ctx context.Context, lab *domain.Lab) error
	CleanLab(ctx context.Context, labId string) error
	UpdateWorkspaceStatus(ctx context.Context, workspaceId string, status string) error

	ListTracks(ctx context.Context) ([]*domain.Track, error)
	ListLabsByTrackID(ctx context.Context, trackID string) ([]*domain.Lab, error)
	CreateTrack(ctx context.Context, track *domain.Track) error
	UpdateLab(ctx context.Context, lab *domain.Lab) error
	DeleteLab(ctx context.Context, labID string) error
	UpdateTrack(ctx context.Context, track *domain.Track) error
	DeleteTrack(ctx context.Context, trackID string) error
	GetTrackByID(ctx context.Context, id string) (*domain.Track, error)
	Ping(ctx context.Context) error
}
