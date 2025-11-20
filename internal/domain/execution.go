package domain

type ExecutionType string 

const (
	TypeTerraform ExecutionType = "terraform"
	TypeAnsible   ExecutionType = "ansible"
	TypeLinux     ExecutionType = "linux"
)

type ExecutionConfig struct {
	WorkspaceID string
	Code string
	State []byte
	Type ExecutionType
}