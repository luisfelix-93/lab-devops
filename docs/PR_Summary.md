# PR Summary: Implementação de Laboratório Docker/K8s e CI/CD

## Visão Geral
Este Pull Request implementa funcionalidades essenciais para o suporte a laboratórios baseados em Kubernetes (K3s) e Docker, além de estabelecer pipelines de CI/CD automatizados utilizando GitHub Actions.

## Alterações Principais

### 1. Suporte a Kubernetes (K3s)
- **Docker Compose**: Adicionado serviço `k3s` utilizando a imagem `rancher/k3s:latest`.
  - Configurado para expor a porta `6443`.
  - Mapeamento de volumes para persistência de dados e exportação do `kubeconfig`.
- **Executor (Go)**:
  - Correção de bugs no `docker_executor.go` (typo `filePath` -> `filepath`).
  - Implementada lógica para preparar o ambiente K8s:
    - Leitura automática do `kubeconfig` gerado pelo K3s.
    - Ajuste dinâmico do endereço do cluster (substituindo `localhost`/`127.0.0.1` por `k3s` para comunicação interna na rede Docker).
    - Escrita do `kubeconfig.yaml` no workspace de execução.

### 2. Automação CI/CD (GitHub Actions)
- **Auto PR (`auto-pr.yml`)**:
  - Workflow acionado em pushs (exceto na main).
  - Cria automaticamente um Pull Request para a branch `main`.
  - Utiliza este arquivo (`docs/PR_SUMMARY.md`) como corpo do PR.
  - Gera títulos dinâmicos com a data atual (YYYYmmDD).
- **Docker Build (`docker.yml`)**:
  - Workflow acionado ao fechar um PR na `main` (merge).
  - Realiza build e push da imagem Docker para o Docker Hub.
  - Tags da imagem: `latest` e `YYYYmmDD`.

### 3. Outras Melhorias
- **Gitignore**: Adicionado diretório `data/` para ignorar arquivos temporários e de dados do K3s.
- **Logs**: Melhoria nas mensagens de log do executor para identificar tipos de execução Linux/Docker.

## Como Testar
1. Subir o ambiente com `docker-compose up -d`.
2. Verificar se o container `k3s` está saudável.
3. Realizar um push para uma branch de feature e verificar a criação automática do PR.
4. Realizar o merge do PR e verificar o disparo do workflow de build Docker.
