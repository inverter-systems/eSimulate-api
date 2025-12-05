package service

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"esimulate-backend/internal/config"
	"esimulate-backend/internal/domain"
	"esimulate-backend/internal/repository/postgres"
	"esimulate-backend/internal/security"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	Repo         *postgres.PostgresRepo
	Config       *config.Config
	EmailService *EmailService
}

func NewService(repo *postgres.PostgresRepo, cfg *config.Config) *Service {
	return &Service{
		Repo:         repo,
		Config:       cfg,
		EmailService: NewEmailService(),
	}
}

// --- Auth Services ---

func (s *Service) RegisterUser(u domain.User) (domain.User, error) {
	// Validar força da senha
	if err := security.ValidatePasswordStrength(u.Password); err != nil {
		return domain.User{}, err
	}
	
	// Hash password
	hashed, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return domain.User{}, err
	}
	u.Password = string(hashed)
	created, err := s.Repo.CreateUser(u)
	if err != nil {
		return created, err
	}
	
	// Só enviar email de verificação se o usuário não estiver já verificado
	if !created.IsVerified {
		// Gerar token de verificação e enviar email
		token := uuid.New().String()
		expiresAt := time.Now().Add(24 * time.Hour) // Token válido por 24 horas
		if err := s.Repo.CreateToken(created.ID, token, "verification", expiresAt); err == nil {
			// Enviar email de verificação (não bloquear se falhar)
			go s.EmailService.SendVerificationEmail(created.Email, created.Name, token)
		}
	}
	
	return created, nil
}

// LoginResponse representa a resposta do login conforme contrato v2.4.0
type LoginResponse struct {
	User  domain.User `json:"user"`
	Token string      `json:"token"` // Access Token (15 minutos)
}

// LoginUser autentica o usuário e retorna access token + refresh token
func (s *Service) LoginUser(email, password string) (LoginResponse, string, error) {
	u, err := s.Repo.GetUserByEmail(email)
	if err != nil {
		return LoginResponse{}, "", errors.New("usuário não encontrado")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)); err != nil {
		return LoginResponse{}, "", errors.New("senha incorreta")
	}

	// Verificar se email está verificado - conforme contrato FRONTEND_CONTRACT_API.md
	// Se as credenciais estiverem corretas mas isVerified for false, retornar 403
	if !u.IsVerified {
		return LoginResponse{}, "", errors.New("Email não verificado")
	}

	// Gerar Access Token (15 minutos) - conforme contrato v2.4.0
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": u.ID,
		"role":    u.Role,
		"exp":     time.Now().Add(15 * time.Minute).Unix(), // 15 minutos
	})
	
	accessTokenString, err := accessToken.SignedString([]byte(s.Config.JWTSecret))
	if err != nil {
		return LoginResponse{}, "", err
	}

	// Verificar limite de refresh tokens ativos (máximo 5 por usuário)
	activeCount, err := s.Repo.GetActiveRefreshTokensCount(u.ID)
	if err == nil && activeCount >= 5 {
		// Revogar tokens antigos, mantendo apenas os 4 mais recentes
		s.Repo.RevokeOldRefreshTokens(u.ID, 4)
	}
	
	// Gerar Refresh Token criptograficamente seguro (7 dias) - conforme contrato v2.4.0
	refreshToken, err := generateSecureToken()
	if err != nil {
		return LoginResponse{}, "", fmt.Errorf("erro ao gerar refresh token: %w", err)
	}
	refreshExpiresAt := time.Now().Add(7 * 24 * time.Hour) // 7 dias
	
	// Armazenar refresh token no banco
	if err := s.Repo.CreateToken(u.ID, refreshToken, "refresh_token", refreshExpiresAt); err != nil {
		return LoginResponse{}, "", fmt.Errorf("erro ao criar refresh token: %w", err)
	}

	u.Password = "" // Sanitiza antes de devolver
	u.Token = ""    // Não incluir token no user (vai no campo separado)
	
	// Extrair preferences do profile para retornar como campo separado
	if u.Profile != nil {
		if profileMap, ok := u.Profile.(map[string]interface{}); ok {
			if preferences, exists := profileMap["preferences"]; exists {
				u.Preferences = preferences
				// Remover preferences do profile para evitar duplicação
				delete(profileMap, "preferences")
				u.Profile = profileMap
			}
		}
	}
	
	return LoginResponse{
		User:  u,
		Token: accessTokenString,
	}, refreshToken, nil
}

