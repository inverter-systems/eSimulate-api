package postgres

import (
	"database/sql"
	"encoding/json"
	"esimulate-backend/internal/domain"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// --- Base Repository ---
type PostgresRepo struct {
	DB *sql.DB
}

func NewPostgresRepo(db *sql.DB) *PostgresRepo {
	return &PostgresRepo{DB: db}
}

// --- User Implementation ---

func (r *PostgresRepo) CreateUser(u domain.User) (domain.User, error) {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	if u.CreatedAt == 0 {
		u.CreatedAt = time.Now().UnixMilli()
	}

	createdAtTime := time.UnixMilli(u.CreatedAt)

	query := `INSERT INTO users (id, name, email, password_hash, role, provider, created_at, is_verified, onboarding_completed) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`

	err := r.DB.QueryRow(query, u.ID, u.Name, u.Email, u.Password, u.Role, u.Provider, createdAtTime, u.IsVerified, u.OnboardingCompleted).Scan(&u.ID)
	u.Password = "" 
	return u, err
}

func (r *PostgresRepo) GetUserByEmail(email string) (domain.User, error) {
	var u domain.User
	var profileData []byte
	var createdAt time.Time
	query := `SELECT id, name, email, password_hash, role, provider, created_at, profile, is_verified, onboarding_completed FROM users WHERE email=$1`
	err := r.DB.QueryRow(query, email).Scan(&u.ID, &u.Name, &u.Email, &u.Password, &u.Role, &u.Provider, &createdAt, &profileData, &u.IsVerified, &u.OnboardingCompleted)
	if err != nil { return u, err }
	u.CreatedAt = createdAt.UnixMilli()
	if len(profileData) > 0 { json.Unmarshal(profileData, &u.Profile) }
	return u, nil
}

func (r *PostgresRepo) GetUserByID(id string) (domain.User, error) {
	var u domain.User
	var profileData []byte
	var createdAt time.Time
	query := `SELECT id, name, email, password_hash, role, provider, created_at, profile, is_verified, onboarding_completed FROM users WHERE id=$1`
	err := r.DB.QueryRow(query, id).Scan(&u.ID, &u.Name, &u.Email, &u.Password, &u.Role, &u.Provider, &createdAt, &profileData, &u.IsVerified, &u.OnboardingCompleted)
	if err != nil { return u, err }
	u.CreatedAt = createdAt.UnixMilli()
	if len(profileData) > 0 {
		json.Unmarshal(profileData, &u.Profile)
		// Extrair preferences do profile se existir
		if profileMap, ok := u.Profile.(map[string]interface{}); ok {
			if preferences, exists := profileMap["preferences"]; exists {
				u.Preferences = preferences
				// Remover preferences do profile para evitar duplicação
				delete(profileMap, "preferences")
				u.Profile = profileMap
			}
		}
	}
	u.Password = "" // Não retornar senha
	return u, nil
}

func (r *PostgresRepo) GetAllUsers() ([]domain.User, error) {
	rows, err := r.DB.Query("SELECT id, name, email, role, created_at, profile, is_verified, onboarding_completed FROM users")
	if err != nil { return nil, err }
	defer rows.Close()
	var users []domain.User
	for rows.Next() {
		var u domain.User
		var p []byte
		var createdAt time.Time
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &createdAt, &p, &u.IsVerified, &u.OnboardingCompleted); err == nil {
			u.CreatedAt = createdAt.UnixMilli()
			if len(p) > 0 {
				json.Unmarshal(p, &u.Profile)
				// Extrair preferences do profile se existir
				if profileMap, ok := u.Profile.(map[string]interface{}); ok {
					if preferences, exists := profileMap["preferences"]; exists {
						u.Preferences = preferences
						// Remover preferences do profile para evitar duplicação
						delete(profileMap, "preferences")
						u.Profile = profileMap
					}
				}
			}
			users = append(users, u)
		}
	}
	return users, nil
}

