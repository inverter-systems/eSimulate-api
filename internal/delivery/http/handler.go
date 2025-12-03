package http

import (
	"encoding/json"
	"esimulate-backend/internal/domain"
	"esimulate-backend/internal/service"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type Handler struct {
	Service *service.Service
}

func NewHandler(svc *service.Service) *Handler {
	return &Handler{Service: svc}
}

// Helper methods
func (h *Handler) JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) Error(w http.ResponseWriter, status int, msg string) {
	h.JSON(w, status, map[string]string{"error": msg})
}

// --- Handlers Implementation ---

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var u domain.User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		h.Error(w, 400, "Invalid JSON")
		return
	}
	created, err := h.Service.RegisterUser(u)
	if err != nil {
		h.Error(w, 500, err.Error())
		return
	}
	h.JSON(w, 201, created)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var creds struct { Email, Password string }
	json.NewDecoder(r.Body).Decode(&creds)
	user, err := h.Service.LoginUser(creds.Email, creds.Password)
	if err != nil {
		h.Error(w, 401, err.Error())
		return
	}
	h.JSON(w, 200, user)
}

func (h *Handler) GetExams(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(string)
	exams, err := h.Service.Repo.GetExamsByUser(userID)
	if err != nil { h.Error(w, 500, err.Error()); return }
	h.JSON(w, 200, exams)
}

func (h *Handler) GetExam(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	exam, err := h.Service.Repo.GetExamByID(id)
	if err != nil { h.Error(w, 404, "Exam not found"); return }
	h.JSON(w, 200, exam)
}

func (h *Handler) CreateExam(w http.ResponseWriter, r *http.Request) {
	var e domain.Exam
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil { h.Error(w, 400, "Bad JSON"); return }
	
	e.CreatedBy = r.Context().Value("userID").(string)
	if err := h.Service.Repo.CreateExam(e); err != nil { h.Error(w, 500, err.Error()); return }
	
	// Retornar exame atualizado (pode ter sido atualizado se id já existia)
	exam, err := h.Service.Repo.GetExamByID(e.ID)
	if err != nil { h.Error(w, 500, "Failed to fetch exam"); return }
	h.JSON(w, 201, exam)
}

func (h *Handler) DeleteExam(w http.ResponseWriter, r *http.Request) {
	if err := h.Service.Repo.DeleteExam(r.PathValue("id")); err != nil {
		h.Error(w, 500, err.Error()); return
	}
	w.WriteHeader(204)
}

// --- Questions ---
func (h *Handler) GetQuestions(w http.ResponseWriter, r *http.Request) {
	qs, err := h.Service.Repo.GetQuestions()
	if err != nil { h.Error(w, 500, err.Error()); return }
	h.JSON(w, 200, qs)
}
func (h *Handler) CreateQuestion(w http.ResponseWriter, r *http.Request) {
	var q domain.Question
	json.NewDecoder(r.Body).Decode(&q)
	if err := h.Service.Repo.CreateQuestion(q); err != nil { h.Error(w, 500, err.Error()); return }
	h.JSON(w, 201, q)
}
func (h *Handler) BatchQuestions(w http.ResponseWriter, r *http.Request) {
	var qs []domain.Question
	json.NewDecoder(r.Body).Decode(&qs)
	for _, q := range qs { h.Service.Repo.CreateQuestion(q) }
	w.WriteHeader(200)
}
func (h *Handler) DeleteQuestion(w http.ResponseWriter, r *http.Request) {
	h.Service.Repo.DeleteQuestion(r.PathValue("id"))
	w.WriteHeader(204)
}

// --- Results ---
func (h *Handler) SaveResult(w http.ResponseWriter, r *http.Request) {
	var res domain.ExamResult
	json.NewDecoder(r.Body).Decode(&res)
	res.UserID = r.Context().Value("userID").(string)
	if err := h.Service.Repo.CreateResult(res); err != nil { h.Error(w, 500, err.Error()); return }
	h.JSON(w, 201, res)
}
func (h *Handler) GetMyResults(w http.ResponseWriter, r *http.Request) {
	res, err := h.Service.Repo.GetResultsByUser(r.Context().Value("userID").(string))
	if err != nil { h.Error(w, 500, err.Error()); return }
	h.JSON(w, 200, res)
}

