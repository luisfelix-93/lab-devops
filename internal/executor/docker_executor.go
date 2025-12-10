package executor

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"lab-devops/internal/domain"
	"lab-devops/internal/service"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"


	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

const defaultLocalstackProviderConfig = `
provider "aws" {
	region                      = "us-east-1"
	access_key                  = "test"
	secret_key                  = "test"
	
	skip_credentials_validation = true
	skip_metadata_api_check     = true
	skip_region_validation      = true 
	s3_use_path_style           = true

	endpoints {
		s3        = "http://simulador-iac:4566"
		s3control = "http://simulador-iac:4566"
		ec2       = "http://simulador-iac:4566"
		lambda    = "http://simulador-iac:4566"
		sqs       = "http://simulador-iac:4566"
		iam       = "http://simulador-iac:4566"
		sts       = "http://simulador-iac:4566" # Importante para evitar erros de validação
		route53   = "http://simulador-iac:4566"
	}
}
`
const ansibleLocalInventory = `
[local]
localhost ansible_connection=local
`

type dockerExecutor struct {
	cli           *client.Client
	dockerNetwork string
	tempDirRoot   string
	hostExecPath  string // <-- NOVO CAMPO
}



func NewDockerExecutor(dockerNetwork string, tempDirRoot string) (service.Executor, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("falha ao criar cliente Docker: %w", err)
	}

	if err := os.MkdirAll(tempDirRoot, 0755); err != nil {
		return nil, fmt.Errorf("falha ao criar diretório temporário raiz %s: %w", tempDirRoot, err)
	}

	hostPath := os.Getenv("HOST_EXEC_PATH")
	if hostPath == "" {
		return nil, fmt.Errorf("variável de ambiente HOST_EXEC_PATH não está definida")
	}	

	return &dockerExecutor{
		cli:           cli,
		dockerNetwork: dockerNetwork,
		tempDirRoot:   tempDirRoot,
		hostExecPath:  hostPath,
	}, nil
}

// Helper: Tenta ler provider.f da pasta data, senão usa o default
func (e *dockerExecutor) getTerraformProvider() []byte {
	configDir := filepath.Join(e.tempDirRoot, "data")
	configPath := filepath.Join(configDir, "terraform-provider.tf")

	content, err := os.ReadFile(configPath)
	if err == nil && len(content) > 0 {
		return content
	}
	return []byte(defaultLocalstackProviderConfig)
}


func (e *dockerExecutor) Execute(ctx context.Context, config domain.ExecutionConfig) (<-chan service.ExecutionResult, <-chan service.ExecutionFinalState, error) {
	logStream := make(chan service.ExecutionResult)
	finalState := make(chan service.ExecutionFinalState)

	go func() {
		defer close(logStream)
		defer close(finalState)

		// 1. Preparar arquivos locais
		execDir, err := e.prepareWorkspace(config)
		if err != nil {
			reportError(config.WorkspaceID, err, finalState)
			return
		}
		defer os.RemoveAll(execDir)

		// 2. Configurar o Container
		containerConfig, hostConfig, netConfig, err := e.getContainerConfig(config)
		if err != nil {
			reportError(config.WorkspaceID, err, finalState)
			return
		}

		// 3. Criar o Container
		resp, err := e.cli.ContainerCreate(ctx, containerConfig, hostConfig, netConfig, nil, "")
		if err != nil {
			// Se a imagem não existir, tenta fazer pull (opcional, mas recomendado)
			if client.IsErrNotFound(err) {
				log.Printf("INFO [Executor]: Imagem %s não encontrada. Tentando pull...", containerConfig.Image)
				_, pullErr := e.cli.ImagePull(ctx, containerConfig.Image, image.PullOptions{})
				if pullErr != nil {
					reportError(config.WorkspaceID, fmt.Errorf("falha ao baixar imagem %s: %w", containerConfig.Image, pullErr), finalState)
					return
				}
				// Tenta criar de novo
				resp, err = e.cli.ContainerCreate(ctx, containerConfig, hostConfig, netConfig, nil, "")
			}
			
			if err != nil {
				reportError(config.WorkspaceID, fmt.Errorf("falha ao criar container: %w", err), finalState)
				return
			}
		}

		containerID := resp.ID
		// Garante a remoção do container ao final
		defer func() {
			removeOpts := container.RemoveOptions{Force: true}
			if err := e.cli.ContainerRemove(context.Background(), containerID, removeOpts); err != nil {
				log.Printf("ERRO [Executor]: Falha ao remover container %s: %v", containerID, err)
			}
		}()

		// 4. Iniciar o Container
		if err := e.cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
			reportError(config.WorkspaceID, fmt.Errorf("falha ao iniciar container: %w", err), finalState)
			return
		}

		// 5. Capturar Logs (Stdout/Stderr)
		out, err := e.cli.ContainerLogs(ctx, containerID, container.LogsOptions{ShowStdout: true, ShowStderr: true, Follow: true})
		if err != nil {
			reportError(config.WorkspaceID, fmt.Errorf("falha ao obter logs: %w", err), finalState)
			return
		}
		
		// Goroutine para processar o stream de logs
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			// StdCopy demultiplexa o stream do Docker em stdout e stderr
			// Aqui jogamos ambos para um PipeReader para processar linha a linha
			rd, wr := io.Pipe()
			
			go func() {
				// Escreve tanto stdout quanto stderr no pipe
				stdcopy.StdCopy(wr, wr, out)
				wr.Close()
			}()

			e.streamLogs(rd, logStream)
		}()

		// 6. Aguardar finalização
		statusCh, errCh := e.cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
		select {
		case err := <-errCh:
			if err != nil {
				reportError(config.WorkspaceID, fmt.Errorf("erro durante execução do container: %w", err), finalState)
				return // Importante retornar para não processar estado final inválido
			}
		case status := <-statusCh:
			wg.Wait() // Espera terminar de processar logs
			
			var execErr error
			if status.StatusCode != 0 {
				execErr = fmt.Errorf("comando finalizou com código de saída: %d", status.StatusCode)
			}

			// 7. Ler estado final (Terraform)
			newState, readErr := e.readFinalState(execDir, config)
			if readErr != nil && execErr == nil {
				execErr = readErr
			}

			finalState <- service.ExecutionFinalState{
				WorkspaceID: config.WorkspaceID,
				NewState:    newState,
				Error:       execErr,
			}
		}
	}()

	return logStream, finalState, nil
}

