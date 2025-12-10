package domain

type ExecutionType string 

const (
	TypeTerraform 		ExecutionType = "terraform"
	TypeAnsible   		ExecutionType = "ansible"
	TypeLinux     		ExecutionType = "linux"
	TypeDocker    		ExecutionType = "docker"
	TypeK8s       		ExecutionType = "kubernetes"
	TypeGithubActions 	ExecutionType = "github-actions"
)

type ExecutionConfig struct {
	WorkspaceID string
	Code string
	State []byte
	ValidationCode string
	Type ExecutionType
}