func (r *PostgresRepo) UpdateUserProfile(id string, profile any) error {
	pJSON, _ := json.Marshal(profile)
	_, err := r.DB.Exec("UPDATE users SET profile=$1 WHERE id=$2", pJSON, id)
	return err
}

func (r *PostgresRepo) UpdateUser(id string, updates map[string]interface{}) error {
	// Atualiza name se fornecido
	if name, ok := updates["name"]; ok {
		_, err := r.DB.Exec("UPDATE users SET name=$1 WHERE id=$2", name, id)
		if err != nil { return err }
	}
	// Atualiza profile se fornecido
	if profile, ok := updates["profile"]; ok {
		pJSON, _ := json.Marshal(profile)
		_, err := r.DB.Exec("UPDATE users SET profile=$1 WHERE id=$2", pJSON, id)
		if err != nil { return err }
	}
	// Atualiza preferences se fornecido (armazena em profile por enquanto)
	// TODO: Criar campo separado preferences ou criptografar llmApiKey
	if preferences, ok := updates["preferences"]; ok {
		// Por enquanto, mescla preferences no profile
		// Futuro: campo separado ou criptografia de llmApiKey
		var currentProfile map[string]interface{}
		var profileData []byte
		err := r.DB.QueryRow("SELECT profile FROM users WHERE id=$1", id).Scan(&profileData)
		if err == nil && len(profileData) > 0 {
			json.Unmarshal(profileData, &currentProfile)
		} else {
			currentProfile = make(map[string]interface{})
		}
		currentProfile["preferences"] = preferences
		pJSON, _ := json.Marshal(currentProfile)
		_, err = r.DB.Exec("UPDATE users SET profile=$1 WHERE id=$2", pJSON, id)
		if err != nil { return err }
	}
	// Atualiza onboarding_completed se fornecido
	if onboardingCompleted, ok := updates["onboardingCompleted"]; ok {
		_, err := r.DB.Exec("UPDATE users SET onboarding_completed=$1 WHERE id=$2", onboardingCompleted, id)
		if err != nil { return err }
	}
	// Atualiza password se fornecido
	if password, ok := updates["password"]; ok {
		_, err := r.DB.Exec("UPDATE users SET password_hash=$1 WHERE id=$2", password, id)
		if err != nil { return err }
	}
	// Atualiza is_verified se fornecido
	if isVerified, ok := updates["is_verified"]; ok {
		_, err := r.DB.Exec("UPDATE users SET is_verified=$1 WHERE id=$2", isVerified, id)
		if err != nil { return err }
	}
	return nil
}

func (r *PostgresRepo) DeleteUser(id string) error {
	// Cascade garante limpeza de Exams e Results
	_, err := r.DB.Exec("DELETE FROM users WHERE id=$1", id)
	return err
}

// --- Exam Implementation ---

