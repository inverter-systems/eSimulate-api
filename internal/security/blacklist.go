package security

import (
	"sync"
	"time"
)

// TokenBlacklist gerencia tokens revogados
type TokenBlacklist struct {
	tokens map[string]time.Time
	mu     sync.RWMutex
}

// NewTokenBlacklist cria um novo blacklist
func NewTokenBlacklist() *TokenBlacklist {
	bl := &TokenBlacklist{
		tokens: make(map[string]time.Time),
	}
	
	// Limpar tokens expirados periodicamente
	go bl.cleanup()
	
	return bl
}

// Add adiciona um token ao blacklist com tempo de expiração
func (bl *TokenBlacklist) Add(tokenID string, expiresAt time.Time) {
	bl.mu.Lock()
	defer bl.mu.Unlock()
	bl.tokens[tokenID] = expiresAt
}

// IsBlacklisted verifica se um token está no blacklist
func (bl *TokenBlacklist) IsBlacklisted(tokenID string) bool {
	bl.mu.RLock()
	defer bl.mu.RUnlock()
	
	expiresAt, exists := bl.tokens[tokenID]
	if !exists {
		return false
	}
	
	// Se expirou, remover e retornar false
	if time.Now().After(expiresAt) {
		bl.mu.RUnlock()
		bl.mu.Lock()
		delete(bl.tokens, tokenID)
		bl.mu.Unlock()
		bl.mu.RLock()
		return false
	}
	
	return true
}

// cleanup remove tokens expirados periodicamente
func (bl *TokenBlacklist) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		bl.mu.Lock()
		now := time.Now()
		for tokenID, expiresAt := range bl.tokens {
			if now.After(expiresAt) {
				delete(bl.tokens, tokenID)
			}
		}
		bl.mu.Unlock()
	}
}

