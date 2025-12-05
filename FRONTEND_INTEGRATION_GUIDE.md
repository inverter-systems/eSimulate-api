# 🔐 Guia de Integração Frontend - Melhorias de Segurança

**Versão:** 2.4.0+ (Secure Auth & Security Enhancements)

Este documento descreve as mudanças de segurança implementadas no backend e como o frontend deve se adaptar para funcionar corretamente.

---

## 🚨 Mudanças Críticas que Requerem Ajustes no Frontend

### 1. Rotação Automática de Refresh Tokens

**O que mudou:**
- A cada chamada ao endpoint `/api/auth/refresh`, o backend agora **rota automaticamente** o refresh token
- Um **novo refresh token** é retornado no cookie `refresh_token`
- O token antigo é invalidado imediatamente

**O que o frontend precisa fazer:**
- ✅ **Nada!** O browser gerencia o cookie automaticamente
- ⚠️ **Importante:** Não armazene o refresh token manualmente - ele é gerenciado via cookie HttpOnly
- ⚠️ **Importante:** Se você estiver fazendo refresh manual, certifique-se de que o cookie está sendo atualizado

**Exemplo de fluxo:**
```javascript
// O frontend continua fazendo refresh normalmente
// O backend automaticamente atualiza o cookie
fetch('/api/auth/refresh', {
  method: 'POST',
  credentials: 'include' // IMPORTANTE: incluir cookies
})
.then(res => res.json())
.then(data => {
  // data.token contém o novo access token
  // O cookie refresh_token foi atualizado automaticamente
})
```

---

### 2. Rate Limiting Implementado

**O que mudou:**
- Endpoints de autenticação agora têm limites de requisições:
  - **Login:** 5 tentativas por minuto por IP
  - **Registro:** 3 tentativas por hora por IP
  - **Refresh:** 10 tentativas por minuto por IP
  - **Esqueci senha:** 3 tentativas por hora por IP
  - **Verificar email:** 5 tentativas por minuto por IP

**O que o frontend precisa fazer:**
- ✅ Tratar erro **429 (Too Many Requests)** quando o limite for excedido
- ✅ Exibir mensagem amigável ao usuário
- ✅ Implementar retry com backoff exponencial
- ✅ Mostrar contador de tempo até poder tentar novamente (usar header `Retry-After`)

**Exemplo de tratamento:**
```javascript
async function login(email, password) {
  try {
    const response = await fetch('/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ email, password })
    });
    
    if (response.status === 429) {
      const retryAfter = response.headers.get('Retry-After') || '60';
      throw new Error(`Muitas tentativas. Tente novamente em ${retryAfter} segundos.`);
    }
    
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Erro ao fazer login');
    }
    
    return await response.json();
  } catch (error) {
    // Tratar erro
  }
}
```

---

### 3. Mensagens de Erro Genéricas

**O que mudou:**
- Mensagens de erro agora são genéricas para não vazar informações:
  - ❌ Antes: "Usuário não encontrado" ou "Senha incorreta"
  - ✅ Agora: "Credenciais inválidas" (para ambos os casos)

**O que o frontend precisa fazer:**
- ✅ Usar mensagens genéricas na UI também
- ✅ Não tentar diferenciar entre "usuário não existe" e "senha incorreta"
- ✅ Mostrar mensagem única: "Email ou senha incorretos"

**Exemplo:**
```javascript
// ❌ ERRADO - não fazer isso
if (error.message === 'usuário não encontrado') {
  showError('Este email não está cadastrado');
} else if (error.message === 'senha incorreta') {
  showError('Senha incorreta');
}

// ✅ CORRETO
showError('Email ou senha incorretos. Verifique suas credenciais.');
```

---

### 4. Validação de Força de Senha

**O que mudou:**
- Senhas agora devem atender critérios mínimos:
  - Mínimo 8 caracteres
  - Pelo menos uma letra maiúscula
  - Pelo menos uma letra minúscula
  - Pelo menos um número
  - Pelo menos um símbolo

**O que o frontend precisa fazer:**
- ✅ Validar senha no frontend **antes** de enviar (melhor UX)
- ✅ Mostrar indicador de força de senha em tempo real
- ✅ Exibir mensagens de erro específicas do backend quando a validação falhar
- ✅ Guiar o usuário sobre os requisitos

