# Especificação de Requisitos e Regras de Negócio - eSimulate

Este documento define os requisitos funcionais, não funcionais e as regras de negócio do sistema eSimulate.

## 📋 Índice

- [1. Visão Geral](#1-visão-geral)
- [2. Requisitos Funcionais](#2-requisitos-funcionais)
- [3. Requisitos Não Funcionais](#3-requisitos-não-funcionais)
- [4. Regras de Negócio](#4-regras-de-negócio)
- [5. Casos de Uso](#5-casos-de-uso)
- [6. Modelos de Dados](#6-modelos-de-dados)
- [7. Fluxos Principais](#7-fluxos-principais)

---

## 1. Visão Geral

### 1.1. Objetivo

O eSimulate é uma plataforma completa para criação, gerenciamento e execução de simulados e provas online, com suporte a:
- Usuários autenticados (estudantes, professores, administradores)
- Empresas (B2B) que podem criar links públicos para candidatos
- Candidatos públicos que acessam exames via links sem necessidade de cadastro

### 1.2. Escopo

O sistema permite:
- Gerenciamento de usuários com diferentes perfis (admin, user, company)
- Criação e gerenciamento de exames/simulados
- Banco de questões reutilizáveis
- Sistema de taxonomia (matérias e tópicos)
- Execução de exames com controle de tempo
- Armazenamento de resultados e estatísticas
- Geração de links públicos para acesso externo
- Conformidade com LGPD

### 1.3. Atores do Sistema

- **Administrador (admin)**: Acesso total ao sistema, pode gerenciar usuários, matérias, tópicos e questões
- **Usuário (user)**: Pode criar exames, responder questões e visualizar seus resultados
- **Empresa (company)**: Pode criar links públicos para candidatos e visualizar resultados
- **Candidato Público**: Acessa exames via link público sem autenticação

---

## 2. Requisitos Funcionais

### 2.1. Autenticação e Autorização

#### RF-001: Registro de Usuário
- **Descrição**: O sistema deve permitir o registro de novos usuários
- **Prioridade**: Alta
- **Regras**:
  - Email deve ser único no sistema
  - Senha deve ser hasheada com BCrypt antes de armazenar
  - Role padrão é "user" se não especificado
  - Usuário criado com `is_verified = false` e `onboarding_completed = false`

#### RF-002: Login
- **Descrição**: O sistema deve autenticar usuários e retornar token JWT
- **Prioridade**: Alta
- **Regras**:
  - Token JWT válido por 72 horas
  - Token contém `user_id` e `role`
  - Senha nunca é retornada na resposta

#### RF-003: Autorização por Role
- **Descrição**: O sistema deve controlar acesso baseado em roles
- **Prioridade**: Alta
- **Regras**:
  - Rotas protegidas requerem token JWT válido
  - Algumas rotas são públicas (GET /subjects, GET /topics, GET /public/exam/{token})
  - Admin tem acesso a todas as rotas administrativas

### 2.2. Gerenciamento de Usuários

#### RF-004: Perfil de Usuário
- **Descrição**: Usuários podem atualizar seu perfil
- **Prioridade**: Média
- **Regras**:
  - Campo `profile` é JSONB flexível (CPF, empresa, telefone, endereço)
  - Campo `onboarding_completed` pode ser atualizado
  - Email não pode ser alterado (requer processo separado)

#### RF-005: Gerenciamento de Usuários (Admin)
- **Descrição**: Administradores podem listar e deletar usuários
- **Prioridade**: Alta
- **Regras**:
  - Deletar usuário remove todos os dados relacionados (LGPD - cascade delete)
  - Apenas admin pode acessar lista de usuários

### 2.3. Gerenciamento de Exames

#### RF-006: Criação de Exame
- **Descrição**: Usuários autenticados podem criar exames
- **Prioridade**: Alta
- **Regras**:
  - Exame contém snapshot imutável das questões (JSONB)
  - Campo `created_by` é preenchido automaticamente com ID do usuário logado
  - Exame criado com `is_active = true`
  - `created_at` é definido automaticamente

#### RF-007: Listagem de Exames
- **Descrição**: Usuários podem listar seus exames
- **Prioridade**: Alta
- **Regras**:
  - Lista ordenada por `created_at DESC`
  - Apenas exames do usuário logado são retornados (futuro: filtro por created_by)

#### RF-008: Obter Exame por ID
- **Descrição**: Usuários podem obter detalhes de um exame específico
- **Prioridade**: Alta
- **Regras**:
  - Retorna exame completo com questões
  - Validação de acesso (futuro: verificar se usuário tem permissão)

#### RF-009: Exclusão de Exame
- **Descrição**: Usuários podem deletar seus exames
- **Prioridade**: Média
- **Regras**:
  - Deletar exame remove todos os resultados relacionados (cascade delete)
  - Links públicos relacionados também são removidos

### 2.4. Banco de Questões

#### RF-010: Criação de Questão
- **Descrição**: Usuários podem criar questões no banco
- **Prioridade**: Alta
- **Regras**:
  - Questão pode ter `subject_id` e `topic_id` (FK para normalização)
  - Campo `is_public` indica se questão é pública
  - `correct_index` deve ser >= 0
  - `options` é array JSONB de strings

#### RF-011: Criação em Lote
- **Descrição**: Usuários podem criar múltiplas questões de uma vez
- **Prioridade**: Média
- **Regras**:
  - Aceita array de questões
  - Processa sequencialmente (futuro: transação)

#### RF-012: Listagem de Questões
- **Descrição**: Usuários podem listar questões do banco
- **Prioridade**: Alta
- **Regras**:
  - Retorna todas as questões (futuro: filtros por subject, topic, is_public)

### 2.5. Taxonomia (Matérias e Tópicos)

#### RF-013: Gerenciamento de Matérias
- **Descrição**: Sistema permite gerenciar matérias/disciplinas
- **Prioridade**: Alta
- **Regras**:
  - Nome da matéria deve ser único
  - Listagem é pública (não requer autenticação)
  - Criação e exclusão requerem autenticação

#### RF-014: Gerenciamento de Tópicos
- **Descrição**: Sistema permite gerenciar tópicos dentro de matérias
- **Prioridade**: Alta
- **Regras**:
  - Tópico deve estar vinculado a uma matéria (subject_id)
  - Nome do tópico deve ser único por matéria (constraint unique_topic_per_subject)
  - Listagem é pública
  - Criação e exclusão requerem autenticação

### 2.6. Resultados

#### RF-015: Salvar Resultado
- **Descrição**: Sistema salva resultado de execução de exame
- **Prioridade**: Alta
- **Regras**:
  - Para usuários autenticados: `user_id` é preenchido automaticamente
  - Para candidatos públicos: `candidate_name` e `candidate_email` são obrigatórios
  - `score` deve ser >= 0
  - `total_questions` deve ser > 0
  - `time_spent_seconds` deve ser >= 0
  - `answers` é array JSONB: `[{questionId, selectedIndex, isCorrect}]`

#### RF-016: Histórico de Resultados
- **Descrição**: Usuários podem visualizar seu histórico de resultados
- **Prioridade**: Alta
- **Regras**:
  - Retorna apenas resultados do usuário logado
  - Ordenado por `date DESC`
  - Inclui título do exame (join com exams)

### 2.7. Funcionalidades B2B (Empresas)

#### RF-017: Criação de Link Público
- **Descrição**: Empresas podem criar links públicos para exames
- **Prioridade**: Alta
- **Regras**:
  - Token único de 8 caracteres gerado automaticamente
  - Link criado com `active = true`
  - Campo `expires_at` é opcional (NULL = sem expiração)
  - `company_id` é preenchido automaticamente com ID do usuário logado

#### RF-018: Listagem de Links
- **Descrição**: Empresas podem listar seus links públicos
- **Prioridade**: Alta
- **Regras**:
  - Retorna apenas links da empresa logada
  - Inclui título do exame (join com exams)

#### RF-019: Resultados de Candidatos
- **Descrição**: Empresas podem visualizar resultados de candidatos que usaram seus links
- **Prioridade**: Alta
- **Regras**:
  - Retorna apenas resultados de exames vinculados aos links da empresa
  - Apenas resultados com `candidate_name` preenchido (candidatos públicos)
  - Ordenado por `date DESC`

### 2.8. Acesso Público

#### RF-020: Acesso a Exame via Token
- **Descrição**: Candidatos podem acessar exame via link público
- **Prioridade**: Alta
- **Regras**:
  - **Sanitização obrigatória**: `correctIndex` deve ser -1 e `explanation` deve ser vazia
  - Link deve estar ativo (`active = true`)
  - Link não pode estar expirado (se `expires_at` estiver definido)
  - Retorna exame sanitizado e informações do link

#### RF-021: Submissão de Resultado Público
- **Descrição**: Candidatos podem submeter resultado de exame público
- **Prioridade**: Alta
- **Regras**:
  - Token deve ser válido e link ativo
  - `exam_id` é preenchido automaticamente a partir do link
  - `user_id` permanece NULL (candidato público)
  - `candidate_name` e `candidate_email` são obrigatórios

---

## 3. Requisitos Não Funcionais

### 3.1. Performance

#### RNF-001: Tempo de Resposta
- **Descrição**: APIs devem responder em menos de 200ms para 95% das requisições
- **Prioridade**: Alta
- **Implementação**: Índices otimizados no banco de dados

#### RNF-002: Escalabilidade
- **Descrição**: Sistema deve suportar 1000+ usuários simultâneos
- **Prioridade**: Média
- **Implementação**: Arquitetura stateless, conexões de banco otimizadas

### 3.2. Segurança

#### RNF-003: Autenticação Segura
- **Descrição**: Senhas devem ser hasheadas com BCrypt
- **Prioridade**: Crítica
- **Implementação**: BCrypt com custo padrão

#### RNF-004: Tokens JWT
- **Descrição**: Tokens JWT com expiração de 72 horas
- **Prioridade**: Alta
- **Implementação**: HMAC SHA256 com secret configurável

#### RNF-005: Sanitização de Dados
- **Descrição**: Dados públicos nunca devem expor gabarito
- **Prioridade**: Crítica
- **Implementação**: Sanitização obrigatória em `GetSanitizedExam`

### 3.3. Conformidade

#### RNF-006: LGPD
- **Descrição**: Sistema deve estar em conformidade com LGPD
- **Prioridade**: Crítica
- **Implementação**:
  - Cascade delete em todas as tabelas relacionadas
  - Minimização de dados
  - Senhas nunca expostas

#### RNF-007: Integridade de Dados
- **Descrição**: Dados devem manter integridade referencial
- **Prioridade**: Alta
- **Implementação**: Foreign Keys e constraints no banco

### 3.4. Disponibilidade

#### RNF-008: Uptime
- **Descrição**: Sistema deve ter 99.5% de disponibilidade
- **Prioridade**: Média
- **Implementação**: Tratamento de erros, health checks (futuro)

### 3.5. Manutenibilidade

#### RNF-009: Código Limpo
- **Descrição**: Código deve seguir Clean Architecture
- **Prioridade**: Alta
- **Implementação**: Separação de camadas, injeção de dependência

---

## 4. Regras de Negócio

### 4.1. Autenticação e Autorização

#### RN-001: Validação de Email
- Email deve ser único no sistema
- Email não pode ser alterado após criação

#### RN-002: Hash de Senha
- Senhas devem ser hasheadas com BCrypt antes de armazenar
- Senha nunca é retornada em respostas JSON
- Comparação de senha usa `bcrypt.CompareHashAndPassword`

#### RN-003: Token JWT
- Token contém `user_id` e `role`
- Expiração: 72 horas
- Validação obrigatória em rotas protegidas

#### RN-004: Roles e Permissões
- **admin**: Acesso total, pode gerenciar usuários, matérias, tópicos
- **user**: Pode criar exames, questões, visualizar resultados próprios
- **company**: Pode criar links públicos e visualizar resultados de candidatos

### 4.2. Gerenciamento de Exames

#### RN-005: Snapshot de Questões
- Exames contêm snapshot imutável das questões no momento da criação
- Alterações no banco de questões não afetam exames já criados
- Snapshot garante integridade histórica

#### RN-006: Propriedade de Exames
- Exame pertence ao usuário que o criou (`created_by`)
- Usuário pode deletar apenas seus próprios exames (futuro: validação)

#### RN-007: Soft Delete
- Exames têm campo `is_active` para soft delete
- Exames inativos não aparecem em listagens (futuro: filtro)

### 4.3. Banco de Questões

#### RN-008: Reutilização de Questões
- Questões podem ser reutilizadas em múltiplos exames
- Questões são independentes dos exames (não são deletadas quando exame é deletado)

#### RN-009: Relacionamento com Taxonomia
- Questões podem ter `subject_id` e `topic_id` (FK)
- Campos legados `subject` e `topic` (TEXT) mantidos para compatibilidade

#### RN-010: Questões Públicas
- Campo `is_public` indica se questão pode ser usada publicamente
- Questões públicas podem aparecer em exames públicos

### 4.4. Taxonomia

#### RN-011: Hierarquia Matéria-Tópico
- Tópicos devem estar vinculados a uma matéria
- Não é possível criar tópico sem matéria
- Deletar matéria remove todos os tópicos (CASCADE)

#### RN-012: Unicidade de Nomes
- Nome de matéria deve ser único no sistema
- Nome de tópico deve ser único por matéria (constraint unique_topic_per_subject)

### 4.5. Resultados

#### RN-013: Tipos de Resultado
- **Usuário Autenticado**: `user_id` preenchido, `candidate_name` e `candidate_email` NULL
- **Candidato Público**: `user_id` NULL, `candidate_name` e `candidate_email` obrigatórios

#### RN-014: Validação de Resultado
- `score` deve ser >= 0
- `total_questions` deve ser > 0
- `time_spent_seconds` deve ser >= 0
- `answers` deve ser array válido de objetos

#### RN-015: Histórico de Resultados
- Usuários autenticados veem apenas seus próprios resultados
- Empresas veem apenas resultados de candidatos que usaram seus links

### 4.6. Links Públicos (B2B)

#### RN-016: Geração de Token
- Token único de 8 caracteres gerado automaticamente
- Token deve ser único no sistema (constraint UNIQUE)

#### RN-017: Validação de Link
- Link deve estar ativo (`active = true`)
- Link não pode estar expirado (se `expires_at` estiver definido)
- Link deve estar vinculado a exame válido

#### RN-018: Propriedade de Links
- Links pertencem à empresa que os criou (`company_id`)
- Empresas veem apenas seus próprios links

### 4.7. Sanitização de Dados Públicos

#### RN-019: Sanitização Obrigatória
- Ao acessar exame via link público, questões devem ser sanitizadas:
  - `correctIndex` deve ser definido como `-1`
  - `explanation` deve ser removida (string vazia)
- Sanitização garante que candidatos não vejam gabarito

#### RN-020: Cálculo de Nota
- Nota deve ser calculada no backend (futuro: validação contra gabarito)
- Frontend pode enviar score, mas backend deve validar

### 4.8. LGPD e Privacidade

#### RN-021: Direito ao Esquecimento
- Deletar usuário remove automaticamente:
  - Todos os exames criados pelo usuário
  - Todos os resultados do usuário
  - Todos os links públicos criados (se for empresa)
- Cascade delete garante remoção completa

#### RN-022: Minimização de Dados
- Apenas dados necessários são armazenados
- Senhas são hasheadas
- Dados adicionais em JSONB apenas quando necessário

#### RN-023: Proteção de Dados Sensíveis
- `password_hash` nunca é exposto em respostas JSON
- Tokens JWT não contêm informações sensíveis
- Dados de perfil em JSONB podem ser criptografados (futuro)

---

## 5. Casos de Uso

### 5.1. UC-001: Registrar e Fazer Login

**Ator**: Usuário

**Fluxo Principal**:
1. Usuário acessa tela de registro
2. Preenche nome, email, senha
3. Sistema valida email único
4. Sistema hasheia senha
5. Sistema cria usuário com `is_verified = false`
6. Sistema retorna usuário criado
7. Usuário faz login com email e senha
8. Sistema valida credenciais
9. Sistema gera token JWT
10. Sistema retorna usuário com token

**Fluxos Alternativos**:
- Email já existe: retorna erro 400
- Senha incorreta: retorna erro 401

### 5.2. UC-002: Criar Exame

**Ator**: Usuário Autenticado

**Fluxo Principal**:
1. Usuário seleciona questões do banco
2. Usuário preenche título e descrição
3. Sistema cria snapshot das questões (JSONB)
4. Sistema associa exame ao usuário (`created_by`)
5. Sistema salva exame com `is_active = true`
6. Sistema retorna exame criado

**Regras**:
- Questões são snapshot imutável
- Exame pertence ao usuário que criou

### 5.3. UC-003: Executar Exame (Usuário Autenticado)

**Ator**: Usuário Autenticado

**Fluxo Principal**:
1. Usuário seleciona exame
2. Sistema retorna exame completo (com gabarito)
3. Usuário responde questões
4. Sistema registra tempo gasto
5. Sistema calcula nota
6. Sistema salva resultado com `user_id` preenchido
7. Sistema retorna resultado

### 5.4. UC-004: Criar Link Público (Empresa)

**Ator**: Empresa (role: company)

**Fluxo Principal**:
1. Empresa seleciona exame
2. Empresa preenche label (nome da vaga)
3. Sistema gera token único de 8 caracteres
4. Sistema cria link com `active = true`
5. Sistema associa link à empresa (`company_id`)
6. Sistema retorna link público

### 5.5. UC-005: Acessar Exame Público (Candidato)

**Ator**: Candidato Público

**Fluxo Principal**:
1. Candidato acessa link público (token)
2. Sistema valida token e link ativo
3. Sistema busca exame vinculado
4. Sistema sanitiza questões (remove gabarito)
5. Sistema retorna exame sanitizado e informações do link
6. Candidato responde questões
7. Candidato preenche nome e email
8. Sistema salva resultado com `user_id = NULL`
9. Sistema retorna confirmação

**Regras**:
- Sanitização obrigatória (correctIndex = -1, explanation = "")
- Candidato não precisa estar autenticado

### 5.6. UC-006: Visualizar Resultados (Empresa)

**Ator**: Empresa

**Fluxo Principal**:
1. Empresa acessa área de resultados
2. Sistema busca resultados de exames vinculados aos links da empresa
3. Sistema filtra apenas resultados com `candidate_name` (candidatos públicos)
4. Sistema ordena por data DESC
5. Sistema retorna lista de resultados

---

## 6. Modelos de Dados

### 6.1. User

```json
{
  "id": "uuid",
  "name": "string",
  "email": "string",
  "role": "admin|user|company",
  "provider": "email|google|github",
  "createdAt": "timestamp",
  "isVerified": "boolean",
  "onboardingCompleted": "boolean",
  "profile": {
    "taxId": "string",
    "companyName": "string",
    "phoneNumber": "string",
    "address": "string",
    "city": "string",
    "country": "string"
  },
  "token": "string (apenas no login)"
}
```

### 6.2. Question

```json
{
  "id": "uuid",
  "text": "string",
  "options": ["string", "string", ...],
  "correctIndex": "number (>= 0, -1 se sanitizado)",
  "explanation": "string",
  "subjectId": "uuid (FK)",
  "topicId": "uuid (FK)",
  "isPublic": "boolean"
}
```

### 6.3. Exam

```json
{
  "id": "uuid",
  "title": "string",
  "description": "string",
  "questions": [Question, ...], // Snapshot imutável
  "subjects": ["string", ...], // Array de nomes
  "createdBy": "uuid",
  "createdAt": "timestamp"
}
```

### 6.4. ExamResult

```json
{
  "id": "uuid",
  "examId": "uuid",
  "userId": "uuid (nullable para candidatos públicos)",
  "candidateName": "string (obrigatório se userId NULL)",
  "candidateEmail": "string (obrigatório se userId NULL)",
  "score": "number (>= 0)",
  "totalQuestions": "number (> 0)",
  "answers": [
    {
      "questionId": "uuid",
      "selectedIndex": "number",
      "isCorrect": "boolean"
    }
  ],
  "timeSpentSeconds": "number (>= 0)",
  "date": "timestamp",
  "examTitle": "string"
}
```

### 6.5. PublicLink

```json
{
  "id": "uuid",
  "examId": "uuid",
  "companyId": "uuid",
  "token": "string (8 caracteres, único)",
  "label": "string",
  "active": "boolean",
  "expiresAt": "timestamp (nullable)",
  "createdAt": "timestamp",
  "examTitle": "string"
}
```

---

## 7. Fluxos Principais

### 7.1. Fluxo de Autenticação

```
[Usuário] → [Registro] → [Validação] → [Hash Senha] → [Criação] → [Login] → [Validação] → [JWT] → [Token]
```

### 7.2. Fluxo de Criação de Exame

```
[Usuário] → [Seleciona Questões] → [Cria Snapshot] → [Salva Exame] → [Associa ao Usuário] → [Retorna]
```

### 7.3. Fluxo de Execução Pública

```
[Candidato] → [Acessa Token] → [Valida Link] → [Busca Exame] → [Sanitiza] → [Retorna] → 
[Responde] → [Submete] → [Salva Resultado] → [Confirma]
```

### 7.4. Fluxo B2B

```
[Empresa] → [Cria Link] → [Gera Token] → [Compartilha] → [Candidato Acessa] → 
[Executa] → [Submete] → [Empresa Visualiza Resultados]
```

---

## 8. Validações e Constraints

### 8.1. Validações de Entrada

- Email: formato válido e único
- Senha: mínimo de caracteres (futuro)
- Token: formato válido e único
- UUIDs: formato válido
- JSON: estrutura válida

### 8.2. Constraints de Banco

- `correct_index >= 0` em questions
- `score >= 0` em results
- `total_questions > 0` em results
- `time_spent_seconds >= 0` em results
- `UNIQUE(email)` em users
- `UNIQUE(token)` em public_links
- `UNIQUE(subject_id, name)` em topics

### 8.3. Validações de Negócio

- Link ativo para acesso público
- Link não expirado (se expires_at definido)
- Exame existe e está ativo
- Usuário tem permissão para ação
- Questões sanitizadas em acesso público

---

## 9. Tratamento de Erros

### 9.1. Códigos HTTP

- `200 OK`: Sucesso
- `201 Created`: Recurso criado
- `204 No Content`: Sucesso sem conteúdo
- `400 Bad Request`: Dados inválidos
- `401 Unauthorized`: Token inválido ou ausente
- `404 Not Found`: Recurso não encontrado
- `500 Internal Server Error`: Erro do servidor

### 9.2. Formato de Erro

```json
{
  "error": "Mensagem descritiva do erro"
}
```

---

## 10. Melhorias Futuras

### 10.1. Funcionalidades Planejadas

- Filtros avançados em listagens (por subject, topic, data)
- Paginação em listagens
- Cálculo de nota no backend (validação contra gabarito)
- Health checks e métricas
- Cache para dados frequentemente acessados
- Rate limiting
- Logs estruturados
- Testes automatizados

### 10.2. Segurança

- Criptografia de dados sensíveis no profile
- Refresh tokens
- 2FA (autenticação de dois fatores)
- Auditoria de ações

---

**Versão**: 1.0  
**Última Atualização**: 2024  
**Mantido por**: Equipe de Desenvolvimento eSimulate

