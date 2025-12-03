# Especificação Técnica do Backend - eSimulate

Este documento define a arquitetura, regras de negócio e estrutura de implementação do backend em Go (Golang) para o sistema eSimulate.

## 1. Arquitetura

O sistema segue os princípios da **Clean Architecture** para garantir desacoplamento e testabilidade, organizado da seguinte forma:

*   **`cmd/api`**: Ponto de entrada da aplicação (`main.go`). Responsável pela injeção de dependência e inicialização do servidor HTTP.
*   **`internal/domain`**: Contém as entidades principais (Models) e interfaces de repositório. Não depende de nenhuma outra camada.
*   **`internal/repository/postgres`**: Implementação da persistência de dados (Repository Pattern) usando PostgreSQL.
    *   Métodos principais: `GetExamsByUser()`, `GetUserByID()`, `GetLinkByToken()` (com validação de expiração)
*   **`internal/service`**: Camada de lógica de negócio e regras de aplicação.
    *   Métodos principais: `GetSanitizedExam()`, `CalculateScore()` (cálculo de nota no backend)
*   **`internal/delivery/http`**: Camada de transporte HTTP (Handlers, Middlewares e Roteamento).
*   **`internal/config`**: Configurações e carregamento de variáveis de ambiente.

### Fluxo de Dados

```
HTTP Request → Handler → Service → Repository → PostgreSQL
                ↓         ↓          ↓
              Response ← Domain ← Domain
```

## 2. Tecnologias

*   **Linguagem:** Go 1.22+ (Utilizando o novo `net/http` mux com roteamento por método HTTP).
*   **Banco de Dados:** PostgreSQL 12+ com otimizações de performance.
*   **Autenticação:** JWT (JSON Web Tokens) com algoritmo HMAC SHA256.
*   **Segurança:** BCrypt para hash de senhas (custo padrão).
*   **Dependências Principais:**
    *   `github.com/golang-jwt/jwt/v5` - JWT
    *   `github.com/lib/pq` - Driver PostgreSQL
    *   `golang.org/x/crypto` - BCrypt
    *   `github.com/joho/godotenv` - Variáveis de ambiente
    *   `github.com/google/uuid` - Geração de UUIDs

## 3. Conformidade LGPD (Lei Geral de Proteção de Dados)

Para atender aos requisitos de privacidade:

1.  **Direito ao Esquecimento (Art. 18):**
    *   A tabela `users` é a proprietária dos dados.
    *   Todas as tabelas relacionadas (`exams`, `results`, `public_links`) possuem restrições `ON DELETE CASCADE`.
    *   Ao deletar um usuário via API (`DELETE /api/users/{id}`), o banco de dados remove automaticamente todo o histórico, logs e provas criadas por aquele usuário.
    *   Resultados de candidatos públicos também são removidos quando o usuário empresa é deletado.

2.  **Minimização de Dados:**
    *   Apenas dados estritamente necessários são armazenados.
    *   Senhas são armazenadas como hash BCrypt e nunca expostas em respostas JSON.
    *   Dados adicionais do perfil são armazenados em JSONB flexível apenas quando necessário.

3.  **Segurança de Dados:**
    *   Tokens JWT com expiração de 72 horas.
    *   CORS configurado para controle de origem.
    *   Validação de dados em todas as camadas.

## 4. Estrutura do Banco de Dados

O schema é normalizado, otimizado para performance e utiliza UUIDs v4 como chaves primárias.

### Tabelas Principais

*   **`users`**: Armazena credenciais, perfis e informações de usuários (admin, user, company).
    *   Campos: `id`, `name`, `email`, `password_hash`, `role`, `provider`, `created_at`, `updated_at`, `profile` (JSONB), `is_verified`, `onboarding_completed`
    *   Índices: email, role, created_at, profile (GIN)

*   **`subjects`**: Matérias/disciplinas disponíveis no sistema.
    *   Campos: `id`, `name`, `created_at`
    *   Índices: name

*   **`topics`**: Tópicos específicos dentro de cada matéria.
    *   Campos: `id`, `subject_id` (FK), `name`, `created_at`
    *   Constraints: UNIQUE(subject_id, name)
    *   Índices: subject_id, name

