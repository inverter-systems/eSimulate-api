# Especificação de Backend - eSimulate

Este documento descreve os contratos de API (Endpoints, Request/Response) necessários para atender ao Frontend do eSimulate. O backend deve ser implementado externamente (ex: Go, Node, Python) seguindo estas especificações.

## 1. Visão Geral
*   **Base URL:** `http://localhost:8080/api` (padrão desenvolvimento) ou variável de ambiente.
*   **Formato:** JSON.
*   **Autenticação:** Bearer Token (JWT).
*   **Erros:** Retornar status 4xx/5xx com corpo JSON: `{ "error": "Mensagem descritiva" }`.

## 2. Autenticação & Usuários

### `POST /auth/register`
Cria um novo usuário.
*   **Body:** `{ "name": "...", "email": "...", "password": "...", "role": "user|company|admin", "provider": "email" }`
*   **Response (201):** Objeto `User` completo.

### `POST /auth/login`
Autentica usuário e retorna token.
*   **Body:** `{ "email": "...", "password": "..." }`
*   **Response (200):** Objeto `User` com campo `token` preenchido.

### `POST /users/update`
Atualiza perfil do usuário, incluindo preferências e chaves de API (Bring Your Own Key).
*   **Headers:** `Authorization: Bearer <token>`
*   **Body:** 
    ```json
    { 
      "id": "uuid", 
      "name": "Nome",
      "profile": { "taxId": "...", "companyName": "...", "phoneNumber": "...", "address": "...", "city": "...", "country": "..." }, 
      "preferences": {
        "llmProvider": "gemini",
        "llmApiKey": "AIzaSy..."
      }
    }
    ```
*   **Segurança (Backend Requirement):** O campo `llmApiKey` NUNCA deve ser armazenado em texto plano no banco de dados. Ele deve ser criptografado (AES-256 ou similar) antes da persistência e descriptografado apenas no momento do uso pelo serviço de IA ou retornado ao usuário (opcionalmente mascarado).
*   **Response (200):** Objeto `User` atualizado.

### `GET /users` (Admin)
Lista todos os usuários.
*   **Headers:** `Authorization: Bearer <token>`
*   **Response (200):** Lista de `User`.

### `DELETE /users/:id` (Admin)
Remove um usuário e seus dados (LGPD - Cascade Delete).
*   **Headers:** `Authorization: Bearer <token>`
*   **Response (204):** No Content.

---

## 3. Simulados (Exams)

### `GET /exams`
Lista simulados criados pelo usuário logado.
*   **Headers:** `Authorization: Bearer <token>`
*   **Response (200):** Lista de `Exam` (ordenado por `createdAt` desc).

### `GET /exams/:id`
Obtém detalhes de um simulado.
*   **Headers:** `Authorization: Bearer <token>`
*   **Response (200):** Objeto `Exam`.

### `POST /exams`
Cria ou atualiza um simulado.
*   **Headers:** `Authorization: Bearer <token>`
*   **Body:** Objeto `Exam`. Se `id` existir, atualiza; senão, cria.
*   **Response (201):** Objeto `Exam`.

### `DELETE /exams/:id`
Exclui um simulado.
*   **Headers:** `Authorization: Bearer <token>`
*   **Response (204):** No Content.

---

## 4. Banco de Questões (Question Bank)

### `GET /questions`
Lista questões do banco (pode aceitar filtros query params futuramente).
*   **Headers:** `Authorization: Bearer <token>`
*   **Response (200):** Lista de `Question`.

### `POST /questions`
Salva uma questão única.
*   **Headers:** `Authorization: Bearer <token>`
*   **Body:** Objeto `Question`.
*   **Response (201):** Objeto `Question`.

### `POST /questions/batch`
Salva múltiplas questões de uma vez.
*   **Headers:** `Authorization: Bearer <token>`
*   **Body:** Lista de `Question`.
*   **Response (200):** Vazio ou status.

### `DELETE /questions/:id`
Remove uma questão do banco.
*   **Headers:** `Authorization: Bearer <token>`
*   **Response (204):** No Content.

---

## 5. Matérias e Tópicos (Taxonomia)

### `GET /subjects`
Lista matérias cadastradas.
*   **Response (200):** Lista de `{ "id": "...", "name": "..." }`.

### `POST /subjects` (Admin)
Cria nova matéria.
*   **Headers:** `Authorization: Bearer <token>`
*   **Body:** `{ "name": "..." }`
*   **Response (201):** Objeto `Subject`.

### `DELETE /subjects/:id` (Admin)
*   **Headers:** `Authorization: Bearer <token>`
*   **Response (204):** No Content.

### `GET /topics`
Lista tópicos.
*   **Response (200):** Lista de `{ "id": "...", "subjectId": "...", "name": "..." }`.

### `POST /topics` (Admin)
Cria novo tópico vinculado a uma matéria.
*   **Headers:** `Authorization: Bearer <token>`
*   **Body:** `{ "name": "...", "subjectId": "..." }`
*   **Response (201):** Objeto `Topic`.

### `DELETE /topics/:id` (Admin)
*   **Headers:** `Authorization: Bearer <token>`
*   **Response (204):** No Content.

---

## 6. Resultados (Results)

### `POST /results`
Salva o resultado de uma prova realizada.
*   **Headers:** `Authorization: Bearer <token>`
*   **Body:** Objeto `ExamResult` (inclui respostas detalhadas, score, tempo).
*   **Response (201):** Objeto `ExamResult`.

### `GET /results`
Retorna histórico de resultados do usuário logado.
*   **Headers:** `Authorization: Bearer <token>`
*   **Response (200):** Lista de `ExamResult`.

---

## 7. Empresa (B2B)

### `GET /company/links`
Lista links públicos gerados pela empresa.
*   **Headers:** `Authorization: Bearer <token>` (Role: company)
*   **Response (200):** Lista de `PublicLink`.

### `POST /company/links`
Gera novo link público para candidatos.
*   **Headers:** `Authorization: Bearer <token>` (Role: company)
*   **Body:** `{ "examId": "...", "label": "Nome da Vaga" }`
*   **Response (201):** Objeto `PublicLink`.

### `GET /company/results`
Lista resultados de candidatos que fizeram provas via links desta empresa.
*   **Headers:** `Authorization: Bearer <token>` (Role: company)
*   **Response (200):** Lista de `ExamResult` (com `candidateName` e `candidateEmail` preenchidos).

---

## 8. Acesso Público (Candidatos)

### `GET /public/exam/:token`
Retorna a prova para o candidato realizar.
*   **Segurança:** Deve retornar o objeto `Exam` **SANITIZADO**.
    *   `correctIndex` deve ser removido ou definido como -1.
    *   `explanation` deve ser removida ou vazia.
*   **Response (200):** `{ "exam": ExamSanitized, "link": PublicLink }`

### `POST /public/exam/:token/submit`
Recebe as respostas do candidato.
*   **Body:** Objeto `ExamResult` (apenas respostas selecionadas e dados do candidato).
*   **Lógica Backend:** O backend deve calcular a nota (`score`) comparando com o gabarito original no banco, para evitar fraude no frontend.
*   **Response (200):** `{ "status": "success" }`

---

## Modelos de Dados (Typescript Reference)

Consulte o arquivo `types.ts` na raiz do projeto frontend para ver a estrutura exata dos objetos JSON (User, Question, Exam, ExamResult, etc).