**Exemplo:**
```javascript
// Validação no frontend (melhor UX)
function validatePassword(password) {
  const errors = [];
  if (password.length < 8) errors.push('Mínimo 8 caracteres');
  if (!/[A-Z]/.test(password)) errors.push('Uma letra maiúscula');
  if (!/[a-z]/.test(password)) errors.push('Uma letra minúscula');
  if (!/[0-9]/.test(password)) errors.push('Um número');
  if (!/[^A-Za-z0-9]/.test(password)) errors.push('Um símbolo');
  return errors;
}

// No formulário
const passwordErrors = validatePassword(password);
if (passwordErrors.length > 0) {
  showErrors(passwordErrors);
  return; // Não enviar ao backend
}
```

---

### 5. CORS Restritivo

**O que mudou:**
- CORS agora é configurável via variável de ambiente `CORS_ALLOWED_ORIGINS`
- Em produção, apenas origens específicas são permitidas
- Em desenvolvimento, pode usar `*` (permite todas)

**O que o frontend precisa fazer:**
- ✅ Garantir que o domínio do frontend está na lista de origens permitidas
- ✅ Usar `credentials: 'include'` em todas as requisições que precisam de cookies
- ✅ Configurar o backend com o domínio correto em produção

**Exemplo:**
```javascript
// Todas as requisições autenticadas devem incluir credentials
fetch('/api/exams', {
  method: 'GET',
  headers: {
    'Authorization': `Bearer ${accessToken}`,
    'Content-Type': 'application/json'
  },
  credentials: 'include' // IMPORTANTE para cookies
})
```

---

### 6. Logout Invalida Access Token Imediatamente

**O que mudou:**
- Ao fazer logout, o access token é adicionado a uma blacklist
- Tokens na blacklist são rejeitados mesmo que ainda não tenham expirado
- Refresh token também é invalidado

**O que o frontend precisa fazer:**
- ✅ Após logout, **remover o access token do storage** imediatamente
- ✅ Não tentar usar o token após logout
- ✅ Redirecionar para login após logout bem-sucedido

**Exemplo:**
```javascript
async function logout() {
  try {
    await fetch('/api/auth/logout', {
      method: 'POST',
      credentials: 'include'
    });
    
    // Remover token do storage
    localStorage.removeItem('accessToken');
    // ou sessionStorage.removeItem('accessToken');
    
    // Redirecionar para login
    window.location.href = '/login';
  } catch (error) {
    // Mesmo em caso de erro, limpar tokens locais
    localStorage.removeItem('accessToken');
    window.location.href = '/login';
  }
}
```

---

### 7. Tratamento de Erro 401 (Token Expirado)

**O que mudou:**
- Quando o access token expira (15 minutos), o backend retorna 401
- O frontend deve interceptar 401 e tentar refresh automaticamente
- Se o refresh falhar, fazer logout

**O que o frontend precisa fazer:**
- ✅ Implementar interceptor de requisições que detecta 401
- ✅ Tentar refresh automaticamente quando receber 401
- ✅ Se refresh falhar, fazer logout e redirecionar para login
- ✅ Evitar loops infinitos de refresh

**Exemplo de interceptor:**
```javascript
// Interceptor para refresh automático
let isRefreshing = false;
let failedQueue = [];

async function fetchWithAuth(url, options = {}) {
  const accessToken = localStorage.getItem('accessToken');
  
  const response = await fetch(url, {
    ...options,
    headers: {
      ...options.headers,
      'Authorization': `Bearer ${accessToken}`
    },
    credentials: 'include'
  });
  
  // Se token expirou, tentar refresh
  if (response.status === 401 && !url.includes('/auth/refresh')) {
    if (!isRefreshing) {
      isRefreshing = true;
      
      try {
        const refreshResponse = await fetch('/api/auth/refresh', {
          method: 'POST',
          credentials: 'include'
        });
        
        if (refreshResponse.ok) {
          const data = await refreshResponse.json();
          localStorage.setItem('accessToken', data.token);
          
          // Reprocessar requisições falhadas
          failedQueue.forEach(({ resolve }) => resolve());
          failedQueue = [];
          
          // Retentar requisição original
          return fetchWithAuth(url, options);
        } else {
          // Refresh falhou, fazer logout
          localStorage.removeItem('accessToken');
          window.location.href = '/login';
          throw new Error('Sessão expirada');
        }
      } finally {
        isRefreshing = false;
      }
    } else {
      // Já está fazendo refresh, aguardar
      return new Promise((resolve) => {
        failedQueue.push({ resolve });
      }).then(() => fetchWithAuth(url, options));
    }
  }
  
  return response;
}
```

