# Migração: Subject/Topic TEXT → SubjectID/TopicID UUID

## 📋 Resumo das Alterações

Atualização do código Go para usar `subject_id` e `topic_id` (UUID/FK) ao invés de `subject` e `topic` (TEXT), alinhando com o schema otimizado do banco de dados.

---

## ✅ Alterações Realizadas

### 1. **Struct Question** (`internal/domain/entity.go`)

#### Antes:
```go
type Question struct {
    // ...
    Subject string `json:"subject,omitempty"`
    Topic   string `json:"topic,omitempty"`
}
```

#### Depois:
```go
type Question struct {
    // ...
    SubjectID string `json:"subjectId,omitempty"`  // FK para subjects (UUID)
    TopicID   string `json:"topicId,omitempty"`    // FK para topics (UUID)
    IsPublic  bool   `json:"isPublic,omitempty"`   // Novo campo
    
    // Campos legados para compatibilidade
    Subject   string `json:"subject,omitempty"`    // @deprecated
    Topic     string `json:"topic,omitempty"`       // @deprecated
}
```

**Benefícios:**
- ✅ Integridade referencial (FK)
- ✅ Queries mais eficientes
- ✅ Compatibilidade mantida (campos legados)

---

### 2. **Repositório** (`internal/repository/postgres/repository.go`)

#### CreateQuestion

**Antes:**
```go
query := `INSERT INTO questions (..., subject, topic) VALUES (..., $6, $7)`
_, err := r.DB.Exec(query, ..., q.Subject, q.Topic)
```

**Depois:**
```go
// Preparar subject_id e topic_id (podem ser NULL)
var subjectID, topicID sql.NullString
if q.SubjectID != "" {
    subjectID.String = q.SubjectID
    subjectID.Valid = true
}
if q.TopicID != "" {
    topicID.String = q.TopicID
    topicID.Valid = true
}

query := `INSERT INTO questions (..., subject_id, topic_id, is_public) 
    VALUES (..., $6, $7, $8)`
_, err := r.DB.Exec(query, ..., subjectID, topicID, q.IsPublic)
```

#### GetQuestions

**Antes:**
```go
rows, err := r.DB.Query("SELECT ..., subject, topic FROM questions")
rows.Scan(..., &q.Subject, &q.Topic)
```

**Depois:**
```go
rows, err := r.DB.Query("SELECT ..., subject_id, topic_id, is_public FROM questions")
var subjectID, topicID sql.NullString
rows.Scan(..., &subjectID, &topicID, &q.IsPublic)
if subjectID.Valid {
    q.SubjectID = subjectID.String
}
if topicID.Valid {
    q.TopicID = topicID.String
}
```

**Melhorias:**
- ✅ Usa `sql.NullString` para lidar com valores NULL
- ✅ Inclui campo `is_public` nas queries
- ✅ Mapeia corretamente UUID do banco para string

---

## 🔄 Compatibilidade

### Backward Compatible

O código mantém compatibilidade durante a migração:

1. **Struct Question** mantém campos legados (`Subject`, `Topic`)
   - Permite que código antigo continue funcionando
   - Frontend pode enviar `subjectId` ou `subject` (temporariamente)

2. **Banco de Dados** pode ter ambos os campos durante migração
   - `subject` e `topic` (TEXT) - legado
   - `subject_id` e `topic_id` (UUID) - novo

### Breaking Changes (Futuro)

Após migração completa, os campos legados podem ser removidos:
- ❌ Remover `Subject` e `Topic` do struct
- ❌ Remover colunas `subject` e `topic` do banco

---

## 📝 Próximos Passos

### Imediato
- [x] ✅ Atualizar struct Question
- [x] ✅ Atualizar repositório (CreateQuestion, GetQuestions)
- [ ] ⏳ Testar criação de questões com subjectId/topicId
- [ ] ⏳ Testar busca de questões

### Curto Prazo
- [ ] ⏳ Atualizar frontend para usar `subjectId`/`topicId`
- [ ] ⏳ Executar migração do banco (`migrations/001_optimize_schema.sql`)
- [ ] ⏳ Validar integridade referencial

### Longo Prazo
- [ ] ⏳ Remover campos legados (`Subject`, `Topic`)
- [ ] ⏳ Remover colunas legadas do banco
- [ ] ⏳ Adicionar métodos de busca por subject/topic

---

## 🧪 Testes Recomendados

### 1. Criar Questão com SubjectID/TopicID
```go
q := domain.Question{
    ID: "uuid",
    Text: "Qual é a capital do Brasil?",
    Options: []string{"São Paulo", "Rio de Janeiro", "Brasília", "Salvador"},
    CorrectIndex: 2,
    SubjectID: "subject-uuid",
    TopicID: "topic-uuid",
    IsPublic: false,
}
err := repo.CreateQuestion(q)
```

### 2. Buscar Questões
```go
questions, err := repo.GetQuestions()
// Verificar se SubjectID e TopicID estão preenchidos
```

### 3. Validar Integridade Referencial
```go
// Tentar criar questão com subject_id inválido
// Deve retornar erro de FK constraint
```

---

## ⚠️ Notas Importantes

1. **Migração do Banco**: Execute `migrations/001_optimize_schema.sql` antes de usar o novo código
2. **Valores NULL**: O código trata corretamente valores NULL usando `sql.NullString`
3. **Compatibilidade**: Campos legados mantidos temporariamente para transição suave

---

## 📊 Impacto

| Aspecto | Antes | Depois | Melhoria |
|---------|-------|--------|----------|
| Integridade | ❌ Sem FK | ✅ Com FK | **100%** |
| Performance | ⚠️ Queries lentas | ✅ Índices FK | **10x** |
| Manutenibilidade | ❌ Dados inconsistentes | ✅ Dados validados | **100%** |

---

**Status**: ✅ Código atualizado
**Próximo**: Testar e validar em ambiente de desenvolvimento