func (e *dockerExecutor) getContainerConfig(config domain.ExecutionConfig)  (*container.Config, *container.HostConfig, *network.NetworkingConfig, error) {
	hostDir := filepath.Join(e.hostExecPath, config.WorkspaceID)

	mounts := []mount.Mount{
		{
		Type: mount.TypeBind,
		Source: hostDir,
		Target: "/workspace",
		},
	}

	var img string
	var cmd []string
	var env []string
	workingDir := "/workspace"
	entrypoint := []string{"/bin/sh"}
	switch config.Type {
	case domain.TypeTerraform:
		img = "hashicorp/terraform:latest"
		cmd = []string{"-c", "mkdir -p /tmp/plugins && rm -rf .terraform/ && terraform init -upgrade && terraform apply -auto-approve"}
		env = []string{"TF_PLUGIN_CACHE_DIR=/tmp/plugins"}
	
	case domain.TypeAnsible:
		img = "cytopia/ansible:latest"
		ansibleCmd := "ansible-playbook -i inventory.ini playbook.yml"
		if config.ValidationCode != "" {
			ansibleCmd += " && echo '--- INICIANDO VALIDAÇÃO ---' && ansible-playbook -i inventory.ini validation.yml"
		}
		cmd = []string{"-c", ansibleCmd}

	case domain.TypeLinux:
		img = "alpine:latest"
		cmd = []string{"run.sh"}

	case domain.TypeDocker:
		img = "docker:cli"
		cmd = []string{"run.sh"}
		// Adiciona o socket do Docker
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: "/var/run/docker.sock",
			Target: "/var/run/docker.sock",
		})

	case domain.TypeK8s:
		img = "bitnami/kubectl:latest"
		cmd = []string{"run.sh"}
		env = []string{"KUBECONFIG=/workspace/kubeconfig.yaml"}
	case domain.TypeGithubActions:
		img = "docker:cli"
		entrypoint = nil
		cmd = []string{"/bin/sh", "-c", 
			"apk add --no-cache act --repository=http://dl-cdn.alpinelinux.org/alpine/edge/community && " + 
			"act push " +
			"--bind " + 
			"--directory /workspace " +
			"-P ubuntu-latest=node:18-buster-slim " + 
			"--container-architecture linux/amd64"}

		// Precisamos do socket para o act criar os containers irmãos
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: "/var/run/docker.sock",
			Target: "/var/run/docker.sock",
		})
	default:
		return nil, nil, nil, fmt.Errorf("tipo de execução '%s' não suportado", config.Type)
	}

	containerConfig := &container.Config{
		Image:      img,
		Cmd:        cmd,
		Env:        env,
		WorkingDir: workingDir,
		Entrypoint: entrypoint, // Usa o da imagem ou sobrescreve se necessário
	}

	hostConfig := &container.HostConfig{
		Mounts:      mounts,
		AutoRemove:  false, // Controlamos a remoção manualmente no defer
	}

	netConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			e.dockerNetwork: {},
		},
	}

	return containerConfig, hostConfig, netConfig, nil
}

