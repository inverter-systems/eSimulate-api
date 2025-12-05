package http

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"esimulate-backend/internal/security"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// CORSMiddleware com configuração restritiva
func CORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			
			// Em desenvolvimento, permitir localhost
			if len(allowedOrigins) == 0 {
				allowedOrigins = []string{"*"} // Fallback para desenvolvimento
			}
			
			// Verificar se origin está permitida
			allowed := false
			for _, allowedOrigin := range allowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					break
				}
			}
			
			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
				w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Max-Age", "3600")
			}
			
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// HTTPSMiddleware força HTTPS em produção
func HTTPSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Apenas em produção
		if os.Getenv("ENV") == "production" || os.Getenv("ENV") == "prod" {
			// Verificar se já é HTTPS
			if r.TLS == nil {
				// Verificar X-Forwarded-Proto (para proxies/load balancers)
				if r.Header.Get("X-Forwarded-Proto") != "https" {
					httpsURL := "https://" + r.Host + r.RequestURI
					http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
					return
				}
			}
			
			// HSTS Header
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		
		next.ServeHTTP(w, r)
	})
}

// AuthMiddleware com validação explícita de exp e blacklist
func AuthMiddleware(secret string, blacklist *security.TokenBlacklist) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Credenciais inválidas", 401)
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			
			// Gerar hash do token para blacklist
			hash := sha256.Sum256([]byte(tokenStr))
			tokenID := hex.EncodeToString(hash[:])
			
			// Verificar blacklist
			if blacklist != nil && blacklist.IsBlacklisted(tokenID) {
				http.Error(w, "Credenciais inválidas", 401)
				return
			}
			
			claims := jwt.MapClaims{}
			token, err := jwt.ParseWithClaims(tokenStr, &claims, func(token *jwt.Token) (interface{}, error) {
				// Validar algoritmo
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(secret), nil
			})

			if err != nil || !token.Valid {
				http.Error(w, "Credenciais inválidas", 401)
				return
			}
			
			// Validação explícita de exp
			exp, ok := claims["exp"].(float64)
			if !ok {
				http.Error(w, "Credenciais inválidas", 401)
				return
			}
			
			expTime := time.Unix(int64(exp), 0)
			now := time.Now()
			
			// Tolerância de 5 minutos para clock skew
			if now.After(expTime.Add(5 * time.Minute)) {
				http.Error(w, "Credenciais inválidas", 401)
				return
			}

			ctx := context.WithValue(r.Context(), "userID", claims["user_id"])
			if role, ok := claims["role"].(string); ok {
				ctx = context.WithValue(ctx, "role", role)
			}
			// Adicionar tokenID ao context para uso no logout
			ctx = context.WithValue(ctx, "tokenID", tokenID)
			
			next(w, r.WithContext(ctx))
		}
	}
}

// CSRFMiddleware - Removido: SameSite=Strict nos cookies já fornece proteção adequada
// Se necessário no futuro, pode ser implementado com tokens CSRF
