package domain

// Roles
type Role string

const (
	RoleAdmin   Role = "admin"
	RoleUser    Role = "user"
	RoleCompany Role = "company"
)

// User representa um usuário do sistema
type User struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Email               string `json:"email"`
	Password            string `json:"password,omitempty"` // Omitido no JSON de saída
	Role                Role   `json:"role"`
	Provider            string `json:"provider"`
	CreatedAt           int64  `json:"createdAt"`
	IsVerified          bool   `json:"isVerified"`
	OnboardingCompleted bool   `json:"onboardingCompleted,omitempty"` // Flag para fluxo pós-verificação
	Profile             any    `json:"profile,omitempty"`             // JSONB flexível (UserProfile)
	Token               string `json:"token,omitempty"`               // Usado apenas na resposta de login
}

// Exam representa um simulado
type Exam struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Questions   []Question `json:"questions"`
	Subjects    []string   `json:"subjects"`
	CreatedBy   string     `json:"createdBy,omitempty"`
	CreatedAt   int64      `json:"createdAt"`
}

// Question representa uma questão
type Question struct {
	ID           string   `json:"id"`
	Text         string   `json:"text"`
	Options      []string `json:"options"`
	CorrectIndex int      `json:"correctIndex"`          // -1 se for resposta pública sanitizada
	Explanation  string   `json:"explanation,omitempty"` // Vazio se for resposta pública sanitizada
	SubjectID    string   `json:"subjectId,omitempty"`   // FK para subjects (UUID)
	TopicID      string   `json:"topicId,omitempty"`     // FK para topics (UUID)
	IsPublic     bool     `json:"isPublic,omitempty"`    // Indica se a questão é pública
	// Campos legados para compatibilidade (opcional, podem ser removidos depois)
	Subject string `json:"subject,omitempty"` // @deprecated - usar subjectId
	Topic   string `json:"topic,omitempty"`   // @deprecated - usar topicId
}

// ExamResult representa o resultado de uma prova
type ExamResult struct {
	ID               string `json:"id"`
	ExamID           string `json:"examId"`
	UserID           string `json:"userId,omitempty"`
	CandidateName    string `json:"candidateName,omitempty"`
	CandidateEmail   string `json:"candidateEmail,omitempty"`
	Score            int    `json:"score"`
	TotalQuestions   int    `json:"totalQuestions"`
	Answers          any    `json:"answers"` // JSONB
	TimeSpentSeconds int    `json:"timeSpentSeconds"`
	Date             int64  `json:"date"`
	ExamTitle        string `json:"examTitle,omitempty"`
}

// PublicLink é o link gerado por empresas
type PublicLink struct {
	ID        string `json:"id"`
	ExamID    string `json:"examId"`
	CompanyID string `json:"companyId"`
	Token     string `json:"token"`
	Label     string `json:"label"`
	Active    bool   `json:"active"`
	ExpiresAt int64  `json:"expiresAt,omitempty"` // Timestamp em milissegundos (0 se não expira)
	CreatedAt int64  `json:"createdAt"`
	ExamTitle string `json:"examTitle,omitempty"`
}

type Subject struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Topic struct {
	ID        string `json:"id"`
	SubjectID string `json:"subjectId"`
	Name      string `json:"name"`
}
