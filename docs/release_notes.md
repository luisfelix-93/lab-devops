# Release Notes - 0.4.20251208.1

## 🌟 Destaques

### Validação Automática de Laboratórios
A experiência de execução de laboratórios foi significativamente melhorada. Agora, ao submeter uma solução (`execute`), o sistema não apenas roda o código, mas também **inicia automaticamente a validação** caso a execução seja bem-sucedida.

- **Feedback Imediato**: Receba emojis (✅/❌) em tempo real indicando o progresso da execução e da validação.
- **Fluxo Simplificado**: Não é mais necessário clicar em "Validar" separadamente após uma execução bem-sucedida.

### Configuração via Variáveis de Ambiente
A aplicação agora é totalmente configurável via variáveis de ambiente, seguindo os princípios 12-Factor App, facilitando o deploy em diferentes ambientes (dev, staging, prod).

## 🚀 Melhorias e Alterações

### Backend & API
- **Validação Encadeada**: O endpoint WebSocket de execução (`HandlerLabExecute`) foi refatorado para disparar a validação automaticamente após o sucesso da execução do usuário.
- **Porta da API**: A porta padrão foi alterada para `8081` no docker-compose para evitar conflitos comuns com outros serviços na porta 8080.
- **Ansible Executor**: Suporte a validação integrada para laboratórios Ansible (`ansible-playbook validation.yml`).

### Infraestrutura
- **Configuração Dinâmica**: Novas variáveis de ambiente suportadas:
    - `DB_PATH`
    - `MIGRATIONS_PATH`
    - `DOCKER_NETWORK`
    - `TEMP_DIR_ROOT`
    - `SERVER_PORT`
- **Docker Compose**: O serviço `iam` foi removido da inicialização padrão do LocalStack para otimizar recursos.

## 🐛 Correções
- Correção nas tags JSON da struct `CreateLabRequest` para garantir o parsing correto dos dados de entrada.

---
*Gerado automaticamente a partir da análise dos commits `d14fa22` e `30511dc`.*