func (r *PostgresRepo) CreateExam(e domain.Exam) error {
	// Iniciar transação
	tx, err := r.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	
	// Preparar dados
	sJSON, _ := json.Marshal(e.Subjects)
	if e.CreatedAt == 0 {
		e.CreatedAt = time.Now().UnixMilli()
	}
	createdAtTime := time.UnixMilli(e.CreatedAt)
	
	// 1. Criar/Atualizar exame (sem campo questions JSONB - usando apenas exam_questions)
	// is_verified removido: calculado no frontend baseado nas questões
	query := `INSERT INTO exams (id, title, description, subjects, time_limit, is_public, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET 
			title=$2, 
			description=$3, 
			subjects=$4,
			time_limit=$5,
			is_public=$6,
			updated_at=NOW()`
	_, err = tx.Exec(query, e.ID, e.Title, e.Description, sJSON, e.TimeLimit, e.IsPublic, e.CreatedBy, createdAtTime)
	if err != nil {
		return err
	}
	
	// 2. Remover relacionamentos antigos (se for update)
	_, err = tx.Exec("DELETE FROM exam_questions WHERE exam_id=$1", e.ID)
	if err != nil {
		return err
	}
	
	// 3. Para cada questão: fazer upsert na tabela questions e criar relacionamento
	for _, q := range e.Questions {
		// Gerar ID se não existir
		if q.ID == "" {
			q.ID = uuid.New().String()
		}
		
		// Upsert questão
		optJSON, _ := json.Marshal(q.Options)
		var subjectID, topicID sql.NullString
		if q.SubjectID != "" {
			subjectID.String = q.SubjectID
			subjectID.Valid = true
		}
		if q.TopicID != "" {
			topicID.String = q.TopicID
			topicID.Valid = true
		}
		
		upsertQuery := `INSERT INTO questions (id, text, options, correct_index, explanation, subject_id, topic_id, is_public, is_verified)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (id) DO UPDATE SET 
				text=$2, 
				options=$3, 
				correct_index=$4, 
				explanation=$5, 
				subject_id=$6, 
				topic_id=$7,
				is_public=$8,
				is_verified=$9,
				updated_at=NOW()`
		_, err = tx.Exec(upsertQuery, q.ID, q.Text, optJSON, q.CorrectIndex, q.Explanation, subjectID, topicID, q.IsPublic, q.IsVerified)
		if err != nil {
			return err
		}
		
		// Criar relacionamento exam_questions
		_, err = tx.Exec("INSERT INTO exam_questions (exam_id, question_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", e.ID, q.ID)
		if err != nil {
			return err
		}
	}
	
	// Commit transação
	return tx.Commit()
}

func (r *PostgresRepo) GetExams() ([]domain.Exam, error) {
	// Buscar exames (sem questions - usando apenas exam_questions)
	rows, err := r.DB.Query(`
		SELECT id, title, description, subjects, time_limit, is_public, created_by, created_at 
		FROM exams 
		ORDER BY created_at DESC`)
	if err != nil { return nil, err }
	defer rows.Close()
	
	var exams []domain.Exam
	examMap := make(map[string]*domain.Exam)
	
	for rows.Next() {
		var e domain.Exam
		var s []byte
		var timeLimit sql.NullInt64
		var createdAt time.Time
		var createdBy string
		rows.Scan(&e.ID, &e.Title, &e.Description, &s, &timeLimit, &e.IsPublic, &createdBy, &createdAt)
		e.CreatedAt = createdAt.UnixMilli()
		e.CreatedBy = createdBy
		if timeLimit.Valid {
			e.TimeLimit = int(timeLimit.Int64)
		}
		json.Unmarshal(s, &e.Subjects)
		e.Questions = []domain.Question{} // Inicializar array vazio
		exams = append(exams, e)
		examMap[e.ID] = &exams[len(exams)-1]
	}
	
	// Buscar questões para cada exame (JOIN) - similar ao GetExamsByUser
	if len(examMap) > 0 {
		examIDs := make([]string, 0, len(examMap))
		for id := range examMap {
			examIDs = append(examIDs, id)
		}
		
		placeholders := ""
		for i := range examIDs {
			if i > 0 {
				placeholders += ","
			}
			placeholders += fmt.Sprintf("$%d", i+1)
		}
		
		query := fmt.Sprintf(`
			SELECT eq.exam_id, q.id, q.text, q.options, q.correct_index, q.explanation, q.subject_id, q.topic_id, q.is_public, q.is_verified
			FROM exam_questions eq
			JOIN questions q ON eq.question_id = q.id
			WHERE eq.exam_id IN (%s)
			ORDER BY eq.exam_id`, placeholders)
		
		args := make([]interface{}, len(examIDs))
		for i, id := range examIDs {
			args[i] = id
		}
		
		qRows, err := r.DB.Query(query, args...)
		if err == nil {
			defer qRows.Close()
			for qRows.Next() {
				var examID string
				var q domain.Question
				var opt []byte
				var subjectID, topicID sql.NullString
				qRows.Scan(&examID, &q.ID, &q.Text, &opt, &q.CorrectIndex, &q.Explanation, &subjectID, &topicID, &q.IsPublic, &q.IsVerified)
				if subjectID.Valid {
					q.SubjectID = subjectID.String
				}
				if topicID.Valid {
					q.TopicID = topicID.String
				}
				json.Unmarshal(opt, &q.Options)
				if exam, exists := examMap[examID]; exists {
					exam.Questions = append(exam.Questions, q)
				}
			}
		}
	}
	
	return exams, nil
}