func (e *dockerExecutor) streamLogs(reader io.Reader, logStream chan<- service.ExecutionResult) {
	buf := make([]byte, 1024)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			lines := strings.Split(string(buf[:n]), "\n")
			for _, line := range lines {
				logStream <- service.ExecutionResult{Line: line}
			}
		}
		if err != nil {
			break
		}
	}
}



func reportError(wsID string, err error, ch chan<- service.ExecutionFinalState) {
	log.Printf("ERRO [Executor]: %v", err)
	ch <- service.ExecutionFinalState{WorkspaceID: wsID, Error: err}
}


func (e *dockerExecutor) prepareWorkspace(config domain.ExecutionConfig) (string, error) {
	execDir := filepath.Join(e.tempDirRoot, config.WorkspaceID)

	if err := os.RemoveAll(execDir); err != nil {
		return "", err
	}
	if err := os.MkdirAll(execDir, 0755); err != nil {
		return "", err
	}

	cleanCode := strings.ReplaceAll(config.Code, "\r\n", "\n")

	log.Printf("DEBUG [Executor]: a preparar workspace. Tipo recebido: '%s'", config.Type)
	log.Printf("DEBUG [Executor]: a comparar com: '%s'", domain.TypeTerraform)

	switch config.Type {
	case domain.TypeTerraform:
		log.Printf("DEBUG [Executor]: 'if' deu VERDADEIRO. A escrever ficheiros Terraform...")
		if err := os.WriteFile(filepath.Join(execDir, "main.tf"), []byte(cleanCode), 0644); err != nil {
			return "", err
		}
		providerConfig := e.getTerraformProvider()
		if err := os.WriteFile(filepath.Join(execDir, "provider.tf"), providerConfig, 0644); err != nil {
			return "", err
		}
		if err := os.WriteFile(filepath.Join(execDir, "terraform.tfstate"), config.State, 0644); err != nil {
			return "", err
		}
	case domain.TypeAnsible:
		log.Printf("DEBUG [Executor]: A escrever ficheiros Ansible...")
		if err := os.WriteFile(filepath.Join(execDir, "playbook.yml"), []byte(cleanCode), 0644); err != nil {
			return "", err
		}
		if config.ValidationCode != "" {
			cleanValidation := strings.ReplaceAll(config.ValidationCode, "\r\n", "\n")
			if err := os.WriteFile(filepath.Join(execDir, "validation.yml"), []byte(cleanValidation), 0644); err != nil {
				return "", err
			}
		}
		if err := os.WriteFile(filepath.Join(execDir, "inventory.ini"), []byte(ansibleLocalInventory), 0644); err != nil {
			return "", err
		}
	case domain.TypeLinux, domain.TypeDocker:
		log.Printf("DEBUG [Executor]: A escrever ficheiros Linux | Docker ... ")
		if err := os.WriteFile(filepath.Join(execDir, "run.sh"), []byte(cleanCode), 0755); err != nil {
			return "", err
		}
	case domain.TypeK8s:
		log.Printf("DEBUG [Executor]: A preparar ambiente Kubernetes...")

		if err := os.WriteFile(filepath.Join(execDir, "run.sh"), []byte(cleanCode), 0755); err != nil {
			return "", err
		}

		k3sConfifPath := "/app/data/k3s/kubeconfig.yaml"
		content, err := os.ReadFile(k3sConfifPath)
		if err != nil {
			return "", fmt.Errorf("falha ao ler kubeconfig do K3s (o cluster está de pé?): %w", err)
		}

		kcStr := string(content)
		kcStr = strings.Replace(kcStr, "127.0.0.1", "k3s", -1)
		kcStr = strings.Replace(kcStr, "localhost", "k3s", -1)

		if err := os.WriteFile(filepath.Join(execDir, "kubeconfig.yaml"), []byte(kcStr), 0644); err != nil {
			return "", err
		}

	case domain.TypeGithubActions:
		log.Printf("DEBUG [Executor]: A preparar ambiente Github Actions...")
		workflowDir := filepath.Join(execDir, ".github", "workflows")
		if err := os.MkdirAll(workflowDir, 0755); err != nil {
			return "", fmt.Errorf("falha ao criar diretório de workflows: %w", err)
		}
		
		if err := os.WriteFile(filepath.Join(workflowDir, "main.yml"), []byte(cleanCode), 0644); err != nil {
			return "", err
		}
	default:
		os.WriteFile(filepath.Join(execDir, "run.sh"), []byte(cleanCode), 0755)
	}

	return execDir, nil
}

