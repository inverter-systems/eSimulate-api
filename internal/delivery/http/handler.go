package http

import (
	"encoding/json"
	"esimulate-backend/internal/domain"
	"esimulate-backend/internal/service"
	"net/http"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
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
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		h.Error(w, 400, "Invalid JSON")
		return
	}
	user, err := h.Service.LoginUser(creds.Email, creds.Password)
	if err != nil {
		h.Error(w, 401, err.Error())
		return
	}
	h.JSON(w, 200, user)
}

func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req struct { Email string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Error(w, 400, "Invalid JSON")
		return
	}
	
	// Buscar usuário por email
	user, err := h.Service.Repo.GetUserByEmail(req.Email)
	if err != nil {
		// Por segurança, sempre retornar sucesso mesmo se usuário não existir
		h.JSON(w, 200, map[string]string{"message": "Email enviado"})
		return
	}
	
	// Gerar token de reset
	token := uuid.New().String()
	expiresAt := time.Now().Add(1 * time.Hour) // Token válido por 1 hora
	if err := h.Service.Repo.CreateToken(user.ID, token, "password_reset", expiresAt); err == nil {
		// Enviar email (não bloquear se falhar)
		go h.Service.EmailService.SendPasswordResetEmail(user.Email, user.Name, token)
	}
	
	h.JSON(w, 200, map[string]string{"message": "Email enviado"})
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req struct { Token, Password string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Error(w, 400, "Invalid JSON")
		return
	}
	
	// Validar token
	userID, tokenType, expiresAt, used, err := h.Service.Repo.GetToken(req.Token)
	if err != nil {
		h.Error(w, 400, "Token inválido")
		return
	}
	
	if tokenType != "password_reset" {
		h.Error(w, 400, "Token inválido")
		return
	}
	
	if used {
		h.Error(w, 400, "Token já foi utilizado")
		return
	}
	
	if time.Now().After(expiresAt) {
		// Excluir token expirado automaticamente
		h.Service.Repo.DeleteExpiredTokens()
		h.Error(w, 400, "Token expirado")
		return
	}
	
	// Atualizar senha
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		h.Error(w, 500, "Erro ao processar senha")
		return
	}
	
	updates := map[string]interface{}{
		"password": string(hashed),
	}
	if err := h.Service.Repo.UpdateUser(userID, updates); err != nil {
		h.Error(w, 500, "Erro ao atualizar senha")
		return
	}
	
	// Marcar token como usado
	h.Service.Repo.MarkTokenAsUsed(req.Token)
	
	h.JSON(w, 200, map[string]string{"message": "Senha alterada"})
}

func (h *Handler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req struct { Token string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Error(w, 400, "Invalid JSON")
		return
	}
	
	// Validar token
	userID, tokenType, expiresAt, used, err := h.Service.Repo.GetToken(req.Token)
	if err != nil {
		h.Error(w, 400, "Token inválido")
		return
	}
	
	if tokenType != "verification" {
		h.Error(w, 400, "Token inválido")
		return
	}
	
	if used {
		h.Error(w, 400, "Token já foi utilizado")
		return
	}
	
	if time.Now().After(expiresAt) {
		// Excluir token expirado automaticamente
		h.Service.Repo.DeleteExpiredTokens()
		h.Error(w, 400, "Token expirado")
		return
	}
	
	// Atualizar is_verified
	updates := map[string]interface{}{
		"is_verified": true,
	}
	if err := h.Service.Repo.UpdateUser(userID, updates); err != nil {
		h.Error(w, 500, "Erro ao verificar email")
		return
	}
	
	// Marcar token como usado
	h.Service.Repo.MarkTokenAsUsed(req.Token)
	
	h.JSON(w, 200, map[string]string{"message": "Email verificado"})
}

func (h *Handler) GetExams(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(string)
	
	// Query params: ?public=true ou ?owner=me
	publicOnly := r.URL.Query().Get("public") == "true"
	ownerOnly := r.URL.Query().Get("owner") == "me"
	
	exams, err := h.Service.Repo.GetExamsByUser(userID, publicOnly, ownerOnly)
	if err != nil { h.Error(w, 500, err.Error()); return }
	h.JSON(w, 200, exams)
}

func (h *Handler) GetExam(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := r.Context().Value("userID").(string)
	exam, err := h.Service.Repo.GetExamByID(id)
	if err != nil { 
		h.Error(w, 404, "Exam not found")
		return 
	}
	
	// Verificar acesso: se não é público, só o criador pode ver
	if !exam.IsPublic && exam.CreatedBy != userID {
		h.Error(w, 403, "Access denied")
		return
	}
	
	h.JSON(w, 200, exam)
}