func (r *PostgresRepo) GetExamsByUser(userID string, publicOnly bool, ownerOnly bool) ([]domain.Exam, error) {
	// Construir query baseada nos filtros
	query := `SELECT e.id, e.title, e.description, e.subjects, e.time_limit, e.is_public, e.created_by, e.created_at FROM exams e WHERE 1=1`
	args := []interface{}{}
	argIndex := 1
	
	if ownerOnly {
		query += fmt.Sprintf(" AND e.created_by=$%d", argIndex)
		args = append(args, userID)
		argIndex++
	} else if publicOnly {
		query += " AND e.is_public=true"
	} else {
		// Padrão: exames do usuário OU exames públicos
		query += fmt.Sprintf(" AND (e.created_by=$%d OR e.is_public=true)", argIndex)
		args = append(args, userID)
		argIndex++
	}
	
	query += " ORDER BY e.created_at DESC"
	
	rows, err := r.DB.Query(query, args...)
	if err != nil { return nil, err }
	defer rows.Close()
	
	var exams []domain.Exam
	examMap := make(map[string]*domain.Exam)
	
	for rows.Next() {
		var e domain.Exam
		var s []byte
		var timeLimit sql.NullInt64
		var createdAt time.Time
		var createdBy string
		rows.Scan(&e.ID, &e.Title, &e.Description, &s, &timeLimit, &e.IsPublic, &createdBy, &createdAt)
		e.CreatedBy = createdBy
		e.CreatedAt = createdAt.UnixMilli()
		if timeLimit.Valid {
			e.TimeLimit = int(timeLimit.Int64)
		}
		json.Unmarshal(s, &e.Subjects)
		e.Questions = []domain.Question{} // Inicializar array vazio
		exams = append(exams, e)
		examMap[e.ID] = &exams[len(exams)-1]
	}
	
	// Buscar questões para cada exame (JOIN)
	if len(examMap) > 0 {
		examIDs := make([]string, 0, len(examMap))
		for id := range examMap {
			examIDs = append(examIDs, id)
		}
		
		// Query para buscar questões relacionadas
		placeholders := ""
		for i := range examIDs {
			if i > 0 {
				placeholders += ","
			}
			placeholders += fmt.Sprintf("$%d", i+1)
		}
		
		query := fmt.Sprintf(`
			SELECT eq.exam_id, q.id, q.text, q.options, q.correct_index, q.explanation, q.subject_id, q.topic_id, q.is_public, q.is_verified
			FROM exam_questions eq
			JOIN questions q ON eq.question_id = q.id
			WHERE eq.exam_id IN (%s)
			ORDER BY eq.exam_id`, placeholders)
		
		args := make([]interface{}, len(examIDs))
		for i, id := range examIDs {
			args[i] = id
		}
		
		qRows, err := r.DB.Query(query, args...)
		if err == nil {
			defer qRows.Close()
			for qRows.Next() {
				var examID string
				var q domain.Question
				var opt []byte
				var subjectID, topicID sql.NullString
				qRows.Scan(&examID, &q.ID, &q.Text, &opt, &q.CorrectIndex, &q.Explanation, &subjectID, &topicID, &q.IsPublic, &q.IsVerified)
				if subjectID.Valid {
					q.SubjectID = subjectID.String
				}
				if topicID.Valid {
					q.TopicID = topicID.String
				}
				json.Unmarshal(opt, &q.Options)
				if exam, exists := examMap[examID]; exists {
					exam.Questions = append(exam.Questions, q)
				}
			}
		}
	}
	
	return exams, nil
}

