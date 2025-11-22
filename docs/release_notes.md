# Notas de Lançamento

## [v3.0.0] - 2025-11-22

### 🚀 Novas Funcionalidades
- **Suporte a Kubernetes**: Adicionado suporte completo para execução de laboratórios Kubernetes usando um cluster K3s local.
  - Integrado serviço `rancher/k3s` no Docker Compose.
  - Gerenciamento automático do `kubeconfig` para execução isolada.
  - Suporte para comandos `kubectl` nos laboratórios.
- **Pipelines de CI/CD**:
  - **Auto-PR**: Criação automática de Pull Requests para branches de feature usando GitHub Actions.
  - **Build Docker**: Build e push automatizados de imagens Docker para o Docker Hub ao realizar merge na `main`.

### 🐛 Correções de Bugs
- Corrigido um erro de digitação crítico (`filePath` -> `filepath`) em `docker_executor.go` que impedia a execução correta de laboratórios Kubernetes.

### 🛠 Melhorias
- Adicionado diretório `data/` ao `.gitignore` para evitar o commit de arquivos temporários de execução e dados do K3s.
- Melhoria nos logs do executor Docker para distinguir melhor entre os tipos de execução Linux e Docker.
