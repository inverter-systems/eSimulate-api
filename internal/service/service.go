package service

import (
	"errors"
	"esimulate-backend/internal/config"
	"esimulate-backend/internal/domain"
	"esimulate-backend/internal/repository/postgres"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	Repo   *postgres.PostgresRepo
	Config *config.Config
}

func NewService(repo *postgres.PostgresRepo, cfg *config.Config) *Service {
	return &Service{Repo: repo, Config: cfg}
}

// --- Auth Services ---

func (s *Service) RegisterUser(u domain.User) (domain.User, error) {
	// Hash password
	hashed, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return domain.User{}, err
	}
	u.Password = string(hashed)
	return s.Repo.CreateUser(u)
}

func (s *Service) LoginUser(email, password string) (domain.User, error) {
	u, err := s.Repo.GetUserByEmail(email)
	if err != nil {
		return domain.User{}, errors.New("usuário não encontrado")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)); err != nil {
		return domain.User{}, errors.New("senha incorreta")
	}

	// Generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": u.ID,
		"role":    u.Role,
		"exp":     time.Now().Add(72 * time.Hour).Unix(),
	})
	
	tokenString, err := token.SignedString([]byte(s.Config.JWTSecret))
	if err != nil {
		return domain.User{}, err
	}
	u.Token = tokenString
	u.Password = "" // Sanitiza antes de devolver
	
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
	
	return u, nil
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

	// Sanitização Crítica: Remover gabarito
	for i := range exam.Questions {
		exam.Questions[i].CorrectIndex = -1
		exam.Questions[i].Explanation = ""
	}

	return exam, link, nil
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
