package security

import (
	"net/http"
	"sync"
	"time"
)

// RateLimiter implementa rate limiting simples em memória
type RateLimiter struct {
	requests map[string][]time.Time
	mu       sync.RWMutex
	limits   map[string]RateLimit
}

// RateLimit define limites para diferentes endpoints
type RateLimit struct {
	MaxRequests int
	Window      time.Duration
}

// NewRateLimiter cria um novo rate limiter
func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		limits: map[string]RateLimit{
			"login":           {MaxRequests: 5, Window: 1 * time.Minute},
			"register":        {MaxRequests: 3, Window: 1 * time.Hour},
			"refresh":         {MaxRequests: 10, Window: 1 * time.Minute},
			"forgot-password": {MaxRequests: 3, Window: 1 * time.Hour},
			"verify-email":    {MaxRequests: 5, Window: 1 * time.Minute},
		},
	}
	
	// Limpar requisições antigas periodicamente
	go rl.cleanup()
	
	return rl
}

// Allow verifica se uma requisição é permitida
func (rl *RateLimiter) Allow(key, endpoint string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	limit, exists := rl.limits[endpoint]
	if !exists {
		return true // Sem limite para endpoints não configurados
	}
	
	now := time.Now()
	windowStart := now.Add(-limit.Window)
	
	// Limpar requisições antigas
	requests := rl.requests[key]
	validRequests := []time.Time{}
	for _, reqTime := range requests {
		if reqTime.After(windowStart) {
			validRequests = append(validRequests, reqTime)
		}
	}
	
	// Verificar limite
	if len(validRequests) >= limit.MaxRequests {
		return false
	}
	
	// Adicionar nova requisição
	validRequests = append(validRequests, now)
	rl.requests[key] = validRequests
	
	return true
}

// cleanup remove requisições antigas periodicamente
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, requests := range rl.requests {
			validRequests := []time.Time{}
			for _, reqTime := range requests {
				// Manter apenas requisições dos últimos 10 minutos
				if reqTime.After(now.Add(-10 * time.Minute)) {
					validRequests = append(validRequests, reqTime)
				}
			}
			if len(validRequests) == 0 {
				delete(rl.requests, key)
			} else {
				rl.requests[key] = validRequests
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimitMiddleware cria um middleware de rate limiting
func RateLimitMiddleware(rl *RateLimiter, endpoint string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Usar IP como chave
			key := getClientIP(r) + ":" + endpoint
			
			if !rl.Allow(key, endpoint) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error": "Muitas requisições. Tente novamente mais tarde."}`))
				return
			}
			
			next(w, r)
		}
	}
}

// getClientIP extrai o IP real do cliente
func getClientIP(r *http.Request) string {
	// Verificar X-Forwarded-For (proxy/load balancer)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return forwarded
	}
	
	// Verificar X-Real-IP
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}
	
	// Usar RemoteAddr como fallback
	return r.RemoteAddr
}

