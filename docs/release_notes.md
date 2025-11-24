# Release Notes - Validação Automática de Labs

## Novidades

### ✨ Validação Automática de Código
Agora, a plataforma Lab DevOps conta com um sistema inteligente de validação de desafios!
- **Feedback Instantâneo**: Ao submeter sua solução, o sistema verifica automaticamente se o objetivo do laboratório foi alcançado.
- **Correção Precisa**: Cada lab possui critérios específicos de sucesso (ex: verificar se um Pod Kubernetes está rodando ou se um bucket S3 foi criado).
- **Acompanhamento de Progresso**: Seus laboratórios só serão marcados como "Concluídos" após passarem na validação automática.

### 🚀 Novos Desafios
- **Labs Kubernetes (CKA)**: Adicionamos suporte a laboratórios preparatórios para a certificação CKA, com validação automática de recursos.

---

## Melhorias Técnicas

- **API WebSocket**: O endpoint de execução agora suporta o modo de validação (`action: "validate"`), permitindo separar a execução de testes da execução livre.
- **Banco de Dados**: Otimizações na estrutura de dados para suportar scripts de validação personalizados por laboratório.

---

*Aproveite as novidades e bons estudos!*
