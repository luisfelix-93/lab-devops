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
	"sync"
)

const localstackProviderConfig = `
provider "aws" {
	region  					= "us-east-1"
	access_key 					= "dummy"
	secret_key 					= "dummy"
	skip_credentials_validation = true
	skip_metadata_api_check 	= true
	skip_requesting_account_id 	= true
	s3_use_path_style = true

	endpoints {
		s3 		= "http://simulador-iac:4566"
		ec2 	= "http://simulador-iac:4566"
		lambda 	= "http://simulador-iac:4566"
		sqs 	= "http://simulador-iac:4566"
	}
}
`
const ansibleLocalInventory = `
[local]
localhost ansible_connection=local
`

type dockerExecutor struct {
	dockerNetwork string
	tempDirRoot   string
	hostExecPath  string // <-- NOVO CAMPO
}

func NewDockerExecutor(dockerNetwork string, tempDirRoot string) (service.Executor, error) {
	if err := os.MkdirAll(tempDirRoot, 0755); err != nil {
		return nil, fmt.Errorf("falha ao criar diretório temporário raiz %s: %w", tempDirRoot, err)
	}

	hostPath := os.Getenv("HOST_EXEC_PATH")
	if hostPath == "" {
		return nil, fmt.Errorf("variável de ambiente HOST_EXEC_PATH não está definida")
	}

	return &dockerExecutor{
		dockerNetwork: dockerNetwork,
		tempDirRoot:   tempDirRoot,
		hostExecPath:  hostPath,
	}, nil

}
func (e *dockerExecutor) Execute(ctx context.Context, config domain.ExecutionConfig) (<-chan service.ExecutionResult, <-chan service.ExecutionFinalState, error) {
	logStream := make(chan service.ExecutionResult)
	finalState := make(chan service.ExecutionFinalState)

	// example minimal goroutine; real execution logic should send results to logstream
	go func() {
		defer close(logStream)
		defer close(finalState)

		execDir, err := e.prepareWorkspace(config)
		if err != nil {
			log.Printf("ERRO [Executor]: Falha ao preparar workspace: %v", err)
			finalState <- service.ExecutionFinalState{Error: fmt.Errorf("falha ao preparar workspace: %w", err)}
			return
		}

		defer os.RemoveAll(execDir)

		cmd, err := e.buildCommand(ctx, execDir, config)
		if err != nil {
			log.Printf("ERRO [Executor]: Falha ao montar comando: %v", err)
			finalState <- service.ExecutionFinalState{Error: fmt.Errorf("falha ao montar comando: %w", err)}
			return
		}

		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			finalState <- service.ExecutionFinalState{Error: fmt.Errorf("falha ao obter stdout pipe: %w", err)}
			return
		}
		stderrPipe, err := cmd.StderrPipe()
		if err != nil {
			finalState <- service.ExecutionFinalState{Error: fmt.Errorf("falha ao obter stderr pipe: %w", err)}
			return
		}

		var wg sync.WaitGroup
		wg.Add(2) // Um para stdout, um para stderr
		go e.streamPipe(stdoutPipe, logStream, &wg)
		go e.streamPipe(stderrPipe, logStream, &wg)

		log.Printf("INFO [Executor]: Iniciando execução para %s...", config.WorkspaceID)
		if err := cmd.Start(); err != nil {
			finalState <- service.ExecutionFinalState{Error: fmt.Errorf("falha ao iniciar comando: %w", err)}
			return
		}

		execErr := cmd.Wait()
		log.Printf("INFO [Executor]: Execução para %s concluída com erro: %v", config.WorkspaceID, execErr)

		wg.Wait()

		newState, readErr := e.readFinalState(execDir, config)
		if readErr != nil {
			if execErr == nil {
				execErr = readErr
			}
		}

		finalState <- service.ExecutionFinalState{
			NewState: newState,
			Error:    execErr,
		}
		log.Printf("INFO [Executor]: Goroutine para %s finalizada.", config.WorkspaceID)

	}()

	return logStream, finalState, nil
}

func (e *dockerExecutor) prepareWorkspace(config domain.ExecutionConfig) (string, error) {
	execDir := filepath.Join(e.tempDirRoot, config.WorkspaceID)

	if err := os.RemoveAll(execDir); err != nil {
		return "", err
	}
	if err := os.MkdirAll(execDir, 0755); err != nil {
		return "", err
	}

	log.Printf("DEBUG [Executor]: a preparar workspace. Tipo recebido: '%s'", config.Type)
	log.Printf("DEBUG [Executor]: a comparar com: '%s'", domain.TypeTerraform)

	switch config.Type {
	case domain.TypeTerraform:
		log.Printf("DEBUG [Executor]: 'if' deu VERDADEIRO. A escrever ficheiros Terraform...")
		if err := os.WriteFile(filepath.Join(execDir, "main.tf"), []byte(config.Code), 0644); err != nil {
			return "", err
		}
		if err := os.WriteFile(filepath.Join(execDir, "provider.tf"), []byte(localstackProviderConfig), 0644); err != nil {
			return "", err
		}
		if err := os.WriteFile(filepath.Join(execDir, "terraform.tfState"), config.State, 0644); err != nil {
			return "", err
		}
	case domain.TypeAnsible:
		log.Printf("DEBUG [Executor]: A escrever ficheiros Ansible...")
		if err := os.WriteFile(filepath.Join(execDir, "playbook.yml"), []byte(config.Code), 0644); err != nil {
			return "", err
		}
		if err := os.WriteFile(filepath.Join(execDir, "inventory.ini"), []byte(ansibleLocalInventory), 0644); err != nil {
			return "", err
		}
	}

	return execDir, nil
}

func (e *dockerExecutor) buildCommand(ctx context.Context, execDir string, config domain.ExecutionConfig) (*exec.Cmd, error) {
	hostDir := filepath.Join(e.hostExecPath, config.WorkspaceID)
	switch config.Type {
	case domain.TypeTerraform:
		image := "hashicorp/terraform:latest"
		tfCommand := "terraform init && terraform apply -auto-approve"
		
		
		
		args := []string{
			"run", "--rm",
			"--network", e.dockerNetwork,
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
