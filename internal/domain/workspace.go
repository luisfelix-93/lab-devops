package domain

import "time"

const (
	WorkspaceStatusInProgress = "in_progress"
	WorkspaceStatusCompleted  = "completed"
)

type Workspace struct {
	ID        string 	`json:"id"`
	LabID     string 	`json:"lab_id"`
	UserCode  string 	`json:"user_code"`
	State     []byte 	`json:"state"`
	UpdatedAt time.Time `json:"updated_at"`

	Status    string    `json:"status"`
}	