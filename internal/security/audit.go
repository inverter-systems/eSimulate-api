package security

import (
	"esimulate-backend/internal/logger"
	"time"
)

// SecurityEvent representa um evento de segurança
type SecurityEvent struct {
	Type      string    `json:"type"`
	UserID    string    `json:"userId,omitempty"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"userAgent,omitempty"`
	Details   string    `json:"details,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// AuditLogger registra eventos de segurança
type AuditLogger struct{}

// NewAuditLogger cria um novo audit logger
func NewAuditLogger() *AuditLogger {
	return &AuditLogger{}
}

// LogEvent registra um evento de segurança
func (al *AuditLogger) LogEvent(eventType, userID, ip, userAgent, details string) {
	event := SecurityEvent{
		Type:      eventType,
		UserID:    userID,
		IP:        ip,
		UserAgent: userAgent,
		Details:   details,
		Timestamp: time.Now(),
	}
	
	// Log como JSON estruturado para facilitar análise
	logger.Warn("[SECURITY] %s | User: %s | IP: %s | Details: %s", 
		event.Type, event.UserID, event.IP, event.Details)
}

// LogLogin registra tentativa de login
func (al *AuditLogger) LogLogin(userID, ip, userAgent string, success bool) {
	status := "SUCCESS"
	if !success {
		status = "FAILED"
	}
	al.LogEvent("LOGIN_"+status, userID, ip, userAgent, "")
}

// LogRefresh registra tentativa de refresh token
func (al *AuditLogger) LogRefresh(userID, ip, userAgent string, success bool) {
	status := "SUCCESS"
	if !success {
		status = "FAILED"
	}
	al.LogEvent("REFRESH_"+status, userID, ip, userAgent, "")
}

// LogTokenReuse registra reutilização suspeita de token
func (al *AuditLogger) LogTokenReuse(userID, ip, userAgent string) {
	al.LogEvent("TOKEN_REUSE", userID, ip, userAgent, "Refresh token reutilizado após invalidação")
}

// LogRateLimit registra bloqueio por rate limit
func (al *AuditLogger) LogRateLimit(endpoint, ip string) {
	al.LogEvent("RATE_LIMIT", "", ip, "", "Endpoint: "+endpoint)
}

// LogPasswordReset registra tentativa de reset de senha
func (al *AuditLogger) LogPasswordReset(userID, ip, userAgent string) {
	al.LogEvent("PASSWORD_RESET", userID, ip, userAgent, "")
}

// LogLogout registra logout
func (al *AuditLogger) LogLogout(userID, ip, userAgent string) {
	al.LogEvent("LOGOUT", userID, ip, userAgent, "")
}

