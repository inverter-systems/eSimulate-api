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
    role TEXT NOT NULL DEFAULT 'user',
    provider TEXT DEFAULT 'email',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    profile JSONB DEFAULT '{}',
    is_verified BOOLEAN DEFAULT FALSE,
    onboarding_completed BOOLEAN DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_users_profile_gin ON users USING GIN(profile);

-- ============================================
-- 2. TABELA DE MATÉRIAS (SUBJECTS)
-- ============================================
CREATE TABLE IF NOT EXISTS subjects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_subjects_name ON subjects(name);

-- ============================================
-- 3. TABELA DE TÓPICOS (TOPICS)
-- ============================================
CREATE TABLE IF NOT EXISTS topics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_topic_per_subject UNIQUE(subject_id, name)
);

CREATE INDEX IF NOT EXISTS idx_topics_subject_id ON topics(subject_id);
CREATE INDEX IF NOT EXISTS idx_topics_name ON topics(name);

-- ============================================
-- 4. TABELA DE QUESTÕES (QUESTIONS)
-- ============================================
CREATE TABLE IF NOT EXISTS questions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    text TEXT NOT NULL,
    options JSONB NOT NULL,
    correct_index INT NOT NULL CHECK (correct_index >= 0),
    explanation TEXT,
    subject_id UUID REFERENCES subjects(id) ON DELETE SET NULL,
    topic_id UUID REFERENCES topics(id) ON DELETE SET NULL,
    is_public BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

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
    title TEXT NOT NULL,
    description TEXT,
    questions JSONB NOT NULL,
    subjects JSONB,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_active BOOLEAN DEFAULT TRUE
);

CREATE INDEX IF NOT EXISTS idx_exams_created_by ON exams(created_by);
CREATE INDEX IF NOT EXISTS idx_exams_created_at ON exams(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_exams_is_active ON exams(is_active);
CREATE INDEX IF NOT EXISTS idx_exams_questions_gin ON exams USING GIN(questions);
CREATE INDEX IF NOT EXISTS idx_exams_subjects_gin ON exams USING GIN(subjects);

-- ============================================
-- 6. TABELA DE RELACIONAMENTO EXAM-SUBJECTS
-- ============================================
CREATE TABLE IF NOT EXISTS exam_subjects (
    exam_id UUID NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    PRIMARY KEY (exam_id, subject_id)
);

CREATE INDEX IF NOT EXISTS idx_exam_subjects_exam_id ON exam_subjects(exam_id);
CREATE INDEX IF NOT EXISTS idx_exam_subjects_subject_id ON exam_subjects(subject_id);

-- ============================================
-- 7. TABELA DE RESULTADOS (RESULTS)
-- ============================================
CREATE TABLE IF NOT EXISTS results (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    exam_id UUID NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    candidate_name TEXT,
    candidate_email TEXT,
    score INT NOT NULL CHECK (score >= 0),
    total_questions INT NOT NULL CHECK (total_questions > 0),
    answers JSONB NOT NULL,
    time_spent_seconds INT NOT NULL CHECK (time_spent_seconds >= 0),
    date TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

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
-- 8. TABELA DE LINKS PÚBLICOS (PUBLIC_LINKS)
-- ============================================
CREATE TABLE IF NOT EXISTS public_links (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    exam_id UUID NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    company_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT UNIQUE NOT NULL,
    label TEXT,
    active BOOLEAN DEFAULT TRUE,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_public_links_token ON public_links(token);
CREATE INDEX IF NOT EXISTS idx_public_links_exam_id ON public_links(exam_id);
CREATE INDEX IF NOT EXISTS idx_public_links_company_id ON public_links(company_id);
CREATE INDEX IF NOT EXISTS idx_public_links_active ON public_links(active);
CREATE INDEX IF NOT EXISTS idx_public_links_active_token ON public_links(active, token) 
    WHERE active = TRUE;

-- ============================================
-- 9. TRIGGERS PARA ATUALIZAÇÃO AUTOMÁTICA
-- ============================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_questions_updated_at BEFORE UPDATE ON questions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_exams_updated_at BEFORE UPDATE ON exams
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- 10. VIEWS ÚTEIS
-- ============================================
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
COMMENT ON TABLE users IS 'Usuários do sistema com diferentes roles (admin, user, company)';
COMMENT ON TABLE subjects IS 'Matérias/disciplinas disponíveis no sistema';
COMMENT ON TABLE topics IS 'Tópicos específicos dentro de cada matéria';
COMMENT ON TABLE questions IS 'Banco de questões reutilizáveis que podem ser usadas em múltiplos exames';
COMMENT ON TABLE exams IS 'Simulados/provas criados pelos usuários com snapshot imutável das questões';
COMMENT ON TABLE exam_subjects IS 'Relacionamento many-to-many normalizado entre exames e matérias';
COMMENT ON TABLE results IS 'Resultados de execução de exames com suporte a candidatos públicos e usuários autenticados';
COMMENT ON TABLE public_links IS 'Links públicos gerados por empresas para acesso externo a exames (B2B)';

COMMENT ON COLUMN users.profile IS 'Dados adicionais do perfil em formato JSONB (CPF, empresa, telefone, endereço)';
COMMENT ON COLUMN users.role IS 'Papel do usuário no sistema: admin (administrador), user (usuário comum), company (empresa)';
COMMENT ON COLUMN questions.options IS 'Array JSONB de strings com as opções de resposta da questão';
COMMENT ON COLUMN questions.correct_index IS 'Índice baseado em zero da opção correta no array options';
COMMENT ON COLUMN exams.questions IS 'Snapshot JSONB das questões no momento da criação do exame, mantendo histórico imutável';
COMMENT ON COLUMN exams.subjects IS 'Array JSONB de nomes de matérias, mantido para performance e compatibilidade';
COMMENT ON COLUMN results.answers IS 'Array JSONB de objetos com as respostas: [{questionId, selectedIndex, isCorrect}]';
COMMENT ON COLUMN results.user_id IS 'ID do usuário autenticado ou NULL para candidatos públicos que acessaram via link';
COMMENT ON COLUMN public_links.token IS 'Token único e seguro para acesso público ao exame sem necessidade de autenticação';
COMMENT ON COLUMN public_links.expires_at IS 'Data e hora de expiração do link público (NULL = sem expiração)';