// --- Admin Users ---
func (h *Handler) GetUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.Service.Repo.GetAllUsers()
	if err != nil { h.Error(w, 500, err.Error()); return }
	h.JSON(w, 200, users)
}
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	h.Service.Repo.DeleteUser(r.PathValue("id"))
	w.WriteHeader(204)
}
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	var req struct { 
		ID string
		Name string `json:"name,omitempty"`
		Profile any `json:"profile,omitempty"`
		Preferences any `json:"preferences,omitempty"` // llmProvider, llmApiKey (deve ser criptografado)
		OnboardingCompleted *bool `json:"onboardingCompleted,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Error(w, 400, "Invalid JSON")
		return
	}
	
	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Profile != nil {
		updates["profile"] = req.Profile
	}
	if req.Preferences != nil {
		// TODO: Criptografar llmApiKey antes de armazenar (AES-256)
		// Por enquanto, armazena em profile ou campo separado
		updates["preferences"] = req.Preferences
	}
	if req.OnboardingCompleted != nil {
		updates["onboardingCompleted"] = *req.OnboardingCompleted
	}
	
	if err := h.Service.Repo.UpdateUser(req.ID, updates); err != nil {
		h.Error(w, 500, err.Error())
		return
	}
	
	// Retornar usuário atualizado
	user, err := h.Service.Repo.GetUserByID(req.ID)
	if err != nil {
		h.Error(w, 500, "Failed to fetch updated user")
		return
	}
	h.JSON(w, 200, user)
}

// --- Subjects/Topics ---
func (h *Handler) GetSubjects(w http.ResponseWriter, r *http.Request) {
	s, _ := h.Service.Repo.GetSubjects()
	h.JSON(w, 200, s)
}
func (h *Handler) CreateSubject(w http.ResponseWriter, r *http.Request) {
	var s struct{ Name string }
	json.NewDecoder(r.Body).Decode(&s)
	sub, _ := h.Service.Repo.CreateSubject(s.Name)
	h.JSON(w, 201, sub)
}
func (h *Handler) DeleteSubject(w http.ResponseWriter, r *http.Request) {
	h.Service.Repo.DeleteSubject(r.PathValue("id"))
	w.WriteHeader(204)
}
func (h *Handler) GetTopics(w http.ResponseWriter, r *http.Request) {
	t, _ := h.Service.Repo.GetTopics()
	h.JSON(w, 200, t)
}
func (h *Handler) CreateTopic(w http.ResponseWriter, r *http.Request) {
	var t struct{ Name, SubjectID string }
	json.NewDecoder(r.Body).Decode(&t)
	top, _ := h.Service.Repo.CreateTopic(t.Name, t.SubjectID)
	h.JSON(w, 201, top)
}
func (h *Handler) DeleteTopic(w http.ResponseWriter, r *http.Request) {
	h.Service.Repo.DeleteTopic(r.PathValue("id"))
	w.WriteHeader(204)
}

// --- Company B2B ---
func (h *Handler) CreateLink(w http.ResponseWriter, r *http.Request) {
	var req struct { ExamID, Label string }
	json.NewDecoder(r.Body).Decode(&req)
	link := domain.PublicLink{
		ID: uuid.New().String(), ExamID: req.ExamID, CompanyID: r.Context().Value("userID").(string),
		Token: uuid.New().String()[:8], Label: req.Label, Active: true, CreatedAt: time.Now().UnixMilli(),
	}
	h.Service.Repo.CreateLink(link)
	h.JSON(w, 201, link)
}
func (h *Handler) GetCompanyLinks(w http.ResponseWriter, r *http.Request) {
	l, _ := h.Service.Repo.GetLinks(r.Context().Value("userID").(string))
	h.JSON(w, 200, l)
}
func (h *Handler) GetCompanyResults(w http.ResponseWriter, r *http.Request) {
	res, _ := h.Service.Repo.GetCompanyResults(r.Context().Value("userID").(string))
	h.JSON(w, 200, res)
}

// --- Public Access ---
func (h *Handler) PublicGetExam(w http.ResponseWriter, r *http.Request) {
	exam, link, err := h.Service.GetSanitizedExam(r.PathValue("token"))
	if err != nil { h.Error(w, 404, err.Error()); return }
	h.JSON(w, 200, map[string]interface{}{"exam": exam, "link": link})
}
func (h *Handler) PublicSubmit(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	link, err := h.Service.Repo.GetLinkByToken(token)
	if err != nil { h.Error(w, 404, "Invalid link"); return }
	
	// Validar link ativo e não expirado
	if !link.Active {
		h.Error(w, 400, "Link inativo")
		return
	}
	if link.ExpiresAt > 0 && time.Now().UnixMilli() > link.ExpiresAt {
		h.Error(w, 400, "Link expirado")
		return
	}

	// Obter exame original com gabarito
	exam, err := h.Service.Repo.GetExamByID(link.ExamID)
	if err != nil { h.Error(w, 404, "Exam not found"); return }

	var sub domain.ExamResult
	if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
		h.Error(w, 400, "Invalid JSON")
		return
	}
	
	// Calcular nota no backend (segurança: evitar fraude)
	// O frontend envia apenas as respostas selecionadas, não o score
	var answers []map[string]interface{}
	if sub.Answers != nil {
		// Converter Answers para formato esperado
		if answersMap, ok := sub.Answers.(map[string]interface{}); ok {
			// Se for um mapa, converter para array
			for _, v := range answersMap {
				if answerMap, ok := v.(map[string]interface{}); ok {
					answers = append(answers, answerMap)
				}
			}
		} else if answersArray, ok := sub.Answers.([]interface{}); ok {
			// Se já for array
			for _, v := range answersArray {
				if answerMap, ok := v.(map[string]interface{}); ok {
					answers = append(answers, answerMap)
				}
			}
		}
	}
	
	correctCount, totalQuestions := h.Service.CalculateScore(exam, answers)
	sub.Score = correctCount
	sub.TotalQuestions = totalQuestions
	sub.ID = uuid.New().String()
	sub.ExamID = link.ExamID
	sub.Date = time.Now().UnixMilli()
	
	if err := h.Service.Repo.CreateResult(sub); err != nil { 
		h.Error(w, 500, err.Error())
		return 
	}
	h.JSON(w, 200, map[string]string{"status": "success"})
}