func (e *dockerExecutor) buildCommand(ctx context.Context, execDir string, config domain.ExecutionConfig) (*exec.Cmd, error) {
	hostDir := filepath.Join(e.hostExecPath, config.WorkspaceID)
	switch config.Type {
	case domain.TypeTerraform:
		image := "hashicorp/terraform:latest"
		tfCommand := "mkdir -p /tmp/plugins && rm -rf .terraform/ && terraform init -upgrade && terraform apply -auto-approve"

		args := []string{
			"run", "--rm",
			"--network", e.dockerNetwork,
			"-e", "TF_PLUGIN_CACHE_DIR=/tmp/plugins",
			"-v", fmt.Sprintf("%s:/workspace", hostDir),
			"--entrypoint", "sh",
			"-w", "/workspace",
			image,
			"-c", tfCommand,
		}

		return exec.CommandContext(ctx, "docker", args...), nil
	case domain.TypeAnsible:
		image := "cytopia/ansible:latest"
		ansibleCommand := "ansible-playbook -i inventory.ini playbook.yml"
		if config.ValidationCode != "" {
			ansibleCommand += " && echo '--- INICIANDO VALIDAÇÃO ---' && ansible-playbook -i inventory.ini validation.yml"
		}

		args := []string{
			"run", "--rm",
			// Adicionamos a rede para que o Ansible possa, por exemplo,
			// contactar o 'simulador-iac' (LocalStack) se necessário.
			"--network", e.dockerNetwork,
			"-v", fmt.Sprintf("%s:/workspace", hostDir),
			"--entrypoint", "sh",
			"-w", "/workspace",
			image,
			"-c", ansibleCommand,
		}

		return exec.CommandContext(ctx, "docker", args...), nil

	case domain.TypeLinux:
		image := "alpine:latest"
		linuxCommand := "sh run.sh"

		args := []string{
			"run", "--rm",
			"--network", e.dockerNetwork,
			"-v", fmt.Sprintf("%s:/workspace", hostDir),
			"--entrypoint", "sh",
			"-w", "/workspace",
			image,
			"-c", linuxCommand,
		}

		return exec.CommandContext(ctx, "docker", args...), nil

	case domain.TypeDocker:
		image := "docker:cli"
		dockerCommand := "sh run.sh"

		args := []string{
			"run", "--rm",
			"--network", e.dockerNetwork,
			"-v", fmt.Sprintf("%s:/workspace", hostDir),
			"-v", "/var/run/docker.sock:/var/run/docker.sock",
			"--entrypoint", "sh",
			"-w", "/workspace",
			image,
			"-c", dockerCommand,
		}

		return exec.CommandContext(ctx, "docker", args...), nil

	case domain.TypeK8s:
		image := "bitnami/kubectl:latest"
		k8sCommand := "sh run.sh"

		args := []string{
			"run", "--rm",
			"--network", e.dockerNetwork,
			"-v", fmt.Sprintf("%s:/workspace", hostDir),
			"-e", "KUBECONFIG=/workspace/kubeconfig.yaml",
			"--entrypoint", "sh",
			"-w", "/workspace",
			image,
			"-c", k8sCommand,
		}
		return exec.CommandContext(ctx, "docker", args...), nil
	}

	return nil, fmt.Errorf("tipo de execução desconhecido: %s", config.Type)
}

func (e *dockerExecutor) streamPipe(pipe io.ReadCloser, logstream chan<- service.ExecutionResult, wg *sync.WaitGroup) {
	defer wg.Done()
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()
		logstream <- service.ExecutionResult{Line: line}
	}
}

func (e *dockerExecutor) readFinalState(execDir string, config domain.ExecutionConfig) ([]byte, error) {
	if config.Type != domain.TypeTerraform {
		return nil, nil // Ansible não tem estado
	}

	statePath := filepath.Join(execDir, "terraform.tfstate")

	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		log.Printf("AVISO [Executor]: Arquivo .tfstate não encontrado após execução (pode ser normal se 'apply' falhou): %s", statePath)
		return nil, nil
	}

	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil, fmt.Errorf("falha ao ler arquivo .tfstate final: %w", err)
	}
	return data, nil
}
