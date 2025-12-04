package main

import (
	"database/sql"
	"esimulate-backend/internal/config"
	"esimulate-backend/internal/delivery/http"
	"esimulate-backend/internal/repository/postgres"
	"esimulate-backend/internal/service"
	"fmt"
	"log"
	"os"
	httpNet "net/http"
)

func main() {
	// 1. Config
	cfg := config.LoadConfig()

	// 2. Database
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Cannot connect to DB:", err)
	}
	fmt.Println("Connected to PostgreSQL")

	// 3. Init Schema (Simple migration for MVP)
	// Em produção, usar 'golang-migrate'
	runMigration(db)

	// 4. Layers Init
	repo := postgres.NewPostgresRepo(db)
	svc := service.NewService(repo, cfg)
	
	// 5. Initialize Admin User
	if err := svc.InitializeAdmin(); err != nil {
		log.Printf("Warning: Failed to initialize admin user: %v", err)
	} else {
		fmt.Printf("✓ Admin user ready: %s\n", cfg.AdminEmail)
	}
	
	h := http.NewHandler(svc)

	// 6. Router (Go 1.22)
	mux := httpNet.NewServeMux()

	// Auth
	mux.HandleFunc("POST /api/auth/register", h.Register)
	mux.HandleFunc("POST /api/auth/login", h.Login)
	// Auth Recovery
	mux.HandleFunc("POST /api/auth/forgot-password", h.ForgotPassword)
	mux.HandleFunc("POST /api/auth/reset-password", h.ResetPassword)
	mux.HandleFunc("POST /api/auth/verify-email", h.VerifyEmail)

	// Protected Routes Helper
	protect := func(handler httpNet.HandlerFunc) httpNet.HandlerFunc {
		return http.AuthMiddleware(cfg.JWTSecret, handler)
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
	mux.HandleFunc("GET /api/company/results", protect(h.GetCompanyResults))

	// Public
	mux.HandleFunc("GET /api/public/exam/{token}", h.PublicGetExam)
	mux.HandleFunc("POST /api/public/exam/{token}/submit", h.PublicSubmit)

	server := http.CORSMiddleware(mux)
	fmt.Printf("Server running on port %s\n", cfg.Port)
	log.Fatal(httpNet.ListenAndServe(":"+cfg.Port, server))
}

func runMigration(db *sql.DB) {
	// Lê o arquivo schema.sql e executa
	// Nota: Em um ambiente real, o arquivo estaria em 'migrations/' ou embutido
	// Para este prompt, vamos assumir que o schema está no arquivo schema.sql gerado anteriormente
	// Aqui apenas logamos que deve ser rodado manualmente ou via ferramenta externa
	fmt.Println("INFO: Ensure database schema is applied using schema.sql")

	// Execução simplificada inline para garantir funcionamento no MVP se o arquivo existir
	content, err := os.ReadFile("internal/database/schema.sql")
	if err == nil {
		_, err = db.Exec(string(content))
		if err != nil {
			fmt.Println("Migration warning:", err)
		} else {
			fmt.Println("Schema applied.")
		}
	}
}