---

### 8. Limite de Refresh Tokens por Usuário

**O que mudou:**
- Cada usuário pode ter no máximo **5 refresh tokens ativos** simultaneamente
- Ao criar um novo token (login em novo dispositivo), tokens antigos são revogados
- Isso previne acúmulo de tokens e reduz superfície de ataque

**O que o frontend precisa fazer:**
- ✅ **Nada específico** - o backend gerencia automaticamente
- ⚠️ **Nota:** Se um usuário fizer login em mais de 5 dispositivos, o dispositivo mais antigo será deslogado automaticamente

---

## 📋 Checklist de Implementação no Frontend

### Autenticação
- [ ] Interceptor para refresh automático de tokens (401 → refresh)
- [ ] Tratamento de erro 429 (Rate Limit) com mensagem e retry
- [ ] Logout remove tokens do storage e redireciona
- [ ] Todas as requisições usam `credentials: 'include'`

### Validação
- [ ] Validação de senha no frontend (melhor UX)
- [ ] Indicador de força de senha em tempo real
- [ ] Mensagens de erro genéricas na UI

### Segurança
- [ ] CORS configurado corretamente
- [ ] Domínio do frontend na lista de origens permitidas
- [ ] Não armazenar refresh token manualmente (usar cookie)

### UX
- [ ] Mensagens de erro amigáveis
- [ ] Feedback visual durante requisições
- [ ] Tratamento de erros de rede/timeout

---

## 🔄 Fluxo Completo de Autenticação

```
1. Login
   POST /api/auth/login
   → Retorna: { user: {...}, token: "access_token" }
   → Cookie: refresh_token (HttpOnly)

2. Requisições Autenticadas
   GET /api/exams
   Header: Authorization: Bearer {access_token}
   → Se 401: Ir para passo 3

3. Refresh Automático
   POST /api/auth/refresh
   Cookie: refresh_token (enviado automaticamente)
   → Retorna: { token: "novo_access_token" }
   → Cookie: refresh_token (atualizado automaticamente)
   → Retentar requisição original

4. Logout
   POST /api/auth/logout
   → Invalida refresh_token
   → Adiciona access_token à blacklist
   → Limpar tokens do storage
   → Redirecionar para login
```

---

## ⚠️ Pontos de Atenção

1. **Cookies HttpOnly:** O refresh token está em cookie HttpOnly, então JavaScript não pode acessá-lo diretamente. Isso é **intencional** para segurança.

2. **Access Token:** Deve ser armazenado em memória (variável) ou storage (localStorage/sessionStorage). Recomendação: usar sessionStorage para maior segurança.

3. **Rotação de Tokens:** A rotação é automática. Não tente gerenciar refresh tokens manualmente.

4. **Rate Limiting:** Implemente retry com backoff exponencial para evitar bloqueios.

5. **Mensagens de Erro:** Use mensagens genéricas na UI para não vazar informações sobre a existência de usuários.

---

## 🧪 Testes Recomendados

1. **Login com credenciais inválidas:** Deve mostrar mensagem genérica
2. **Rate limiting:** Fazer 6 tentativas de login rapidamente - deve retornar 429
3. **Token expirado:** Aguardar 15 minutos e fazer requisição - deve fazer refresh automático
4. **Logout:** Após logout, tentar usar token antigo - deve retornar 401
5. **Múltiplos dispositivos:** Login em 6 dispositivos - o primeiro deve ser deslogado
6. **Senha fraca:** Tentar registrar com senha fraca - deve mostrar erros específicos

---

## 📞 Suporte

Em caso de dúvidas ou problemas na integração, consulte:
- Documentação da API: `FRONTEND_CONTRACT_API.md`
- Logs do backend para debugging
- Código de exemplo acima

---

**Última atualização:** 2025-12-05
**Versão do Backend:** 2.4.0+

