# Documentação da API

Esta documentação fornece detalhes sobre os endpoints da API para a aplicação Lab DevOps.

## URL Base

Todos os endpoints da API são prefixados com `/api/v1`.

## Autenticação

A documentação do código não especifica um método de autenticação. Assume-se que as rotas são públicas ou a autenticação é gerenciada em um nível superior (como um gateway de API).

## Endpoints

### Labs

---

#### **GET /labs**

- **Descrição:** Lista todos os laboratórios disponíveis.
- **Respostas:**
  - **200 OK:** Retorna um array de objetos de laboratório.
    ```json
    [
      {
        "id": "lab-tf-01",
        "title": "Terraform Básico",
        "type": "terraform",
        "track_id": "track-devops-01"
        
      }
    ]
    ```
  - **500 Internal Server Error:** Ocorreu um erro no servidor.
    ```json
    {
      "error": "mensagem de erro detalhada"
    }
    ```

---

#### **POST /labs**

- **Descrição:** Cria um novo laboratório.
- **Corpo da Requisição (JSON):**
  ```json
  {
    "title": "Meu Novo Lab",
    "type": "terraform",
    "instructions": "Faça X, Y e Z.",
    "initial_code": "resource \"local_file\" \"example\" { ... }",
    "track_id": "track-devops-01",
    "lab_order": 1
  }
  ```
- **Respostas:**
  - **201 Created:** Retorna o objeto do laboratório criado.
  - **400 Bad Request:** Payload da requisição é inválido.
  - **500 Internal Server Error:** Falha ao criar o laboratório.

---

#### **GET /labs/{labID}**

- **Descrição:** Busca os detalhes de um laboratório específico e seu workspace associado.
- **Parâmetros da URL:**
  - `labID` (string, **obrigatório**): O ID do laboratório.
- **Respostas:**
  - **200 OK:** Retorna um objeto contendo os detalhes do laboratório e do workspace.
    ```json
    {
      "lab": {
        "id": "lab-tf-01",
        "title": "Terraform Básico",
        ...
      },
      "workspace": {
        "id": "ws-tf-01",
        "last_state": "...",
        "status": "pending"
      }
    }
    ```
  - **404 Not Found:** O laboratório com o ID especificado não foi encontrado.

---

#### **DELETE /labs/{labID}**

- **Descrição:** Deleta um laboratório específico.
- **Parâmetros da URL:**
  - `labID` (string, **obrigatório**): O ID do laboratório a ser deletado.
- **Respostas:**
  - **200 OK:**
    ```json
    {
      "message": "Lab deletado com sucesso"
    }
    ```
  - **500 Internal Server Error:** Falha ao deletar o laboratório.

---

#### **GET /labs/{labID}/execute**

- **Descrição:** Inicia uma conexão WebSocket para executar o código de um laboratório em tempo real.
- **Tipo de Conexão:** WebSocket
- **Parâmetros da URL:**
  - `labID` (string, **obrigatório**): O ID do laboratório a ser executado.

- **Fluxo da Comunicação WebSocket:**
  1. **Upgrade:** O cliente solicita o upgrade da conexão HTTP para WebSocket.
  2. **Mensagem do Cliente (Client -> Server):** Após a conexão, o cliente deve enviar uma mensagem JSON para iniciar a execução.
     ```json
     {
       "action": "execute",
       "user_code": "código atualizado do usuário para executar"
     }
     ```
  3. **Mensagens do Servidor (Server -> Client):** O servidor enviará mensagens em tempo real sobre o status da execução.
     - **Logs da Execução:**
       ```json
       {
         "type": "log",
         "payload": "linha de log da execução"
       }
       ```
     - **Erro na Execução:**
       ```json
       {
         "type": "error",
         "payload": "mensagem de erro detalhada"
       }
       ```
     - **Execução Concluída com Sucesso:**
       ```json
       {
         "type": "complete",
         "payload": "Execução concluída com sucesso!"
       }
       ```

### Trilhas (Tracks)

---

#### **GET /tracks**

- **Descrição:** Lista todas as trilhas de aprendizado disponíveis.
- **Respostas:**
  - **200 OK:** Retorna um array de objetos de trilha.
    ```json
    [
      {
        "id": "track-devops-01",
        "title": "Trilha DevOps Completa",
        "description": "Do zero ao deploy."
      }
    ]
    ```
  - **500 Internal Server Error:** Ocorreu um erro no servidor.

---

#### **POST /tracks**

- **Descrição:** Cria uma nova trilha de aprendizado.
- **Corpo da Requisição (JSON):**
  ```json
  {
    "title": "Nova Trilha de Kubernetes",
    "description": "Aprenda a orquestrar contêineres com K8s."
  }
  ```
- **Respostas:**
  - **21 Created:** Retorna o objeto da trilha criada.
  - **400 Bad Request:** Payload da requisição é inválido.
  - **500 Internal Server Error:** Falha ao criar a trilha.

