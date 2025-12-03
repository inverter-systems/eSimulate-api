-- ============================================
-- SCHEMA OTIMIZADO - eSimulate API
-- Melhorias: Normalização, Índices, Performance
-- ============================================

-- Extensão para UUID
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================
-- 1. TABELA DE USUÁRIOS
-- ============================================
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'user', -- 'admin', 'user', 'company'
    provider TEXT DEFAULT 'email',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    profile JSONB DEFAULT '{}',
    is_verified BOOLEAN DEFAULT FALSE,
    onboarding_completed BOOLEAN DEFAULT FALSE
);

-- Índices para users
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email); -- Já tem UNIQUE, mas índice explícito ajuda
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_users_profile_gin ON users USING GIN(profile); -- Para queries JSONB

-- ============================================
-- 2. TABELA DE MATÉRIAS (SUBJECTS)
-- ============================================
CREATE TABLE IF NOT EXISTS subjects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Índice para subjects
CREATE INDEX IF NOT EXISTS idx_subjects_name ON subjects(name);

-- ============================================
-- 3. TABELA DE TÓPICOS (TOPICS)
-- ============================================
CREATE TABLE IF NOT EXISTS topics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_topic_per_subject UNIQUE(subject_id, name) -- Evita duplicatas
);

-- Índices para topics
CREATE INDEX IF NOT EXISTS idx_topics_subject_id ON topics(subject_id);
CREATE INDEX IF NOT EXISTS idx_topics_name ON topics(name);

-- ============================================
-- 4. TABELA DE QUESTÕES (QUESTIONS)
-- ============================================
-- MELHORIA: subject_id e topic_id agora são FK ao invés de TEXT
CREATE TABLE IF NOT EXISTS questions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    text TEXT NOT NULL,
    options JSONB NOT NULL, -- Array de strings armazenado como JSON
    correct_index INT NOT NULL CHECK (correct_index >= 0),
    explanation TEXT,
    subject_id UUID REFERENCES subjects(id) ON DELETE SET NULL, -- FK ao invés de TEXT
    topic_id UUID REFERENCES topics(id) ON DELETE SET NULL, -- FK ao invés de TEXT
    is_public BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Índices para questions
CREATE INDEX IF NOT EXISTS idx_questions_subject_id ON questions(subject_id);
CREATE INDEX IF NOT EXISTS idx_questions_topic_id ON questions(topic_id);
CREATE INDEX IF NOT EXISTS idx_questions_is_public ON questions(is_public);
CREATE INDEX IF NOT EXISTS idx_questions_options_gin ON questions USING GIN(options); -- Para queries JSONB
CREATE INDEX IF NOT EXISTS idx_questions_created_at ON questions(created_at DESC);

-- Índice composto para queries comuns
CREATE INDEX IF NOT EXISTS idx_questions_subject_topic ON questions(subject_id, topic_id) WHERE subject_id IS NOT NULL AND topic_id IS NOT NULL;

-- ============================================
-- 5. TABELA DE SIMULADOS (EXAMS)
-- ============================================
-- NOTA: questions JSONB é snapshot (imutável) - OK para histórico
-- subjects JSONB pode ser mantido para performance, mas também criamos tabela de relacionamento
CREATE TABLE IF NOT EXISTS exams (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title TEXT NOT NULL,
    description TEXT,
    questions JSONB NOT NULL, -- Snapshot das questões (imutável após criação)
    subjects JSONB, -- Lista de matérias (para performance/legacy)
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_active BOOLEAN DEFAULT TRUE -- Para soft delete
);

