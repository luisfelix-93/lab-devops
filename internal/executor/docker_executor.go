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
	"path/filepath"
	"strings"
	"sync"
	"time"

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
	hostExecPath  string
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

		// Aguardar sincronização do filesystem (Docker Desktop WSL2)
		// O prepareWorkspace escreve ficheiros via bind mount, mas o Docker daemon
		// pode não ver os ficheiros imediatamente devido ao delay de sync do WSL2.
		time.Sleep(1 * time.Second)

		// 2. Iniciar Container (Session Manager)
		containerID, err := e.startContainer(ctx, config)
		if err != nil {
			reportError(config.WorkspaceID, fmt.Errorf("falha ao iniciar container: %w", err), finalState)
			return
		}
		defer e.stopContainer(context.Background(), containerID)

		// 3. Executar Código do Usuário
		logStream <- service.ExecutionResult{Line: "--- INICIANDO EXECUÇÃO ---"}
		execCmd, execEnv := e.getStepCommand(config, false)

		// Pequeno sleep para garantir que container está pronto (workaround para race conditions)
		time.Sleep(500 * time.Millisecond)

		execResult := e.execStep(ctx, containerID, execCmd, execEnv, "/workspace", logStream)

		var validationResult domain.StepResult

		// 4. Executar Validação (Se necessário e se execução passou)
		if execResult.ExitCode == 0 && config.ValidationCode != "" {
			logStream <- service.ExecutionResult{Line: "\n--- INICIANDO VALIDAÇÃO ---"}
			valCmd, valEnv := e.getStepCommand(config, true)

			// Se for Kubernetes, usa lógica de retry
			if config.Type == domain.TypeK8s {
				validationResult = e.runWithRetry(ctx, containerID, valCmd, valEnv, "/workspace", logStream)
			} else {
				validationResult = e.execStep(ctx, containerID, valCmd, valEnv, "/workspace", logStream)
			}
		}

		// 5. Ler Estado Final (Terraform)
		newState, readErr := e.readFinalState(execDir, config)
		var finalErr error
		if execResult.ExitCode != 0 {
			finalErr = fmt.Errorf("execução falhou com código %d", execResult.ExitCode)
		} else if readErr != nil {
			finalErr = readErr
		}

		finalState <- service.ExecutionFinalState{
			WorkspaceID:      config.WorkspaceID,
			NewState:         newState,
			Error:            finalErr,
			ExecutionResult:  execResult,
			ValidationResult: validationResult,
		}
	}()

	return logStream, finalState, nil
}

