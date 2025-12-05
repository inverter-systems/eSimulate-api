package main

import (
	"database/sql"
	"esimulate-backend/internal/config"
	"esimulate-backend/internal/delivery/http"
	"esimulate-backend/internal/logger"
	"esimulate-backend/internal/repository/postgres"
	"esimulate-backend/internal/security"
	"esimulate-backend/internal/service"
	"os"
	"strings"
	httpNet "net/http"
)

func main() {
	// 1. Config
	cfg := config.LoadConfig()

	// 2. Database
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		logger.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		logger.Fatal("Cannot connect to DB:", err)
	}
	logger.Info("Connected to PostgreSQL")

	// 3. Init Schema (Simple migration for MVP)
	// Em produção, usar 'golang-migrate'
	runMigration(db)

	// 4. Layers Init
	repo := postgres.NewPostgresRepo(db)
	svc := service.NewService(repo, cfg)
	
	// 5. Initialize Admin User
	if err := svc.InitializeAdmin(); err != nil {
		logger.Warn("Failed to initialize admin user: %v", err)
	} else {
		logger.Info("Admin user ready: %s", cfg.AdminEmail)
	}
	
	// 6. Start Cleanup Service (limpeza automática de dados expirados)
	// Limpa tokens e links expirados diariamente às 3h da manhã
	cleanupService := service.NewCleanupService(repo, 3)
	cleanupService.Start()
	defer cleanupService.Stop()
	
	// 7. Inicializar componentes de segurança
	rateLimiter := security.NewRateLimiter()
	auditLogger := security.NewAuditLogger()
	tokenBlacklist := security.NewTokenBlacklist()
	
	h := http.NewHandler(svc, rateLimiter, auditLogger, tokenBlacklist)

	// 8. Router (Go 1.22)
	mux := httpNet.NewServeMux()

	// Auth com rate limiting
	loginRateLimit := security.RateLimitMiddleware(rateLimiter, "login")
	registerRateLimit := security.RateLimitMiddleware(rateLimiter, "register")
	refreshRateLimit := security.RateLimitMiddleware(rateLimiter, "refresh")
	forgotRateLimit := security.RateLimitMiddleware(rateLimiter, "forgot-password")
	verifyRateLimit := security.RateLimitMiddleware(rateLimiter, "verify-email")
	
	mux.HandleFunc("POST /api/auth/register", registerRateLimit(h.Register))
	mux.HandleFunc("POST /api/auth/login", loginRateLimit(h.Login))
	mux.HandleFunc("POST /api/auth/refresh", refreshRateLimit(h.RefreshToken))
	mux.HandleFunc("POST /api/auth/logout", h.Logout)
	// Auth Recovery
	mux.HandleFunc("POST /api/auth/forgot-password", forgotRateLimit(h.ForgotPassword))
	mux.HandleFunc("POST /api/auth/reset-password", h.ResetPassword)
	mux.HandleFunc("POST /api/auth/verify-email", verifyRateLimit(h.VerifyEmail))

	// Protected Routes Helper com blacklist
	protect := func(handler httpNet.HandlerFunc) httpNet.HandlerFunc {
		return http.AuthMiddleware(cfg.JWTSecret, tokenBlacklist)(handler)
	}

	// Exams
	mux.HandleFunc("GET /api/exams", protect(h.GetExams))
	mux.HandleFunc("GET /api/exams/{id}", protect(h.GetExam))
	mux.HandleFunc("POST /api/exams", protect(h.CreateExam))
	mux.HandleFunc("DELETE /api/exams/{id}", protect(h.DeleteExam))

	// Questions
	mux.HandleFunc("GET /api/questions", protect(h.GetQuestions))
	mux.HandleFunc("POST /api/questions", protect(h.CreateQuestion))
	mux.HandleFunc("POST /api/questions/batch", protect(h.BatchQuestions))
	mux.HandleFunc("DELETE /api/questions/{id}", protect(h.DeleteQuestion))

	// Results
	mux.HandleFunc("GET /api/results", protect(h.GetMyResults))
	mux.HandleFunc("POST /api/results", protect(h.SaveResult))

	// Admin Users
	mux.HandleFunc("GET /api/users", protect(h.GetUsers))
	mux.HandleFunc("DELETE /api/users/{id}", protect(h.DeleteUser))
	mux.HandleFunc("POST /api/users/update", protect(h.UpdateUser))

	// Subjects/Topics
	mux.HandleFunc("GET /api/subjects", h.GetSubjects)
	mux.HandleFunc("POST /api/subjects", protect(h.CreateSubject))
	mux.HandleFunc("DELETE /api/subjects/{id}", protect(h.DeleteSubject))
	mux.HandleFunc("GET /api/topics", h.GetTopics)
	mux.HandleFunc("POST /api/topics", protect(h.CreateTopic))
	mux.HandleFunc("DELETE /api/topics/{id}", protect(h.DeleteTopic))

	// Company
	mux.HandleFunc("GET /api/company/links", protect(h.GetCompanyLinks))
	mux.HandleFunc("POST /api/company/links", protect(h.CreateLink))
	mux.HandleFunc("POST /api/company/invite", protect(h.CompanyInvite))
	mux.HandleFunc("GET /api/company/results", protect(h.GetCompanyResults))
	
	// Contact
	mux.HandleFunc("POST /api/contact/admin", h.ContactAdmin)

	// Public
	mux.HandleFunc("GET /api/public/exam/{token}", h.PublicGetExam)
	mux.HandleFunc("POST /api/public/exam/{token}/submit", h.PublicSubmit)

	// Aplicar middlewares de segurança
	// 1. HTTPS enforcement (em produção)
	server := http.HTTPSMiddleware(mux)
	
	// 2. CORS restritivo
	allowedOrigins := strings.Split(os.Getenv("CORS_ALLOWED_ORIGINS"), ",")
	if len(allowedOrigins) == 0 || allowedOrigins[0] == "" {
		// Fallback para desenvolvimento
		allowedOrigins = []string{"*"}
	}
	server = http.CORSMiddleware(allowedOrigins)(server)
	
	logger.Info("Server running on port %s", cfg.Port)
	logger.Fatal(httpNet.ListenAndServe(":"+cfg.Port, server))
}

func runMigration(db *sql.DB) {
	// Lê o arquivo schema.sql e executa
	// Nota: Em um ambiente real, o arquivo estaria em 'migrations/' ou embutido
	// Para este prompt, vamos assumir que o schema está no arquivo schema.sql gerado anteriormente
	// Aqui apenas logamos que deve ser rodado manualmente ou via ferramenta externa
	logger.Debug("Verificando schema do banco de dados...")

	// Execução simplificada inline para garantir funcionamento no MVP se o arquivo existir
	content, err := os.ReadFile("internal/database/schema.sql")
	if err == nil {
		_, err = db.Exec(string(content))
		if err != nil {
			logger.Warn("Migration warning: %v", err)
		} else {
			logger.Debug("Schema aplicado com sucesso")
		}
	}
}
