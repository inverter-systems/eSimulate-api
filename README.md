# eSimulate API

API RESTful desenvolvida em Go (Golang) para o sistema eSimulate - plataforma de simulados e provas online com suporte a acesso público e B2B.

## 📋 Índice

- [Sobre o Projeto](#sobre-o-projeto)
- [Tecnologias](#tecnologias)
- [Arquitetura](#arquitetura)
- [Requisitos](#requisitos)
- [Instalação](#instalação)
- [Configuração](#configuração)
- [Estrutura do Projeto](#estrutura-do-projeto)
- [Endpoints da API](#endpoints-da-api)
- [Banco de Dados](#banco-de-dados)
- [Conformidade LGPD](#conformidade-lgpd)
- [Desenvolvimento](#desenvolvimento)
- [Documentação Adicional](#documentação-adicional)

## 🎯 Sobre o Projeto

O eSimulate é uma plataforma completa para criação, gerenciamento e execução de simulados e provas online. A API oferece:

- ✅ Autenticação e autorização com JWT + Refresh Tokens
- ✅ Gerenciamento de usuários (admin, user, company, specialist)
- ✅ Criação e gerenciamento de exames
- ✅ Banco de questões reutilizáveis
- ✅ Sistema de resultados e estatísticas
- ✅ Links públicos para acesso externo (B2B)
- ✅ Conformidade com LGPD
- ✅ API RESTful completa
- ✅ **Medidas de segurança avançadas** (Rate limiting, Token rotation, CSRF protection, etc.)

## 🛠 Tecnologias

- **Linguagem:** Go 1.22+
- **Banco de Dados:** PostgreSQL 12+
- **Autenticação:** JWT (JSON Web Tokens) - HMAC SHA256
- **Segurança:** 
  - BCrypt para hash de senhas
  - Refresh Tokens com rotação automática
  - Rate limiting
  - Token blacklist
  - Validação de força de senha
  - CORS restritivo
  - HTTPS enforcement
  - SameSite cookies (proteção CSRF)
  - Logging de segurança
- **HTTP Router:** Go 1.22 `net/http` mux (padrão)
- **Dependências Principais:**
  - `github.com/golang-jwt/jwt/v5` - JWT
  - `github.com/lib/pq` - Driver PostgreSQL
  - `golang.org/x/crypto` - BCrypt
  - `github.com/joho/godotenv` - Variáveis de ambiente

## 🏗 Arquitetura

O projeto segue os princípios da **Clean Architecture** para garantir desacoplamento e testabilidade:

```
cmd/api/              # Ponto de entrada da aplicação
internal/
  ├── config/         # Configurações e variáveis de ambiente
  ├── domain/         # Entidades e interfaces (camada de domínio)
  ├── repository/     # Implementação de persistência (PostgreSQL)
  ├── service/        # Lógica de negócio
  └── delivery/       # Handlers HTTP e middlewares
```

### Camadas

- **Domain**: Entidades e interfaces puras (sem dependências externas)
- **Repository**: Implementação de persistência usando Repository Pattern
- **Service**: Lógica de negócio e regras de aplicação
- **Delivery**: Camada de transporte HTTP (handlers, middlewares, rotas)

## 📦 Requisitos

- Go 1.22 ou superior
- PostgreSQL 12 ou superior
- Make (opcional, para comandos auxiliares)

## 🚀 Instalação

### 1. Clone o repositório

```bash
git clone <repository-url>
cd eSimulate-api
```

### 2. Instale as dependências

```bash
go mod download
```

### 3. Configure o banco de dados

Crie um banco de dados PostgreSQL:

```sql
CREATE DATABASE esimulate;
```

### 4. Configure as variáveis de ambiente

O sistema procura as variáveis de ambiente na seguinte ordem:

1. **Arquivo `.env` na raiz do projeto** (recomendado para desenvolvimento)
2. **Variáveis de ambiente do sistema operacional** (usado em produção)

#### Opção 1: Usando arquivo `.env` (Desenvolvimento)

Crie um arquivo `.env` na raiz do projeto (veja `env.example` como referência):

```bash
cp env.example .env
```

Edite o arquivo `.env` com seus valores:

```env
PORT=8080
DATABASE_URL=postgres://usuario:senha@localhost:5432/esimulate?sslmode=disable
JWT_SECRET=seu_secret_jwt_super_seguro_aqui
```

#### Opção 2: Variáveis de ambiente do sistema (Produção)

Configure as variáveis diretamente no sistema operacional:

```bash
export PORT=8080
export DATABASE_URL=postgres://usuario:senha@localhost:5432/esimulate?sslmode=disable
export JWT_SECRET=seu_secret_jwt_super_seguro_aqui
```

**Nota:** Se uma variável não for encontrada, o sistema usará valores padrão e exibirá um aviso no log.

#### Variáveis de Admin Inicial

O sistema cria automaticamente um usuário admin na primeira inicialização:

- `ADMIN_EMAIL`: Email do admin (padrão: `admin@esimulate.com`)
- `ADMIN_PASSWORD`: Senha do admin (padrão: `admin123`)

**IMPORTANTE:** Altere essas credenciais em produção!

### 5. Execute o schema do banco

```bash
psql -U usuario -d esimulate -f internal/database/schema.sql
```

Ou use o arquivo em `migrations/schema.sql`:

```bash
psql -U usuario -d esimulate -f migrations/schema.sql
```

### 6. Execute a aplicação

```bash
go run ./cmd/api/main.go
```

A API estará disponível em `http://localhost:8080`

## ⚙️ Configuração

### Variáveis de Ambiente

| Variável | Descrição | Padrão |
|----------|-----------|--------|
| `PORT` | Porta do servidor HTTP | `8080` |
| `DATABASE_URL` | String de conexão PostgreSQL | `postgres://postgres:postgres@localhost:5432/esimulate?sslmode=disable` |
| `JWT_SECRET` | Chave secreta para assinatura JWT | `change_this_secret_in_production_please` |

⚠️ **Importante:** Altere o `JWT_SECRET` em produção para um valor seguro e aleatório.

## 📁 Estrutura do Projeto

```
eSimulate-api/
├── cmd/
│   └── api/
│       └── main.go              # Ponto de entrada
├── internal/
│   ├── config/
│   │   └── config.go            # Configurações
│   ├── domain/
│   │   ├── entity.go            # Entidades do domínio
│   │   └── repository.go        # Interfaces de repositório
│   ├── repository/
│   │   └── postgres/
│   │       └── repository.go    # Implementação PostgreSQL
│   ├── service/
│   │   └── service.go           # Lógica de negócio
│   ├── delivery/
│   │   └── http/
│   │       ├── handler.go       # Handlers HTTP
│   │       └── middleware.go    # Middlewares (CORS, Auth)
│   └── database/
│       └── schema.sql           # Schema do banco de dados
├── migrations/
│   └── schema.sql               # Schema para migrações
├── go.mod                        # Dependências Go
├── go.sum                        # Checksums das dependências
└── README.md                     # Este arquivo
```

## 🔌 Endpoints da API

### Autenticação

| Método | Endpoint | Descrição | Autenticação |
|--------|----------|-----------|--------------|
| POST | `/api/auth/register` | Registrar novo usuário | ❌ |
| POST | `/api/auth/login` | Login e obter token JWT | ❌ |
| POST | `/api/auth/forgot-password` | Solicitar recuperação de senha | ❌ |
| POST | `/api/auth/reset-password` | Redefinir senha | ❌ |
| POST | `/api/auth/verify-email` | Verificar email | ❌ |

### Exames

| Método | Endpoint | Descrição | Autenticação |
|--------|----------|-----------|--------------|
| GET | `/api/exams` | Listar exames | ✅ |
| GET | `/api/exams/{id}` | Obter exame por ID | ✅ |
| POST | `/api/exams` | Criar novo exame | ✅ |
| DELETE | `/api/exams/{id}` | Deletar exame | ✅ |

### Questões

| Método | Endpoint | Descrição | Autenticação |
|--------|----------|-----------|--------------|
| GET | `/api/questions` | Listar questões | ✅ |
| POST | `/api/questions` | Criar questão | ✅ |
| POST | `/api/questions/batch` | Criar múltiplas questões | ✅ |
| DELETE | `/api/questions/{id}` | Deletar questão | ✅ |

### Resultados

| Método | Endpoint | Descrição | Autenticação |
|--------|----------|-----------|--------------|
| GET | `/api/results` | Obter meus resultados | ✅ |
| POST | `/api/results` | Salvar resultado | ✅ |

### Usuários (Admin)

| Método | Endpoint | Descrição | Autenticação |
|--------|----------|-----------|--------------|
| GET | `/api/users` | Listar usuários | ✅ |
| DELETE | `/api/users/{id}` | Deletar usuário | ✅ |
| POST | `/api/users/update` | Atualizar usuário | ✅ |

### Matérias e Tópicos

| Método | Endpoint | Descrição | Autenticação |
|--------|----------|-----------|--------------|
| GET | `/api/subjects` | Listar matérias | ❌ |
| POST | `/api/subjects` | Criar matéria | ✅ |
| DELETE | `/api/subjects/{id}` | Deletar matéria | ✅ |
| GET | `/api/topics` | Listar tópicos | ❌ |
| POST | `/api/topics` | Criar tópico | ✅ |
| DELETE | `/api/topics/{id}` | Deletar tópico | ✅ |

### Empresa (B2B)

| Método | Endpoint | Descrição | Autenticação |
|--------|----------|-----------|--------------|
| GET | `/api/company/links` | Listar links públicos | ✅ |
| POST | `/api/company/links` | Criar link público | ✅ |
| GET | `/api/company/results` | Obter resultados da empresa | ✅ |

### Acesso Público

| Método | Endpoint | Descrição | Autenticação |
|--------|----------|-----------|--------------|
| GET | `/api/public/exam/{token}` | Obter exame via token público | ❌ |
| POST | `/api/public/exam/{token}/submit` | Submeter resultado público | ❌ |

### Autenticação

Para endpoints protegidos, inclua o header:

```
Authorization: Bearer <seu_token_jwt>
```

## 🗄 Banco de Dados

### Schema

O banco de dados utiliza PostgreSQL com:

- ✅ Normalização adequada
- ✅ Foreign Keys para integridade referencial
- ✅ Índices otimizados para performance
- ✅ UUIDs v4 como chaves primárias
- ✅ JSONB para dados flexíveis
- ✅ Triggers para atualização automática
- ✅ Constraints para validação

### Tabelas Principais

- `users` - Usuários do sistema
- `subjects` - Matérias/disciplinas
- `topics` - Tópicos dentro de matérias
- `questions` - Banco de questões
- `exams` - Simulados/provas
- `exam_subjects` - Relacionamento exames-matérias
- `results` - Resultados de execução
- `public_links` - Links públicos para acesso externo

### Migração

O schema está disponível em:
- `internal/database/schema.sql` - Schema completo
- `migrations/schema.sql` - Schema para migrações

Para aplicar o schema:

```bash
psql -U usuario -d esimulate -f internal/database/schema.sql
```

## 🔒 Conformidade LGPD

O sistema foi desenvolvido com foco em conformidade com a Lei Geral de Proteção de Dados (LGPD):

### Direito ao Esquecimento (Art. 18)

- Todas as tabelas relacionadas possuem `ON DELETE CASCADE`
- Ao deletar um usuário, todos os dados relacionados são removidos automaticamente
- Histórico, logs e provas criadas pelo usuário são eliminados

### Minimização de Dados

- Apenas dados estritamente necessários são armazenados
- Senhas são armazenadas como hash (BCrypt)
- `password_hash` nunca é exposto em respostas JSON

### Segurança

- Autenticação via JWT
- Senhas hasheadas com BCrypt
- CORS configurado
- Validação de dados

## 💻 Desenvolvimento

### Executar em modo desenvolvimento

```bash
go run ./cmd/api/main.go
```

### Compilar

```bash
go build -o bin/api ./cmd/api/main.go
```

### Executar binário compilado

```bash
./bin/api
```

### Testes

```bash
go test ./...
```

### Formatação

```bash
go fmt ./...
```

### Linting

```bash
golangci-lint run
```

### Boas Práticas

Ao modificar o código:

1. ✅ Mantenha a lógica de negócio fora dos Handlers HTTP
2. ✅ Use injeção de dependência via structs
3. ✅ Sempre verifique erros explicitamente
4. ✅ Nunca exponha `password_hash` em respostas JSON
5. ✅ Siga a Clean Architecture
6. ✅ Documente funções públicas
7. ✅ Use mensagens de erro genéricas (não vaze informações)
8. ✅ Implemente rate limiting em novos endpoints sensíveis
9. ✅ Valide entrada do usuário (senha, email, etc.)
10. ✅ Use logging de segurança para eventos importantes

## 📚 Documentação Adicional

### Especificações
- [REQUIREMENTS.md](./REQUIREMENTS.md) - **Especificação de requisitos e regras de negócio**
- [BACKEND_SPEC.md](./BACKEND_SPEC.md) - Especificação técnica detalhada
- [FRONTEND_CONTRACT_API.md](../node/react/eSimulate/docs/FRONTEND_CONTRACT_API.md) - Contrato de API para frontend

### Segurança
- [SECURITY_IMPLEMENTATION.md](./SECURITY_IMPLEMENTATION.md) - **Documentação completa de segurança**
- [FRONTEND_INTEGRATION_GUIDE.md](./FRONTEND_INTEGRATION_GUIDE.md) - **Guia de integração para frontend**

### Banco de Dados
- [DATABASE_ANALYSIS.md](./DATABASE_ANALYSIS.md) - Análise e otimização do banco de dados
- [DATABASE_SUMMARY.md](./DATABASE_SUMMARY.md) - Resumo das melhorias do banco
- [MIGRATION_SUBJECT_TOPIC.md](./MIGRATION_SUBJECT_TOPIC.md) - Migração para subject_id/topic_id

## 🤝 Contribuindo

1. Faça um fork do projeto
2. Crie uma branch para sua feature (`git checkout -b feature/AmazingFeature`)
3. Commit suas mudanças (`git commit -m 'Add some AmazingFeature'`)
4. Push para a branch (`git push origin feature/AmazingFeature`)
5. Abra um Pull Request

## 📝 Licença

Este projeto está sob a licença especificada no arquivo [LICENSE](./LICENSE).

## 👥 Autores

- Equipe de Desenvolvimento eSimulate

## 🙏 Agradecimentos

- Comunidade Go
- PostgreSQL
- Todos os mantenedores das bibliotecas utilizadas

---

**Desenvolvido com ❤️ usando Go**
