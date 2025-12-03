package domain

type UserRepository interface {
	Create(user User) (User, error)
	GetByEmail(email string) (User, error)
	GetByID(id string) (User, error)
	GetAll() ([]User, error)
	UpdateProfile(id string, profile any) error
	UpdateUser(id string, updates map[string]interface{}) error
	Delete(id string) error
}

type ExamRepository interface {
	Create(exam Exam) error
	GetAll() ([]Exam, error)
	GetByID(id string) (Exam, error)
	Delete(id string) error
}

type QuestionRepository interface {
	Create(q Question) error
	CreateBatch(qs []Question) error
	GetAll() ([]Question, error)
	Delete(id string) error
}

type ResultRepository interface {
	Create(r ExamResult) error
	GetByUserID(userID string) ([]ExamResult, error)
	GetByCompanyID(companyID string) ([]ExamResult, error)
}

type MetaRepository interface {
	GetSubjects() ([]Subject, error)
	CreateSubject(name string) (Subject, error)
	DeleteSubject(id string) error
	GetTopics() ([]Topic, error)
	CreateTopic(name, subjectID string) (Topic, error)
	DeleteTopic(id string) error
}

type LinkRepository interface {
	Create(l PublicLink) error
	GetByCompanyID(companyID string) ([]PublicLink, error)
	GetByToken(token string) (PublicLink, error)
}