func (r *PostgresRepo) GetExamByID(id string) (domain.Exam, error) {
	var e domain.Exam
	var s []byte
	var timeLimit sql.NullInt64
	var createdAt time.Time
	
	// Buscar exame
	err := r.DB.QueryRow(`
		SELECT id, title, description, subjects, time_limit, is_public, created_by, created_at 
		FROM exams 
		WHERE id=$1`, id).
		Scan(&e.ID, &e.Title, &e.Description, &s, &timeLimit, &e.IsPublic, &e.CreatedBy, &createdAt)
	if err != nil {
		return e, err
	}
	
	e.CreatedAt = createdAt.UnixMilli()
	if timeLimit.Valid {
		e.TimeLimit = int(timeLimit.Int64)
	}
	json.Unmarshal(s, &e.Subjects)
	
	// Buscar questões relacionadas (JOIN)
	rows, err := r.DB.Query(`
		SELECT q.id, q.text, q.options, q.correct_index, q.explanation, q.subject_id, q.topic_id, q.is_public, q.is_verified
		FROM exam_questions eq
		JOIN questions q ON eq.question_id = q.id
		WHERE eq.exam_id = $1`, id)
	if err == nil {
		defer rows.Close()
		e.Questions = []domain.Question{}
		for rows.Next() {
			var q domain.Question
			var opt []byte
			var subjectID, topicID sql.NullString
			rows.Scan(&q.ID, &q.Text, &opt, &q.CorrectIndex, &q.Explanation, &subjectID, &topicID, &q.IsPublic, &q.IsVerified)
			if subjectID.Valid {
				q.SubjectID = subjectID.String
			}
			if topicID.Valid {
				q.TopicID = topicID.String
			}
			json.Unmarshal(opt, &q.Options)
			e.Questions = append(e.Questions, q)
		}
	}
	
	return e, nil
}

func (r *PostgresRepo) DeleteExam(id string) error {
	_, err := r.DB.Exec("DELETE FROM exams WHERE id=$1", id)
	return err
}

// --- Question Implementation ---

func (r *PostgresRepo) CreateQuestion(q domain.Question) error {
	optJSON, _ := json.Marshal(q.Options)
	
	// Preparar subject_id e topic_id (podem ser NULL)
	var subjectID, topicID sql.NullString
	if q.SubjectID != "" {
		subjectID.String = q.SubjectID
		subjectID.Valid = true
	}
	if q.TopicID != "" {
		topicID.String = q.TopicID
		topicID.Valid = true
	}
	
	query := `INSERT INTO questions (id, text, options, correct_index, explanation, subject_id, topic_id, is_public, is_verified)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET 
			text=$2, 
			options=$3, 
			correct_index=$4, 
			explanation=$5, 
			subject_id=$6, 
			topic_id=$7,
			is_public=$8,
			is_verified=$9`
	_, err := r.DB.Exec(query, q.ID, q.Text, optJSON, q.CorrectIndex, q.Explanation, subjectID, topicID, q.IsPublic, q.IsVerified)
	return err
}

func (r *PostgresRepo) GetQuestions() ([]domain.Question, error) {
	rows, err := r.DB.Query("SELECT id, text, options, correct_index, explanation, subject_id, topic_id, is_public, is_verified FROM questions")
	if err != nil { return nil, err }
	defer rows.Close()
	var questions []domain.Question
	for rows.Next() {
		var q domain.Question
		var opt []byte
		var subjectID, topicID sql.NullString
		err := rows.Scan(&q.ID, &q.Text, &opt, &q.CorrectIndex, &q.Explanation, &subjectID, &topicID, &q.IsPublic, &q.IsVerified)
		if err != nil { continue }
		if subjectID.Valid {
			q.SubjectID = subjectID.String
		}
		if topicID.Valid {
			q.TopicID = topicID.String
		}
		json.Unmarshal(opt, &q.Options)
		questions = append(questions, q)
	}
	return questions, nil
}

func (r *PostgresRepo) DeleteQuestion(id string) error {
	_, err := r.DB.Exec("DELETE FROM questions WHERE id=$1", id)
	return err
}

