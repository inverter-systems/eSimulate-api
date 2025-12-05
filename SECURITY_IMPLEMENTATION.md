# 🔐 Documentação de Segurança - eSimulate API

**Versão:** 2.4.0+ (Secure Auth & Security Enhancements)
**Data:** 2025-12-05

Este documento descreve todas as medidas de segurança implementadas no sistema.

---

## 📋 Resumo Executivo

O sistema implementa um conjunto abrangente de medidas de segurança para proteger contra:
- Ataques de força bruta
- Roubo de tokens
- Session fixation
- CSRF (proteção via SameSite cookies)
- XSS
- Enumeração de usuários
- E outros vetores de ataque comuns

---

## 🔒 Medidas de Segurança Implementadas

### 1. Sistema de Autenticação Híbrido

#### 1.1. Access Tokens (JWT)
- **Duração:** 15 minutos
- **Armazenamento:** Memória/storage do frontend
- **Transmissão:** Header `Authorization: Bearer <token>`
- **Algoritmo:** HMAC SHA256
- **Validação:** Expiração explícita com tolerância de 5 minutos para clock skew

#### 1.2. Refresh Tokens
- **Duração:** 7 dias
- **Armazenamento:** Cookie HttpOnly (não acessível via JavaScript)
- **Transmissão:** Cookie automático pelo browser
- **Geração:** Tokens criptograficamente seguros (32 bytes, base64URL)
- **Rotação:** Automática a cada refresh
- **Limite:** Máximo 5 tokens ativos por usuário

#### 1.3. Token Blacklist
- **Propósito:** Invalidar access tokens após logout
- **Armazenamento:** Memória (mapa com sync.RWMutex)
- **Expiração:** Tokens são removidos automaticamente após expiração
- **Limpeza:** Automática a cada 5 minutos

---

### 2. Rate Limiting

#### 2.1. Limites por Endpoint
| Endpoint | Limite | Janela |
|----------|--------|--------|
| Login | 5 requisições | 1 minuto |
| Registro | 3 requisições | 1 hora |
| Refresh | 10 requisições | 1 minuto |
| Esqueci Senha | 3 requisições | 1 hora |
| Verificar Email | 5 requisições | 1 minuto |

#### 2.2. Implementação
- **Método:** Rate limiting em memória (mapa com sync.RWMutex)
- **Chave:** IP do cliente + endpoint
- **Resposta:** HTTP 429 (Too Many Requests) com header `Retry-After`
- **Limpeza:** Automática a cada 5 minutos

---

### 3. Validação de Senha

#### 3.1. Requisitos Mínimos
- Mínimo 8 caracteres
- Máximo 128 caracteres
- Pelo menos uma letra maiúscula
- Pelo menos uma letra minúscula
- Pelo menos um número
- Pelo menos um símbolo

#### 3.2. Proteções Adicionais
- Verificação contra lista de senhas comuns
- Rejeição de senhas apenas numéricas ou apenas alfabéticas
- Hash com BCrypt (custo padrão)

---

### 4. Rotação de Refresh Tokens

#### 4.1. Como Funciona
1. Usuário faz login → Recebe access token + refresh token
2. Access token expira (15 min) → Frontend chama `/api/auth/refresh`
3. Backend:
   - Valida refresh token antigo
   - Marca como usado
   - Gera novo access token
   - Gera novo refresh token
   - Invalida refresh token antigo
   - Retorna novo access token
   - Atualiza cookie com novo refresh token

#### 4.2. Benefícios
- Reduz janela de ataque se token for comprometido
- Detecta reutilização de tokens (possível comprometimento)
- Limita tempo de validade de tokens roubados

---

### 5. Detecção de Reutilização de Tokens

#### 5.1. Mecanismo
- Tokens são marcados como "usados" após refresh
- Se um token usado for reutilizado, todos os tokens do usuário são invalidados
- Isso indica possível comprometimento ou ataque

#### 5.2. Resposta
- Log de segurança é gerado
- Todos os refresh tokens do usuário são revogados
- Usuário precisa fazer login novamente

---

### 6. Limite de Tokens por Usuário

#### 6.1. Regra
- Máximo 5 refresh tokens ativos por usuário
- Ao criar o 6º token, os tokens mais antigos são revogados
- Mantém apenas os 4 mais recentes + o novo

#### 6.2. Benefícios
- Previne acúmulo de tokens
- Reduz superfície de ataque
- Força logout em dispositivos antigos

