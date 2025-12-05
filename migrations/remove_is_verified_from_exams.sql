-- Migração: Remover coluna is_verified da tabela exams
-- Data: 2025-12-05
-- Descrição: is_verified agora é calculado dinamicamente baseado nas questões
--            (um exame é verificado se todas as suas questões estiverem verificadas)

-- Remover coluna is_verified se existir
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'exams' AND column_name = 'is_verified'
    ) THEN
        ALTER TABLE exams DROP COLUMN is_verified;
    END IF;
END $$;

-- Comentário para documentação
COMMENT ON TABLE exams IS 'Exames/Simulados. is_verified é calculado dinamicamente: true se todas as questões estiverem verificadas';