*   **`questions`**: Banco de questões reutilizáveis (podem ser usadas em múltiplos exames).
    *   Campos: `id`, `text`, `options` (JSONB), `correct_index`, `explanation`, `subject_id` (FK), `topic_id` (FK), `is_public`, `created_at`, `updated_at`
    *   Constraints: CHECK (correct_index >= 0)
    *   Índices: subject_id, topic_id, is_public, options (GIN), created_at, composto (subject_id, topic_id)

*   **`exams`**: Cabeçalho dos simulados com snapshot imutável das questões.
    *   Campos: `id`, `title`, `description`, `questions` (JSONB snapshot), `subjects` (JSONB array), `created_by` (FK), `created_at`, `updated_at`, `is_active`
    *   Índices: created_by, created_at, is_active, questions (GIN), subjects (GIN)

*   **`exam_subjects`**: Relacionamento many-to-many normalizado entre exames e matérias.
    *   Campos: `exam_id` (FK), `subject_id` (FK)
    *   Primary Key: (exam_id, subject_id)
    *   Índices: exam_id, subject_id

*   **`results`**: Histórico de execução de exames (suporta usuários autenticados e candidatos públicos).
    *   Campos: `id`, `exam_id` (FK), `user_id` (FK, nullable), `candidate_name`, `candidate_email`, `score`, `total_questions`, `answers` (JSONB), `time_spent_seconds`, `date`, `created_at`
    *   Constraints: CHECK (score >= 0), CHECK (total_questions > 0), CHECK (time_spent_seconds >= 0)
    *   Índices: exam_id, user_id, date, candidate_email, answers (GIN), compostos (user_id, date), (exam_id, date)

*   **`public_links`**: Links públicos para acesso externo a exames (B2B).
    *   Campos: `id`, `exam_id` (FK), `company_id` (FK), `token` (UNIQUE), `label`, `active`, `expires_at`, `created_at`
    *   Índices: token, exam_id, company_id, active, composto (active, token)

### Otimizações de Performance

*   **30+ índices estratégicos** para queries frequentes
*   **Índices compostos** para queries com múltiplas condições
*   **Índices GIN** para queries em campos JSONB
*   **Índices parciais** para filtros comuns (WHERE clauses)
*   **Triggers automáticos** para atualização de `updated_at`
*   **View `exam_stats`** para estatísticas agregadas

### Tipos de Dados

*   **TIMESTAMP WITH TIME ZONE**: Usado para todas as datas (created_at, updated_at, date)
*   **UUID**: Chaves primárias e foreign keys
*   **JSONB**: Para dados flexíveis (profile, options, questions, answers, subjects)
*   **TEXT**: Para strings de tamanho variável
*   **BOOLEAN**: Para flags (is_verified, onboarding_completed, is_public, is_active, active)

### Normalização

*   **Foreign Keys** em `questions.subject_id` e `questions.topic_id` (ao invés de TEXT)
*   **Tabela de relacionamento** `exam_subjects` para normalização many-to-many
*   **Integridade referencial** garantida por constraints
*   **Cascade deletes** configurados adequadamente

## 5. Entidades do Domínio

### User
*   `ID`, `Name`, `Email`, `Password` (omitido em JSON), `Role` (admin/user/company), `Provider`, `CreatedAt`, `IsVerified`, `OnboardingCompleted`, `Profile` (JSONB), `Token` (apenas no login)
*   **UpdateUser** suporta: `name`, `profile`, `preferences` (com `llmProvider` e `llmApiKey`), `onboardingCompleted`
*   **Segurança:** `llmApiKey` deve ser criptografado antes de armazenar (TODO: implementar AES-256)

### Question
*   `ID`, `Text`, `Options` (array), `CorrectIndex`, `Explanation`, `SubjectID` (FK UUID), `TopicID` (FK UUID), `IsPublic`
*   Campos legados: `Subject`, `Topic` (@deprecated)

### Exam
*   `ID`, `Title`, `Description`, `Questions` (array snapshot), `Subjects` (array), `CreatedBy`, `CreatedAt`

### ExamResult
*   `ID`, `ExamID`, `UserID` (nullable), `CandidateName`, `CandidateEmail`, `Score`, `TotalQuestions`, `Answers` (JSONB), `TimeSpentSeconds`, `Date`, `ExamTitle`