// RefreshAccessToken gera um novo access token a partir de um refresh token válido
// Implementa rotação de refresh tokens e detecção de reutilização
func (s *Service) RefreshAccessToken(refreshToken string) (string, string, error) {
	// Buscar refresh token no banco
	userID, expiresAt, err := s.Repo.GetRefreshToken(refreshToken)
	if err != nil {
		return "", "", errors.New("refresh token inválido")
	}

	// Verificar se não expirou
	if time.Now().After(expiresAt) {
		// Limpar token expirado
		s.Repo.InvalidateRefreshToken(refreshToken)
		return "", "", errors.New("refresh token expirado")
	}

	// Verificar se token já foi usado (detecção de reutilização)
	// Se já foi usado, pode ser um ataque - invalidar todos os tokens do usuário
	_, _, _, used, err := s.Repo.GetToken(refreshToken)
	if err == nil && used {
		// Token foi reutilizado após invalidação - possível comprometimento
		s.Repo.InvalidateAllUserRefreshTokens(userID)
		return "", "", errors.New("refresh token inválido")
	}

	// Marcar token antigo como usado (antes de gerar novo)
	s.Repo.MarkRefreshTokenAsUsed(refreshToken)

	// Buscar usuário para obter role
	user, err := s.Repo.GetUserByID(userID)
	if err != nil {
		return "", "", errors.New("usuário não encontrado")
	}

	// Gerar novo Access Token (15 minutos)
	newAccessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"role":    user.Role,
		"exp":     time.Now().Add(15 * time.Minute).Unix(),
	})

	accessTokenString, err := newAccessToken.SignedString([]byte(s.Config.JWTSecret))
	if err != nil {
		return "", "", err
	}

	// Rotação: Gerar novo refresh token
	newRefreshToken, err := generateSecureToken()
	if err != nil {
		return "", "", fmt.Errorf("erro ao gerar novo refresh token: %w", err)
	}
	newRefreshExpiresAt := time.Now().Add(7 * 24 * time.Hour)
	
	// Armazenar novo refresh token
	if err := s.Repo.CreateToken(userID, newRefreshToken, "refresh_token", newRefreshExpiresAt); err != nil {
		return "", "", fmt.Errorf("erro ao criar novo refresh token: %w", err)
	}

	// Invalidar token antigo
	s.Repo.InvalidateRefreshToken(refreshToken)

	return accessTokenString, newRefreshToken, nil
}

// generateSecureToken gera um token criptograficamente seguro
func generateSecureToken() (string, error) {
	b := make([]byte, 32) // 32 bytes = 256 bits
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// --- Exam Services ---

func (s *Service) GetSanitizedExam(token string) (domain.Exam, domain.PublicLink, error) {
	link, err := s.Repo.GetLinkByToken(token)
	if err != nil {
		return domain.Exam{}, domain.PublicLink{}, errors.New("link inválido")
	}
	
	// Validar link ativo
	if !link.Active {
		return domain.Exam{}, domain.PublicLink{}, errors.New("link inativo")
	}
	
	// Validar expiração
	if link.ExpiresAt > 0 {
		now := time.Now().UnixMilli()
		if now > link.ExpiresAt {
			return domain.Exam{}, domain.PublicLink{}, errors.New("link expirado")
		}
	}

	exam, err := s.Repo.GetExamByID(link.ExamID)
	if err != nil {
		return domain.Exam{}, domain.PublicLink{}, errors.New("prova não encontrada")
	}

	// Calcular isVerified baseado nas questões (antes de sanitizar, mas após buscar)
	// Nota: isVerified é calculado antes de sanitizar para manter a informação
	exam.IsVerified = calculateExamIsVerified(exam)

	// Sanitização Crítica: Remover gabarito
	for i := range exam.Questions {
		exam.Questions[i].CorrectIndex = -1
		exam.Questions[i].Explanation = ""
	}

	return exam, link, nil
}

// calculateExamIsVerified calcula isVerified baseado nas questões (todas devem estar verificadas)
func calculateExamIsVerified(exam domain.Exam) bool {
	if len(exam.Questions) == 0 {
		return false // Exame sem questões não pode ser verificado
	}
	// Todas as questões devem estar verificadas
	for _, q := range exam.Questions {
		if !q.IsVerified {
			return false
		}
	}
	return true
}

// CalculateScore calcula a nota comparando respostas com gabarito do exame
func (s *Service) CalculateScore(exam domain.Exam, answers []map[string]interface{}) (int, int) {
	correctCount := 0
	totalQuestions := len(exam.Questions)
	
	// Criar mapa de questões por ID para busca rápida
	questionMap := make(map[string]domain.Question)
	for _, q := range exam.Questions {
		questionMap[q.ID] = q
	}
	
	// Comparar cada resposta com o gabarito
	for _, answer := range answers {
		questionID, ok1 := answer["questionId"].(string)
		selectedIndex, ok2 := answer["selectedIndex"].(float64)
		
		if !ok1 || !ok2 {
			continue
		}
		
		question, exists := questionMap[questionID]
		if !exists {
			continue
		}
		
		// Verificar se resposta está correta
		if int(selectedIndex) == question.CorrectIndex {
			correctCount++
		}
	}
	
	return correctCount, totalQuestions
}

// InitializeAdmin cria um usuário admin padrão se não existir nenhum admin no sistema
func (s *Service) InitializeAdmin() error {
	// Verificar se já existe um admin verificando pelo email
	_, err := s.Repo.GetUserByEmail(s.Config.AdminEmail)
	if err == nil {
		// Admin já existe
		return nil
	}
	
	// Verificar se existe algum admin no sistema (caso o email seja diferente)
	users, err := s.Repo.GetAllUsers()
	if err == nil {
		for _, user := range users {
			if user.Role == domain.RoleAdmin {
				return nil // Admin já existe
			}
		}
	}
	
	// Criar admin padrão
	admin := domain.User{
		ID:                 "",
		Name:               "Administrador",
		Email:              s.Config.AdminEmail,
		Password:           s.Config.AdminPassword,
		Role:               domain.RoleAdmin,
		Provider:           "email",
		IsVerified:          true,
		OnboardingCompleted: true,
	}
	
	_, err = s.RegisterUser(admin)
	if err != nil {
		return err
	}
	
	return nil
}
