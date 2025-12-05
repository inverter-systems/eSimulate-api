-- Migração: Adicionar campo is_verified na tabela exams
-- Data: 2025-12-05
-- Descrição: Adiciona o campo is_verified conforme contrato FRONTEND_CONTRACT_API.md

-- Adicionar coluna is_verified se não existir
DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'exams' AND column_name = 'is_verified'
    ) THEN
        ALTER TABLE exams ADD COLUMN is_verified BOOLEAN DEFAULT FALSE;
        COMMENT ON COLUMN exams.is_verified IS 'Indica se o exame foi verificado (admin/specialist podem definir)';
    END IF;
END $$;