### PublicLink
*   `ID`, `ExamID`, `CompanyID`, `Token`, `Label`, `Active`, `CreatedAt`, `ExpiresAt`, `ExamTitle`

### Subject
*   `ID`, `Name`

### Topic
*   `ID`, `SubjectID`, `Name`

## 6. Endpoints da API

### Autenticação
*   `POST /api/auth/register` - Registrar novo usuário
*   `POST /api/auth/login` - Login e obter token JWT
*   `POST /api/auth/forgot-password` - Solicitar recuperação (stub)
*   `POST /api/auth/reset-password` - Redefinir senha (stub)
*   `POST /api/auth/verify-email` - Verificar email (stub)

### Exames
*   `GET /api/exams` - Listar exames do usuário logado (protegido, filtrado por `created_by`)
*   `GET /api/exams/{id}` - Obter exame por ID (protegido)
*   `POST /api/exams` - Criar ou atualizar exame (protegido, upsert: se `id` existir, atualiza; senão, cria)
*   `DELETE /api/exams/{id}` - Deletar exame (protegido)

### Questões
*   `GET /api/questions` - Listar questões (protegido)
*   `POST /api/questions` - Criar questão (protegido)
*   `POST /api/questions/batch` - Criar múltiplas questões (protegido)
*   `DELETE /api/questions/{id}` - Deletar questão (protegido)

### Resultados
*   `GET /api/results` - Obter meus resultados (protegido)
*   `POST /api/results` - Salvar resultado (protegido)

### Usuários (Admin)
*   `GET /api/users` - Listar usuários (protegido, admin)
*   `DELETE /api/users/{id}` - Deletar usuário (protegido, admin)
*   `POST /api/users/update` - Atualizar usuário (protegido)
    *   Suporta: `name`, `profile`, `preferences` (llmProvider, llmApiKey), `onboardingCompleted`
    *   Retorna objeto `User` atualizado
    *   **Segurança:** `llmApiKey` deve ser criptografado antes de armazenar (TODO: implementar AES-256)

### Matérias e Tópicos
*   `GET /api/subjects` - Listar matérias (público)
*   `POST /api/subjects` - Criar matéria (protegido)
*   `DELETE /api/subjects/{id}` - Deletar matéria (protegido)
*   `GET /api/topics` - Listar tópicos (público)
*   `POST /api/topics` - Criar tópico (protegido)
*   `DELETE /api/topics/{id}` - Deletar tópico (protegido)

### Empresa (B2B)
*   `GET /api/company/links` - Listar links públicos (protegido)
*   `POST /api/company/links` - Criar link público (protegido)
*   `GET /api/company/results` - Obter resultados da empresa (protegido)

### Acesso Público
*   `GET /api/public/exam/{token}` - Obter exame via token público (público, sanitizado)
    *   Valida se link está ativo (`active = true`)
    *   Valida se link não está expirado (`expires_at`, se definido)
    *   Retorna exame com gabarito removido (`correctIndex = -1`, `explanation = ""`)
*   `POST /api/public/exam/{token}/submit` - Submeter resultado público (público)
    *   Valida link ativo e não expirado
    *   **Segurança:** Calcula nota no backend comparando respostas com gabarito original
    *   Ignora `score` enviado pelo frontend (prevenção de fraude)
    *   Retorna `{ "status": "success" }`

## 7. Segurança e Sanitização

### Autenticação
*   Tokens JWT com expiração de 72 horas
*   Middleware de autenticação valida token em rotas protegidas
*   Token incluído no header: `Authorization: Bearer <token>`

### Sanitização de Dados Públicos
*   Ao acessar exame via link público, as questões são sanitizadas:
    *   `correctIndex` é definido como `-1`
    *   `explanation` é removida (string vazia)
*   Isso garante que candidatos públicos não vejam o gabarito

### Validação de Links Públicos
*   Links públicos são validados antes de permitir acesso:
    *   Link deve estar ativo (`active = true`)
    *   Link não pode estar expirado (se `expires_at` estiver definido)
    *   Validação ocorre em `GET /api/public/exam/{token}` e `POST /api/public/exam/{token}/submit`