func (h *Handler) CreateExam(w http.ResponseWriter, r *http.Request) {
	var e domain.Exam
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil { h.Error(w, 400, "Bad JSON"); return }
	
	userID := r.Context().Value("userID").(string)
	userRole := r.Context().Value("role").(string)
	e.CreatedBy = userID
	
	// Verificar se é update (ID existe) ou create (ID vazio)
	isUpdate := e.ID != ""
	if !isUpdate {
		e.ID = uuid.New().String()
		e.CreatedAt = time.Now().UnixMilli()
	} else {
		// Se for update, buscar exame existente para validar regras
		existingExam, err := h.Service.Repo.GetExamByID(e.ID)
		if err == nil {
			// Regra: Se isPublic estava true e está sendo alterado para false, só admin/specialist pode
			if existingExam.IsPublic && !e.IsPublic && userRole != "admin" && userRole != "specialist" {
				h.Error(w, 403, "Apenas admin ou specialist podem tornar provas públicas em privadas")
				return
			}
		}
	}
	
	// isVerified removido de Exam - será calculado no frontend baseado nas questões
	// Validar isVerified das questões: só admin/specialist podem marcar questões como verificadas
	for i := range e.Questions {
		if e.Questions[i].IsVerified && userRole != "admin" && userRole != "specialist" {
			e.Questions[i].IsVerified = false
		}
	}
	
	if err := h.Service.Repo.CreateExam(e); err != nil { h.Error(w, 500, err.Error()); return }
	
	// Retornar exame atualizado
	exam, err := h.Service.Repo.GetExamByID(e.ID)
	if err != nil { h.Error(w, 500, "Failed to fetch exam"); return }
	
	// Retornar 200 se for update, 201 se for create
	if isUpdate {
		h.JSON(w, 200, exam)
	} else {
		h.JSON(w, 201, exam)
	}
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
	if err := json.NewDecoder(r.Body).Decode(&qs); err != nil {
		h.Error(w, 400, "Invalid JSON")
		return
	}
	
	count := 0
	for _, q := range qs {
		if err := h.Service.Repo.CreateQuestion(q); err == nil {
			count++
		}
	}
	
	h.JSON(w, 200, map[string]interface{}{
		"status": "success",
		"count":  count,
	})
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
		h.JSON(w, 200, map[string]string{
		"status":  "success",
		"message": "Prova recebida.",
	})
}

// --- Company Invite ---
func (h *Handler) CompanyInvite(w http.ResponseWriter, r *http.Request) {
	// Verificar se é role company
	userRole, ok := r.Context().Value("role").(string)
	if !ok || userRole != "company" {
		h.Error(w, 403, "Apenas empresas podem enviar convites")
		return
	}
	
	var req struct {
		Email     string `json:"email"`
		LinkToken string `json:"linkToken"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Error(w, 400, "Invalid JSON")
		return
	}
	
	// Validar link
	link, err := h.Service.Repo.GetLinkByToken(req.LinkToken)
	if err != nil || !link.Active {
		h.Error(w, 404, "Link inválido ou inativo")
		return
	}
	
	// Buscar perfil da empresa
	companyID := r.Context().Value("userID").(string)
	company, err := h.Service.Repo.GetUserByID(companyID)
	if err != nil {
		h.Error(w, 500, "Erro ao buscar dados da empresa")
		return
	}
	
	// Extrair commercialName e companyLogo do profile
	commercialName := "eSimulate Recruiter"
	companyLogo := ""
	if company.Profile != nil {
		if profileMap, ok := company.Profile.(map[string]interface{}); ok {
			if name, exists := profileMap["commercialName"]; exists {
				if nameStr, ok := name.(string); ok && nameStr != "" {
					commercialName = nameStr
				}
			}
			if logo, exists := profileMap["companyLogo"]; exists {
				if logoStr, ok := logo.(string); ok {
					companyLogo = logoStr
				}
			}
		}
	}
	
	// Enviar email de convite
	go h.Service.EmailService.SendCompanyInviteEmail(
		req.Email,
		"", // candidateName - pode ser extraído se necessário
		commercialName,
		companyLogo,
		req.LinkToken,
	)
	
	h.JSON(w, 200, map[string]string{"message": "Email enviado"})
}

// --- Contact Admin ---
func (h *Handler) ContactAdmin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Subject     string `json:"subject"`
		Message     string `json:"message"`
		SenderEmail string `json:"senderEmail"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Error(w, 400, "Invalid JSON")
		return
	}
	
	// Enviar email para admin
	go h.Service.EmailService.SendContactAdminEmail(req.SenderEmail, req.Subject, req.Message)
	
	h.JSON(w, 200, map[string]string{"message": "Email enviado"})
}
