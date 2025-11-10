package domain

type ExecutionType string 

const (
	TypeTerraform ExecutionType = "terraform"
	TypeAnsible   ExecutionType = "ansible"
)

type ExecutionConfig struct {
	WorkspaceID string
	Code string
	State []byte
	Type ExecutionType
}