---

### 7. CORS Restritivo

#### 7.1. Configuração
- **Desenvolvimento:** `CORS_ALLOWED_ORIGINS=*` (permite todas)
- **Produção:** `CORS_ALLOWED_ORIGINS=https://app.seudominio.com,https://www.seudominio.com`
- **Headers permitidos:** Authorization, Content-Type, X-CSRF-Token
- **Credentials:** Permitido (para cookies)

#### 7.2. Proteção
- Previne requisições de origens não autorizadas
- Reduz risco de CSRF
- Controla acesso à API

---

### 8. HTTPS Enforcement

#### 8.1. Em Produção
- Redireciona HTTP para HTTPS automaticamente
- Header HSTS (Strict-Transport-Security) configurado
- Cookies Secure apenas em HTTPS

#### 8.2. Configuração
- Ativado quando `ENV=production`
- Verifica `X-Forwarded-Proto` para proxies/load balancers

---

### 9. Mensagens de Erro Genéricas

#### 9.1. Proteção
- Não diferencia entre "usuário não existe" e "senha incorreta"
- Mensagem única: "Credenciais inválidas"
- Previne enumeração de usuários

#### 9.2. Exceções
- Erros de validação de senha são específicos (ajudam o usuário)
- Erros de formato são específicos (ajudam o desenvolvedor)

---

### 10. Logging de Segurança

#### 10.1. Eventos Registrados
- Tentativas de login (sucesso/falha)
- Tentativas de refresh (sucesso/falha)
- Reutilização de tokens
- Bloqueios por rate limit
- Logouts
- Reset de senha

#### 10.2. Informações Capturadas
- Tipo de evento
- User ID (se disponível)
- IP do cliente
- User-Agent
- Timestamp
- Detalhes adicionais

---

### 11. Validação de JWT

#### 11.1. Validações Implementadas
- Assinatura válida
- Algoritmo correto (HMAC)
- Expiração explícita (com tolerância de 5 min)
- Token não está na blacklist
- Claims obrigatórios presentes

#### 11.2. Resposta a Erros
- Token inválido → 401 Unauthorized
- Token expirado → 401 Unauthorized
- Token na blacklist → 401 Unauthorized

---

## 🛡️ Proteções Adicionais

### 12. Sanitização de Dados Públicos
- Gabaritos removidos de exames públicos
- `correctIndex` = -1
- `explanation` = ""

### 13. Limpeza Automática
- Tokens expirados removidos diariamente
- Links públicos expirados removidos diariamente
- Blacklist limpa tokens expirados a cada 5 minutos

### 14. BCrypt para Senhas
- Hash com custo padrão (10 rounds)
- Salt automático
- Resistente a rainbow tables

---

## 📊 Métricas de Segurança

### Monitoramento Recomendado
- Tentativas de login falhadas por IP
- Refresh tokens gerados por usuário
- Tokens revogados vs. ativos
- Logins de novos IPs/localizações
- Taxa de erro 401 vs. 403
- Eventos de reutilização de tokens

---

## 🔧 Configuração

### Variáveis de Ambiente

```bash
# Segurança
JWT_SECRET=<secret_forte_minimo_32_bytes>
ENV=production
CORS_ALLOWED_ORIGINS=https://app.seudominio.com

# Opcional
LOG_LEVEL=INFO
```

### Requisitos do JWT_SECRET
- Mínimo 32 bytes (256 bits)
- Gerado por CSPRNG
- Único por ambiente
- Nunca commitado no código
- Rotacionado periodicamente (a cada 90 dias)

---

## 🚨 Resposta a Incidentes

### Se um Token for Comprometido
1. Usuário faz logout → Token adicionado à blacklist
2. Todos os refresh tokens do usuário são invalidados
3. Usuário precisa fazer login novamente
4. Novo conjunto de tokens é gerado

### Se Múltiplos Tokens Forem Comprometidos
1. Invalidar todos os tokens do usuário
2. Forçar redefinição de senha
3. Notificar usuário
4. Investigar origem do comprometimento

---

## 📚 Referências

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [JWT Best Practices](https://datatracker.ietf.org/doc/html/rfc8725)
- [CORS Security](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS)
- [Rate Limiting](https://cloud.google.com/architecture/rate-limiting-strategies-techniques)

---

**Última atualização:** 2025-12-05
**Versão:** 2.4.0+