// --- Result Implementation ---

func (r *PostgresRepo) CreateResult(res domain.ExamResult) error {
	ansJSON, _ := json.Marshal(res.Answers)
	var userID sql.NullString
	if res.UserID != "" { userID.String = res.UserID; userID.Valid = true }

	query := `INSERT INTO results (id, exam_id, user_id, candidate_name, candidate_email, score, total_questions, answers, time_spent_seconds, date)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := r.DB.Exec(query, res.ID, res.ExamID, userID, res.CandidateName, res.CandidateEmail, res.Score, res.TotalQuestions, ansJSON, res.TimeSpentSeconds, res.Date)
	return err
}

func (r *PostgresRepo) GetResultsByUser(userID string) ([]domain.ExamResult, error) {
	query := `SELECT r.id, r.exam_id, r.score, r.total_questions, r.time_spent_seconds, r.date, e.title 
		FROM results r JOIN exams e ON r.exam_id = e.id WHERE r.user_id=$1 ORDER BY r.date DESC`
	rows, err := r.DB.Query(query, userID)
	if err != nil { return nil, err }
	defer rows.Close()
	var results []domain.ExamResult
	for rows.Next() {
		var res domain.ExamResult
		rows.Scan(&res.ID, &res.ExamID, &res.Score, &res.TotalQuestions, &res.TimeSpentSeconds, &res.Date, &res.ExamTitle)
		results = append(results, res)
	}
	return results, nil
}

func (r *PostgresRepo) GetCompanyResults(companyID string) ([]domain.ExamResult, error) {
	query := `SELECT r.id, r.candidate_name, r.candidate_email, r.score, r.total_questions, r.date, e.title
		FROM results r
		JOIN public_links pl ON pl.exam_id = r.exam_id
		JOIN exams e ON r.exam_id = e.id
		WHERE pl.company_id = $1 AND r.candidate_name IS NOT NULL
		ORDER BY r.date DESC`
	rows, err := r.DB.Query(query, companyID)
	if err != nil { return nil, err }
	defer rows.Close()
	var results []domain.ExamResult
	for rows.Next() {
		var res domain.ExamResult
		rows.Scan(&res.ID, &res.CandidateName, &res.CandidateEmail, &res.Score, &res.TotalQuestions, &res.Date, &res.ExamTitle)
		results = append(results, res)
	}
	return results, nil
}

// --- Meta & Links ---
func (r *PostgresRepo) CreateLink(l domain.PublicLink) error {
	var expiresAt interface{}
	if l.ExpiresAt > 0 {
		// Converter milissegundos para timestamp
		expiresAt = time.Unix(l.ExpiresAt/1000, 0)
	}
	_, err := r.DB.Exec(`INSERT INTO public_links (id, exam_id, company_id, token, label, active, expires_at, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		l.ID, l.ExamID, l.CompanyID, l.Token, l.Label, l.Active, expiresAt, time.Unix(l.CreatedAt/1000, 0))
	return err
}

func (r *PostgresRepo) GetLinks(companyID string) ([]domain.PublicLink, error) {
	rows, err := r.DB.Query(`SELECT pl.id, pl.exam_id, pl.token, pl.label, pl.active, pl.expires_at, pl.created_at, e.title 
		FROM public_links pl JOIN exams e ON pl.exam_id = e.id WHERE pl.company_id=$1`, companyID)
	if err != nil { return nil, err }
	defer rows.Close()
	var links []domain.PublicLink
	for rows.Next() {
		var l domain.PublicLink
		var expiresAt sql.NullTime
		var createdAt time.Time
		err := rows.Scan(&l.ID, &l.ExamID, &l.Token, &l.Label, &l.Active, &expiresAt, &createdAt, &l.ExamTitle)
		if err != nil { continue }
		if expiresAt.Valid {
			l.ExpiresAt = expiresAt.Time.UnixMilli()
		}
		l.CreatedAt = createdAt.UnixMilli()
		links = append(links, l)
	}
	return links, nil
}