func (e *dockerExecutor) startContainer(ctx context.Context, config domain.ExecutionConfig) (string, error) {
	hostDir := filepath.Join(e.hostExecPath, config.WorkspaceID)
	mounts := []mount.Mount{
		{Type: mount.TypeBind, Source: hostDir, Target: "/workspace"},
	}

	var img string
	switch config.Type {
	case domain.TypeTerraform:
		img = "hashicorp/terraform:latest"
	case domain.TypeAnsible:
		img = "cytopia/ansible:latest"
	case domain.TypeLinux:
		img = "alpine:latest"
	case domain.TypeDocker:
		img = "docker:cli"
		mounts = append(mounts, mount.Mount{Type: mount.TypeBind, Source: "/var/run/docker.sock", Target: "/var/run/docker.sock"})
	case domain.TypeK8s:
		img = "bitnami/kubectl:latest"
	case domain.TypeGithubActions:
		img = "docker:cli"
		mounts = append(mounts, mount.Mount{Type: mount.TypeBind, Source: "/var/run/docker.sock", Target: "/var/run/docker.sock"})
	default:
		return "", fmt.Errorf("tipo não suportado: %s", config.Type)
	}

	containerConfig := &container.Config{
		Image:      img,
		Entrypoint: []string{"tail", "-f", "/dev/null"},
		WorkingDir: "/workspace",
		Tty:        true,
	}

	hostConfig := &container.HostConfig{
		Mounts:     mounts,
		AutoRemove: false,
	}

	netConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			e.dockerNetwork: {},
		},
	}

	// Retry loop: Docker Desktop WSL2 pode demorar a sincronizar o filesystem
	// entre o bind mount do container da API e o host. Tentamos 3x com delay.
	maxRetries := 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		resp, err := e.cli.ContainerCreate(ctx, containerConfig, hostConfig, netConfig, nil, "")
		if client.IsErrNotFound(err) {
			log.Printf("INFO [Executor]: Imagem %s não encontrada. Tentando pull...", img)
			reader, pullErr := e.cli.ImagePull(ctx, img, image.PullOptions{})
			if pullErr != nil {
				return "", fmt.Errorf("falha ao baixar imagem: %w", pullErr)
			}
			io.Copy(io.Discard, reader)
			reader.Close()
			resp, err = e.cli.ContainerCreate(ctx, containerConfig, hostConfig, netConfig, nil, "")
		}

		if err != nil {
			lastErr = err
			if attempt < maxRetries {
				log.Printf("AVISO [Executor]: Tentativa %d/%d falhou ao criar container: %v. Aguardando sync...", attempt, maxRetries, err)
				time.Sleep(time.Duration(attempt) * 1500 * time.Millisecond)
				continue
			}
			break
		}

		if err := e.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
			// Se falhou ao iniciar, remove o container criado antes de tentar novamente
			e.cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
			lastErr = err
			if attempt < maxRetries {
				log.Printf("AVISO [Executor]: Tentativa %d/%d falhou ao iniciar container: %v. Aguardando sync...", attempt, maxRetries, err)
				time.Sleep(time.Duration(attempt) * 1500 * time.Millisecond)
				continue
			}
			break
		}

		log.Printf("INFO [Executor]: Container %s iniciado com sucesso (tentativa %d)", resp.ID[:12], attempt)
		return resp.ID, nil
	}

	return "", fmt.Errorf("falha após %d tentativas: %w", maxRetries, lastErr)
}

func (e *dockerExecutor) stopContainer(ctx context.Context, containerID string) {
	removeOpts := container.RemoveOptions{Force: true}
	if err := e.cli.ContainerRemove(ctx, containerID, removeOpts); err != nil {
		log.Printf("ERRO [Executor]: Falha ao remover container %s: %v", containerID, err)
	}
}

func (e *dockerExecutor) getStepCommand(config domain.ExecutionConfig, isValidation bool) ([]string, []string) {
	var cmd []string
	var env []string

	if isValidation {
		switch config.Type {
		case domain.TypeAnsible:
			return []string{"ansible-playbook", "-i", "inventory.ini", "validation.yml"}, nil
		case domain.TypeK8s:
			if config.ValidationCode != "" {
				return []string{"sh", "validation.sh"}, []string{"KUBECONFIG=/workspace/kubeconfig.yaml"}
			}
		case domain.TypeLinux:
			// Para Linux, se houver código de validação, assumimos que foi escrito em validation.sh (ainda não implementado no prepareWorkspace para TypeLinux, mas podemos adicionar)
			// Por enquanto, retorna echo
			return []string{"echo", "validation not implemented for linux"}, nil
		}
		return []string{"echo", "validation not implemented"}, nil
	}

	switch config.Type {
	case domain.TypeTerraform:
		cmd = []string{"sh", "-c", "mkdir -p /tmp/plugins && rm -rf .terraform/ && terraform init -upgrade && terraform apply -auto-approve"}
		env = []string{"TF_PLUGIN_CACHE_DIR=/tmp/plugins"}
	case domain.TypeAnsible:
		cmd = []string{"ansible-playbook", "-i", "inventory.ini", "playbook.yml"}
	case domain.TypeLinux, domain.TypeDocker:
		cmd = []string{"sh", "run.sh"}
	case domain.TypeK8s:
		cmd = []string{"sh", "run.sh"}
		env = []string{"KUBECONFIG=/workspace/kubeconfig.yaml"}
	case domain.TypeGithubActions:
		cmd = []string{"sh", "-c", "apk add --no-cache act --repository=http://dl-cdn.alpinelinux.org/alpine/edge/community && act push --bind --directory /workspace -P ubuntu-latest=node:18-buster-slim --container-architecture linux/amd64"}
	}
	return cmd, env
}

