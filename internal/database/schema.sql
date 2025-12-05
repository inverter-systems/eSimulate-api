-- ============================================
-- SCHEMA DO BANCO DE DADOS - eSimulate API
-- PostgreSQL com otimizações de performance e normalização
-- ============================================

-- Extensão para geração de UUIDs
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================
-- 1. TABELA DE USUÁRIOS
-- ============================================
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'user', -- Valores: 'admin', 'user', 'company', 'specialist'
    provider TEXT DEFAULT 'email', -- Provedor de autenticação: 'email', 'google', 'github'
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    profile JSONB DEFAULT '{}', -- Dados adicionais do perfil (CPF, empresa, telefone, endereço)
    is_verified BOOLEAN DEFAULT FALSE, -- Indica se o email foi verificado
    onboarding_completed BOOLEAN DEFAULT FALSE -- Indica se o fluxo de onboarding foi concluído
);

-- Índices para users
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_users_profile_gin ON users USING GIN(profile);

-- ============================================
-- 2. TABELA DE MATÉRIAS (SUBJECTS)
-- ============================================
CREATE TABLE IF NOT EXISTS subjects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT UNIQUE NOT NULL, -- Nome da matéria/disciplina (ex: Matemática, Português)
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Índices para subjects
CREATE INDEX IF NOT EXISTS idx_subjects_name ON subjects(name);

-- ============================================
-- 3. TABELA DE TÓPICOS (TOPICS)
-- ============================================
CREATE TABLE IF NOT EXISTS topics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    name TEXT NOT NULL, -- Nome do tópico dentro da matéria
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_topic_per_subject UNIQUE(subject_id, name) -- Evita tópicos duplicados na mesma matéria
);

-- Índices para topics
CREATE INDEX IF NOT EXISTS idx_topics_subject_id ON topics(subject_id);
CREATE INDEX IF NOT EXISTS idx_topics_name ON topics(name);

-- ============================================
-- 4. TABELA DE QUESTÕES (QUESTIONS)
-- ============================================
CREATE TABLE IF NOT EXISTS questions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    text TEXT NOT NULL, -- Enunciado da questão
    options JSONB NOT NULL, -- Array de strings com as opções de resposta
    correct_index INT NOT NULL CHECK (correct_index >= 0), -- Índice da resposta correta (0-based)
    explanation TEXT, -- Explicação da resposta correta
    subject_id UUID REFERENCES subjects(id) ON DELETE SET NULL, -- Matéria relacionada (FK)
    topic_id UUID REFERENCES topics(id) ON DELETE SET NULL, -- Tópico relacionado (FK)
    is_public BOOLEAN DEFAULT FALSE, -- Indica se a questão é pública
    is_verified BOOLEAN DEFAULT FALSE, -- Indica se a questão foi verificada por admin/specialist
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Índices para questions
CREATE INDEX IF NOT EXISTS idx_questions_subject_id ON questions(subject_id);
CREATE INDEX IF NOT EXISTS idx_questions_topic_id ON questions(topic_id);
CREATE INDEX IF NOT EXISTS idx_questions_is_public ON questions(is_public);
CREATE INDEX IF NOT EXISTS idx_questions_options_gin ON questions USING GIN(options);
CREATE INDEX IF NOT EXISTS idx_questions_created_at ON questions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_questions_subject_topic ON questions(subject_id, topic_id) 
    WHERE subject_id IS NOT NULL AND topic_id IS NOT NULL;

-- ============================================
-- 5. TABELA DE SIMULADOS (EXAMS)
-- ============================================
CREATE TABLE IF NOT EXISTS exams (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title TEXT NOT NULL, -- Título do simulado
    description TEXT, -- Descrição do simulado
    -- Campo questions JSONB removido - usando apenas tabela exam_questions para relacionamento N:N
    subjects JSONB, -- Array de nomes de matérias (mantido para performance e compatibilidade)
    time_limit INT, -- Tempo limite em minutos (opcional)
    is_public BOOLEAN DEFAULT FALSE, -- Indica se o exame é público
    -- is_verified removido: calculado no frontend baseado nas questões (todas devem estar verificadas)
    created_by UUID REFERENCES users(id) ON DELETE SET NULL, -- Usuário que criou o exame
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_active BOOLEAN DEFAULT TRUE -- Soft delete: FALSE para exames desativados
);