// --- Token Management ---

func (r *PostgresRepo) CreateToken(userID, token, tokenType string, expiresAt time.Time) error {
	query := `INSERT INTO tokens (user_id, token, type, expires_at) VALUES ($1, $2, $3, $4)`
	_, err := r.DB.Exec(query, userID, token, tokenType, expiresAt)
	return err
}

func (r *PostgresRepo) GetToken(token string) (string, string, time.Time, bool, error) {
	var userID, tokenType string
	var expiresAt time.Time
	var used bool
	query := `SELECT user_id, type, expires_at, used FROM tokens WHERE token=$1`
	err := r.DB.QueryRow(query, token).Scan(&userID, &tokenType, &expiresAt, &used)
	if err != nil {
		return "", "", time.Time{}, false, err
	}
	return userID, tokenType, expiresAt, used, nil
}

func (r *PostgresRepo) MarkTokenAsUsed(token string) error {
	// Excluir imediatamente após uso (não precisamos mais dele)
	_, err := r.DB.Exec("DELETE FROM tokens WHERE token=$1", token)
	return err
}

// DeleteExpiredTokens exclui todos os tokens expirados (chamado quando detectamos um token expirado)
func (r *PostgresRepo) DeleteExpiredTokens() error {
	_, err := r.DB.Exec("DELETE FROM tokens WHERE expires_at < NOW()")
	return err
}

func (r *PostgresRepo) GetLinkByToken(token string) (domain.PublicLink, error) {
	var l domain.PublicLink
	var expiresAt sql.NullTime
	var createdAt time.Time
	err := r.DB.QueryRow("SELECT id, exam_id, label, active, expires_at, created_at FROM public_links WHERE token=$1", token).
		Scan(&l.ID, &l.ExamID, &l.Label, &l.Active, &expiresAt, &createdAt)
	if err != nil { return l, err }
	if expiresAt.Valid {
		l.ExpiresAt = expiresAt.Time.UnixMilli()
	}
	l.CreatedAt = createdAt.UnixMilli()
	return l, err
}

func (r *PostgresRepo) GetSubjects() ([]domain.Subject, error) {
	rows, err := r.DB.Query("SELECT id, name FROM subjects")
	if err != nil { return nil, err }
	defer rows.Close()
	var subs []domain.Subject
	for rows.Next() {
		var s domain.Subject
		rows.Scan(&s.ID, &s.Name)
		subs = append(subs, s)
	}
	return subs, nil
}

func (r *PostgresRepo) CreateSubject(name string) (domain.Subject, error) {
	id := uuid.New().String()
	_, err := r.DB.Exec("INSERT INTO subjects (id, name) VALUES ($1, $2)", id, name)
	return domain.Subject{ID: id, Name: name}, err
}

func (r *PostgresRepo) DeleteSubject(id string) error {
	_, err := r.DB.Exec("DELETE FROM subjects WHERE id=$1", id)
	return err
}

func (r *PostgresRepo) GetTopics() ([]domain.Topic, error) {
	rows, err := r.DB.Query("SELECT id, subject_id, name FROM topics")
	if err != nil { return nil, err }
	defer rows.Close()
	var topics []domain.Topic
	for rows.Next() {
		var t domain.Topic
		rows.Scan(&t.ID, &t.SubjectID, &t.Name)
		topics = append(topics, t)
	}
	return topics, nil
}

func (r *PostgresRepo) CreateTopic(name, subjectID string) (domain.Topic, error) {
	id := uuid.New().String()
	_, err := r.DB.Exec("INSERT INTO topics (id, subject_id, name) VALUES ($1, $2, $3)", id, subjectID, name)
	return domain.Topic{ID: id, SubjectID: subjectID, Name: name}, err
}

func (r *PostgresRepo) DeleteTopic(id string) error {
	_, err := r.DB.Exec("DELETE FROM topics WHERE id=$1", id)
	return err
}