func (e *dockerExecutor) execStep(ctx context.Context, containerID string, cmd []string, env []string, workDir string, logStream chan<- service.ExecutionResult) domain.StepResult {
	execConfig := container.ExecOptions{
		Cmd:          cmd,
		Env:          env,
		WorkingDir:   workDir,
		AttachStdout: true,
		AttachStderr: true,
	}

	execIDResp, err := e.cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return domain.StepResult{Error: err}
	}

	resp, err := e.cli.ContainerExecAttach(ctx, execIDResp.ID, container.ExecStartOptions{})
	if err != nil {
		return domain.StepResult{Error: err}
	}
	defer resp.Close()

	var outBuf strings.Builder

	// Create a WaitGroup to ensure we finish reading logs before returning
	var wg sync.WaitGroup
	wg.Add(1)

	// Use a pipe to split the stream into lines
	rd, wr := io.Pipe()
	go func() {
		defer wg.Done()
		defer wr.Close()
		// StdCopy demultiplexes Docker stream
		_, _ = stdcopy.StdCopy(wr, wr, resp.Reader)
	}()

	// Read lines
	scanner := bufio.NewScanner(rd)
	for scanner.Scan() {
		line := scanner.Text()
		logStream <- service.ExecutionResult{Line: line}
		outBuf.WriteString(line + "\n")
	}

	wg.Wait()

	inspectResp, err := e.cli.ContainerExecInspect(ctx, execIDResp.ID)
	exitCode := 0
	if err == nil {
		exitCode = inspectResp.ExitCode
	}

	return domain.StepResult{
		ExitCode: exitCode,
		Output:   outBuf.String(),
		Error:    err,
	}
}

func (e *dockerExecutor) runWithRetry(ctx context.Context, containerID string, cmd []string, env []string, workDir string, logStream chan<- service.ExecutionResult) domain.StepResult {
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return domain.StepResult{Error: ctx.Err()}
		case <-timeout:
			return domain.StepResult{ExitCode: 1, Error: fmt.Errorf("timeout na validação"), Output: "Timeout aguardando recurso Kubernetes"}
		case <-ticker.C:
			logStream <- service.ExecutionResult{Line: " [K8s] Validando recursos..."}
			res := e.execStep(ctx, containerID, cmd, env, workDir, logStream)
			if res.ExitCode == 0 {
				return res
			}
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

	switch config.Type {
	case domain.TypeTerraform:
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
		if err := os.WriteFile(filepath.Join(execDir, "run.sh"), []byte(cleanCode), 0755); err != nil {
			return "", err
		}
	case domain.TypeK8s:
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

		if config.ValidationCode != "" {
			cleanValidation := strings.ReplaceAll(config.ValidationCode, "\r\n", "\n")
			if err := os.WriteFile(filepath.Join(execDir, "validation.sh"), []byte(cleanValidation), 0755); err != nil {
				return "", err
			}
		}

	case domain.TypeGithubActions:
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

func (e *dockerExecutor) readFinalState(execDir string, config domain.ExecutionConfig) ([]byte, error) {
	if config.Type != domain.TypeTerraform {
		return nil, nil // Ansible não tem estado
	}

	statePath := filepath.Join(execDir, "terraform.tfstate")

	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		log.Printf("AVISO [Executor]: Arquivo .tfstate não encontrado após execução: %s", statePath)
		return nil, nil
	}

	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil, fmt.Errorf("falha ao ler arquivo .tfstate final: %w", err)
	}
	return data, nil
}