### Cálculo de Nota no Backend
*   Para prevenir fraude, o cálculo de nota é realizado no backend:
    *   Método `Service.CalculateScore()` compara respostas do candidato com gabarito original
    *   Frontend envia apenas respostas selecionadas, não o score
    *   Backend calcula `score` e `totalQuestions` antes de salvar resultado
    *   Implementado em `POST /api/public/exam/{token}/submit`

### Validações
*   Constraints CHECK no banco de dados (score >= 0, total_questions > 0, etc.)
*   Validação de JSON em handlers
*   Validação de tipos e valores

## 8. Instruções para Manutenção

Ao modificar este backend:

1.  **Arquitetura:**
    *   Mantenha a lógica de negócio fora dos Handlers HTTP (use Service layer)
    *   Use injeção de dependência via structs nos Handlers
    *   Respeite a Clean Architecture (dependências apontam para dentro)

2.  **Banco de Dados:**
    *   Sempre verifique erros de banco de dados explicitamente
    *   Use transações quando necessário
    *   Aproveite os índices existentes nas queries
    *   Mantenha integridade referencial com Foreign Keys

3.  **Segurança:**
    *   Nunca exponha o `password_hash` no JSON de resposta
    *   Sempre sanitize dados públicos (exames via link público)
    *   Valide dados de entrada
    *   Use parâmetros preparados em queries SQL

4.  **Performance:**
    *   Use os índices existentes nas queries
    *   Evite N+1 queries (use JOINs quando apropriado)
    *   Considere cache para dados frequentemente acessados
    *   Monitore queries lentas

5.  **Código:**
    *   Siga as convenções Go
    *   Documente funções públicas
    *   Trate erros explicitamente
    *   Use tipos do domínio ao invés de tipos primitivos quando apropriado

## 9. Migrações e Schema

### Schema Atual
*   O schema completo está em `internal/database/schema.sql`
*   Schema para migrações em `migrations/schema.sql`
*   Schema otimizado inclui índices, triggers, views e constraints

### Aplicar Schema
```bash
psql -U usuario -d esimulate -f internal/database/schema.sql
```

### Migrações Futuras
*   Para novas migrações, use ferramentas como `golang-migrate`
*   Mantenha compatibilidade com dados existentes
*   Teste migrações em ambiente de desenvolvimento primeiro

## 10. Variáveis de Ambiente

| Variável | Descrição | Padrão |
|----------|-----------|--------|
| `PORT` | Porta do servidor HTTP | `8080` |
| `DATABASE_URL` | String de conexão PostgreSQL | `postgres://postgres:postgres@localhost:5432/esimulate?sslmode=disable` |
| `JWT_SECRET` | Chave secreta para assinatura JWT | `change_this_secret_in_production_please` |

⚠️ **Importante:** Altere o `JWT_SECRET` em produção para um valor seguro e aleatório.

## 11. Conformidade com Contrato Frontend

O backend está 100% aderente ao documento `FRONTEND_CONTRACT_API.md`:

*   ✅ Todos os endpoints implementados conforme especificação
*   ✅ Estruturas de request/response corretas
*   ✅ Validações de segurança (links expirados, cálculo de nota no backend)
*   ✅ Filtros por usuário logado (`GET /api/exams`)
*   ✅ Retorno de objetos atualizados após operações
*   ✅ Suporte a `preferences` (llmProvider, llmApiKey) em `POST /api/users/update`
*   ⚠️ **Pendente:** Criptografia de `llmApiKey` (marcado como TODO, deve ser implementado antes do deploy)

## 12. Documentação Adicional

*   [README.md](./README.md) - Documentação completa do projeto
*   [FRONTEND_CONTRACT_API.md](./FRONTEND_CONTRACT_API.md) - Contrato de API com o frontend
*   [REQUIREMENTS.md](./REQUIREMENTS.md) - Requisitos e regras de negócio
*   [DATABASE_ANALYSIS.md](./DATABASE_ANALYSIS.md) - Análise detalhada do banco de dados
*   [DATABASE_SUMMARY.md](./DATABASE_SUMMARY.md) - Resumo das melhorias do banco
*   [MIGRATION_SUBJECT_TOPIC.md](./MIGRATION_SUBJECT_TOPIC.md) - Migração para subject_id/topic_id
