# Documentação da API WebSocket

A API Lab DevOps fornece um endpoint WebSocket para interação em tempo real com os laboratórios. Esta conexão é usada para executar código do usuário, transmitir logs e validar soluções.

## Conexão

**URL**: `ws://<HOST>/api/v1/labs/:labID/execute`

- `:labID`: O identificador único do laboratório (ex: `lab-tf-01`).

## Protocolo

A comunicação segue um protocolo simples baseado em JSON.

### Mensagens do Cliente

O cliente envia mensagens JSON para iniciar ações.

#### 1. Executar Código
Executa o código fornecido pelo usuário no ambiente isolado do laboratório.

```json
{
  "action": "execute",
  "user_code": "resource \"aws_s3_bucket\" \"example\" { ... }"
}
```

- `action`: Deve ser `"execute"`.
- `user_code`: O conteúdo do arquivo a ser executado (ex: configuração Terraform, playbook Ansible).

#### 2. Validar Solução
Executa o script de validação do laboratório para verificar se o usuário completou o desafio com sucesso.

```json
{
  "action": "validate"
}
```

- `action`: Deve ser `"validate"`.

---

### Mensagens do Servidor

O servidor envia mensagens JSON para fornecer feedback em tempo real.

#### 1. Saída de Log
Saída padrão/erro transmitida do processo de execução.

```json
{
  "type": "log",
  "payload": "Initializing the backend..."
}
```

#### 2. Erro
Enviado quando ocorre um erro durante a execução ou validação.

```json
{
  "type": "error",
  "payload": "Error: Invalid resource type"
}
```

#### 3. Conclusão
Enviado quando o processo termina com sucesso.

```json
{
  "type": "complete",
  "payload": "Execution finished successfully."
}
```

## Fluxo de Exemplo

1.  **Cliente** conecta em `ws://localhost:8080/api/v1/labs/lab-tf-01/execute`.
2.  **Cliente** envia `{"action": "execute", "user_code": "..."}`.
3.  **Servidor** transmite logs:
    - `{"type": "log", "payload": "Terraform init..."}`
    - `{"type": "log", "payload": "Terraform apply..."}`
4.  **Servidor** envia conclusão:
    - `{"type": "complete", "payload": "Comando executado."}`
5.  **Cliente** envia `{"action": "validate"}`.
6.  **Servidor** transmite logs de validação.
7.  **Servidor** envia mensagem de sucesso (se a validação passar):
    - `{"type": "complete", "payload": "✅ Parabéns! Laboratório concluído com sucesso."}`
