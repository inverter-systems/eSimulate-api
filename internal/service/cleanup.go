package service

import (
	"esimulate-backend/internal/logger"
	"esimulate-backend/internal/repository/postgres"
	"time"
)

// CleanupService gerencia limpeza automática de dados expirados
type CleanupService struct {
	repo      *postgres.PostgresRepo
	hour      int // Hora do dia para executar (0-23)
	stopChan  chan bool
	running   bool
}

// NewCleanupService cria um novo serviço de limpeza
// hour: hora do dia para executar a limpeza (0-23), padrão 3 (3h da manhã)
func NewCleanupService(repo *postgres.PostgresRepo, hour int) *CleanupService {
	if hour < 0 || hour > 23 {
		hour = 3 // Padrão: 3h da manhã
	}
	return &CleanupService{
		repo:     repo,
		hour:     hour,
		stopChan: make(chan bool),
		running:   false,
	}
}

// Start inicia o serviço de limpeza em background
// Executa uma vez por dia no horário especificado
func (c *CleanupService) Start() {
	if c.running {
		logger.Warn("CleanupService já está em execução")
		return
	}

	c.running = true
	logger.Info("CleanupService iniciado | Agendado para rodar diariamente às %02d:00", c.hour)

	go func() {
		for {
			// Calcular próximo horário de execução
			now := time.Now()
			nextRun := time.Date(now.Year(), now.Month(), now.Day(), c.hour, 0, 0, 0, now.Location())
			
			// Se já passou o horário de hoje, agendar para amanhã
			if now.After(nextRun) || now.Equal(nextRun) {
				nextRun = nextRun.Add(24 * time.Hour)
			}
			
			waitDuration := time.Until(nextRun)
			logger.Debug("Próxima limpeza agendada para: %s (em %v)", nextRun.Format("2006-01-02 15:04:05"), waitDuration)

			// Aguardar até o horário agendado
			timer := time.NewTimer(waitDuration)
			
			select {
			case <-timer.C:
				c.runCleanup()
			case <-c.stopChan:
				timer.Stop()
				logger.Info("CleanupService parado")
				return
			}
		}
	}()
}

// Stop para o serviço de limpeza
func (c *CleanupService) Stop() {
	if !c.running {
		return
	}
	c.running = false
	close(c.stopChan)
}

// runCleanup executa a limpeza de dados expirados
func (c *CleanupService) runCleanup() {
	logger.Info("Executando limpeza automática de dados expirados...")
	
	start := time.Now()
	err := c.repo.CleanupExpiredData()
	duration := time.Since(start)

	if err != nil {
		logger.Error("Erro na limpeza de dados expirados: %v", err)
	} else {
		logger.Info("Limpeza automática concluída com sucesso | Duração: %v", duration)
	}
}