-- Índices para exams
CREATE INDEX IF NOT EXISTS idx_exams_created_by ON exams(created_by);
CREATE INDEX IF NOT EXISTS idx_exams_created_at ON exams(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_exams_is_active ON exams(is_active);
CREATE INDEX IF NOT EXISTS idx_exams_title_trgm ON exams USING GIN(title gin_trgm_ops); -- Para busca full-text (requer pg_trgm)
CREATE INDEX IF NOT EXISTS idx_exams_questions_gin ON exams USING GIN(questions);
CREATE INDEX IF NOT EXISTS idx_exams_subjects_gin ON exams USING GIN(subjects);

-- ============================================
-- 6. TABELA DE RELACIONAMENTO EXAM-SUBJECTS
-- ============================================
-- MELHORIA: Normalização - relacionamento many-to-many
CREATE TABLE IF NOT EXISTS exam_subjects (
    exam_id UUID NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    PRIMARY KEY (exam_id, subject_id)
);

-- Índices para exam_subjects
CREATE INDEX IF NOT EXISTS idx_exam_subjects_exam_id ON exam_subjects(exam_id);
CREATE INDEX IF NOT EXISTS idx_exam_subjects_subject_id ON exam_subjects(subject_id);

-- ============================================
-- 7. TABELA DE RESULTADOS (RESULTS)
-- ============================================
CREATE TABLE IF NOT EXISTS results (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    exam_id UUID NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE, -- NULL para candidatos públicos
    candidate_name TEXT, 
    candidate_email TEXT,
    score INT NOT NULL CHECK (score >= 0),
    total_questions INT NOT NULL CHECK (total_questions > 0),
    answers JSONB NOT NULL, -- Array de objetos {questionId, selectedIndex, isCorrect}
    time_spent_seconds INT NOT NULL CHECK (time_spent_seconds >= 0),
    date TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Índices para results
CREATE INDEX IF NOT EXISTS idx_results_exam_id ON results(exam_id);
CREATE INDEX IF NOT EXISTS idx_results_user_id ON results(user_id);
CREATE INDEX IF NOT EXISTS idx_results_date ON results(date DESC);
CREATE INDEX IF NOT EXISTS idx_results_candidate_email ON results(candidate_email) WHERE candidate_email IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_results_answers_gin ON results USING GIN(answers); -- Para queries JSONB

-- Índices compostos para queries comuns
CREATE INDEX IF NOT EXISTS idx_results_user_date ON results(user_id, date DESC) WHERE user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_results_exam_date ON results(exam_id, date DESC);

-- ============================================
-- 8. TABELA DE LINKS PÚBLICOS (PUBLIC_LINKS)
-- ============================================
CREATE TABLE IF NOT EXISTS public_links (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    exam_id UUID NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    company_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT UNIQUE NOT NULL,
    label TEXT,
    active BOOLEAN DEFAULT TRUE,
    expires_at TIMESTAMP WITH TIME ZONE, -- Nova: expiração opcional
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Índices para public_links
CREATE INDEX IF NOT EXISTS idx_public_links_token ON public_links(token); -- Já tem UNIQUE, mas índice explícito
CREATE INDEX IF NOT EXISTS idx_public_links_exam_id ON public_links(exam_id);
CREATE INDEX IF NOT EXISTS idx_public_links_company_id ON public_links(company_id);
CREATE INDEX IF NOT EXISTS idx_public_links_active ON public_links(active);
CREATE INDEX IF NOT EXISTS idx_public_links_active_token ON public_links(active, token) WHERE active = TRUE;

-- ============================================
-- 9. TRIGGERS PARA UPDATED_AT
-- ============================================
-- Função para atualizar updated_at automaticamente
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers para atualizar updated_at
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_questions_updated_at BEFORE UPDATE ON questions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_exams_updated_at BEFORE UPDATE ON exams
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- 10. VIEWS ÚTEIS (OPCIONAL)
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
-- 11. COMENTÁRIOS PARA DOCUMENTAÇÃO
-- ============================================
COMMENT ON TABLE users IS 'Usuários do sistema (admin, user, company)';
COMMENT ON TABLE subjects IS 'Matérias/disciplinas disponíveis';
COMMENT ON TABLE topics IS 'Tópicos dentro de cada matéria';
COMMENT ON TABLE questions IS 'Banco de questões (pode ser reutilizado em múltiplos exames)';
COMMENT ON TABLE exams IS 'Simulados/provas criados (questions é snapshot imutável)';
COMMENT ON TABLE exam_subjects IS 'Relacionamento many-to-many entre exames e matérias';
COMMENT ON TABLE results IS 'Resultados de execução de exames (LGPD: cascade delete)';
COMMENT ON TABLE public_links IS 'Links públicos para acesso externo (B2B)';

COMMENT ON COLUMN exams.questions IS 'Snapshot JSONB das questões no momento da criação (imutável)';
COMMENT ON COLUMN exams.subjects IS 'Array JSONB de matérias (mantido para performance/legacy)';
COMMENT ON COLUMN results.answers IS 'Array JSONB: [{questionId, selectedIndex, isCorrect}]';
COMMENT ON COLUMN results.user_id IS 'NULL para candidatos públicos (sem login)';