-- Índices para exams
CREATE INDEX IF NOT EXISTS idx_exams_created_by ON exams(created_by);
CREATE INDEX IF NOT EXISTS idx_exams_created_at ON exams(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_exams_is_active ON exams(is_active);
-- Índice GIN para questions removido (campo removido - usando exam_questions)
CREATE INDEX IF NOT EXISTS idx_exams_subjects_gin ON exams USING GIN(subjects);

-- ============================================
-- 6. TABELA DE RELACIONAMENTO EXAM-QUESTIONS (N:N)
-- ============================================
-- Relacionamento many-to-many normalizado entre exames e questões
CREATE TABLE IF NOT EXISTS exam_questions (
    exam_id UUID NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    question_id UUID NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    PRIMARY KEY (exam_id, question_id)
);

-- Índices para exam_questions
CREATE INDEX IF NOT EXISTS idx_exam_questions_exam_id ON exam_questions(exam_id);
CREATE INDEX IF NOT EXISTS idx_exam_questions_question_id ON exam_questions(question_id);

-- ============================================
-- 7. TABELA DE RESULTADOS (RESULTS)
-- ============================================
CREATE TABLE IF NOT EXISTS results (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    exam_id UUID NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE, -- NULL para candidatos públicos (sem login)
    candidate_name TEXT, -- Nome do candidato (para acesso público)
    candidate_email TEXT, -- Email do candidato (para acesso público)
    score INT NOT NULL CHECK (score >= 0), -- Pontuação obtida
    total_questions INT NOT NULL CHECK (total_questions > 0), -- Total de questões do exame
    answers JSONB NOT NULL, -- Array de objetos: [{questionId, selectedIndex, isCorrect}]
    time_spent_seconds INT NOT NULL CHECK (time_spent_seconds >= 0), -- Tempo gasto em segundos
    date TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- Data/hora da realização
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Índices para results
CREATE INDEX IF NOT EXISTS idx_results_exam_id ON results(exam_id);
CREATE INDEX IF NOT EXISTS idx_results_user_id ON results(user_id);
CREATE INDEX IF NOT EXISTS idx_results_date ON results(date DESC);
CREATE INDEX IF NOT EXISTS idx_results_candidate_email ON results(candidate_email) 
    WHERE candidate_email IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_results_answers_gin ON results USING GIN(answers);
CREATE INDEX IF NOT EXISTS idx_results_user_date ON results(user_id, date DESC) 
    WHERE user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_results_exam_date ON results(exam_id, date DESC);

-- ============================================
-- 8. TABELA DE TOKENS DE VERIFICAÇÃO E RESET
-- ============================================
CREATE TABLE IF NOT EXISTS tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    token TEXT UNIQUE NOT NULL,
    type TEXT NOT NULL, -- 'verification' | 'password_reset'
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    used BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tokens_token ON tokens(token);
CREATE INDEX IF NOT EXISTS idx_tokens_user_id ON tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_tokens_type ON tokens(type);
CREATE INDEX IF NOT EXISTS idx_tokens_expires_at ON tokens(expires_at);

-- ============================================
-- 9. TABELA DE LINKS PÚBLICOS (PUBLIC_LINKS)
-- ============================================
CREATE TABLE IF NOT EXISTS public_links (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    exam_id UUID NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    company_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE, -- Empresa que criou o link
    token TEXT UNIQUE NOT NULL, -- Token único para acesso público ao exame
    label TEXT, -- Rótulo/descrição do link
    active BOOLEAN DEFAULT TRUE, -- Indica se o link está ativo
    expires_at TIMESTAMP WITH TIME ZONE, -- Data de expiração do link (opcional)
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Índices para public_links
CREATE INDEX IF NOT EXISTS idx_public_links_token ON public_links(token);
CREATE INDEX IF NOT EXISTS idx_public_links_exam_id ON public_links(exam_id);
CREATE INDEX IF NOT EXISTS idx_public_links_company_id ON public_links(company_id);
CREATE INDEX IF NOT EXISTS idx_public_links_active ON public_links(active);
CREATE INDEX IF NOT EXISTS idx_public_links_active_token ON public_links(active, token) 
    WHERE active = TRUE;

-- ============================================
-- 10. TRIGGERS PARA ATUALIZAÇÃO AUTOMÁTICA
-- ============================================
-- Função para atualizar updated_at automaticamente
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers para atualizar updated_at em updates
-- Drop se existir antes de criar (evita erro se já existir)
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_questions_updated_at ON questions;
CREATE TRIGGER update_questions_updated_at BEFORE UPDATE ON questions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_exams_updated_at ON exams;
CREATE TRIGGER update_exams_updated_at BEFORE UPDATE ON exams
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- 11. VIEWS ÚTEIS
-- ============================================
-- View para estatísticas de exames
CREATE OR REPLACE VIEW exam_stats AS
SELECT 
    e.id,
    e.title,
    e.created_at,
    COUNT(DISTINCT r.id) as total_results,
    COUNT(DISTINCT r.user_id) FILTER (WHERE r.user_id IS NOT NULL) as total_users,
    AVG(r.score::FLOAT / NULLIF(r.total_questions, 0) * 100) as avg_score_percentage,
    COUNT(DISTINCT pl.id) as total_public_links
FROM exams e
LEFT JOIN results r ON e.id = r.exam_id
LEFT JOIN public_links pl ON e.id = pl.exam_id
WHERE e.is_active = TRUE
GROUP BY e.id, e.title, e.created_at;

-- ============================================
-- 12. COMENTÁRIOS PARA DOCUMENTAÇÃO
-- ============================================
COMMENT ON TABLE users IS 'Usuários do sistema com diferentes roles (admin, user, company)';
COMMENT ON TABLE subjects IS 'Matérias/disciplinas disponíveis no sistema';
COMMENT ON TABLE topics IS 'Tópicos específicos dentro de cada matéria';
COMMENT ON TABLE questions IS 'Banco de questões reutilizáveis que podem ser usadas em múltiplos exames';
COMMENT ON TABLE exams IS 'Simulados/provas criados pelos usuários com relacionamento N:N para questões via exam_questions';
COMMENT ON TABLE results IS 'Resultados de execução de exames com suporte a candidatos públicos e usuários autenticados';
COMMENT ON TABLE public_links IS 'Links públicos gerados por empresas para acesso externo a exames (B2B)';

COMMENT ON COLUMN users.profile IS 'Dados adicionais do perfil em formato JSONB (CPF, empresa, telefone, endereço)';
COMMENT ON COLUMN users.role IS 'Papel do usuário no sistema: admin (administrador), user (usuário comum), company (empresa)';
COMMENT ON COLUMN questions.options IS 'Array JSONB de strings com as opções de resposta da questão';
COMMENT ON COLUMN questions.correct_index IS 'Índice baseado em zero da opção correta no array options';
COMMENT ON COLUMN exams.subjects IS 'Array JSONB de nomes de matérias, mantido para performance e compatibilidade';
COMMENT ON COLUMN results.answers IS 'Array JSONB de objetos com as respostas: [{questionId, selectedIndex, isCorrect}]';
COMMENT ON COLUMN results.user_id IS 'ID do usuário autenticado ou NULL para candidatos públicos que acessaram via link';
COMMENT ON COLUMN public_links.token IS 'Token único e seguro para acesso público ao exame sem necessidade de autenticação';
COMMENT ON COLUMN public_links.expires_at IS 'Data e hora de expiração do link público (NULL = sem expiração)';
