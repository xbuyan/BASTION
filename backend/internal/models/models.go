package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ─── JSONB helper ────────────────────────────────────────────────────────────

type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	return json.Marshal(j)
}
func (j *JSONB) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion to []byte failed")
	}
	return json.Unmarshal(b, j)
}

type JSONBSlice []interface{}

func (j JSONBSlice) Value() (driver.Value, error) { return json.Marshal(j) }
func (j *JSONBSlice) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion to []byte failed")
	}
	return json.Unmarshal(b, j)
}

// ─── User ────────────────────────────────────────────────────────────────────

type User struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	Username     string     `json:"username" db:"username"`
	Email        string     `json:"email" db:"email"`
	PasswordHash string     `json:"-" db:"password_hash"`
	XP           int        `json:"xp" db:"xp"`
	Streak       int        `json:"streak" db:"streak"`
	LastActive   *time.Time `json:"last_active" db:"last_active"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
}

// ─── Exercise ────────────────────────────────────────────────────────────────

type Exercise struct {
	ID                  int        `json:"id" db:"id"`
	Phase               int        `json:"phase" db:"phase"`
	Num                 string     `json:"num" db:"num"`
	Title               string     `json:"title" db:"title"`
	Domain              string     `json:"domain" db:"domain"`
	Difficulty          int        `json:"difficulty" db:"difficulty"`
	Tags                JSONBSlice `json:"tags" db:"tags"`
	Description         string     `json:"description" db:"description"`
	StarterCode         string     `json:"starter_code" db:"starter_code"`
	TestCases           JSONB      `json:"test_cases" db:"test_cases"`
	Hints               JSONBSlice `json:"hints" db:"hints"`
	SolutionExplanation string     `json:"solution_explanation" db:"solution_explanation"`
	TimeComplexity      string     `json:"time_complexity" db:"time_complexity"`
	SpaceComplexity     string     `json:"space_complexity" db:"space_complexity"`
	ConceptsCovered     JSONBSlice `json:"concepts_covered" db:"concepts_covered"`
	RelatedExercises    JSONBSlice `json:"related_exercises" db:"related_exercises"`
	CreatedAt           time.Time  `json:"created_at" db:"created_at"`
}

// ─── UserExercise ────────────────────────────────────────────────────────────

type UserExercise struct {
	UserID      uuid.UUID  `json:"user_id" db:"user_id"`
	ExerciseID  int        `json:"exercise_id" db:"exercise_id"`
	Status      string     `json:"status" db:"status"` // not_started, in_progress, completed
	Code        string     `json:"code" db:"code"`
	Notes       string     `json:"notes" db:"notes"`
	CompletedAt *time.Time `json:"completed_at" db:"completed_at"`
}

// ─── ReviewCard ──────────────────────────────────────────────────────────────

type ReviewCard struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	UserID      uuid.UUID  `json:"user_id" db:"user_id"`
	ItemType    string     `json:"item_type" db:"item_type"` // exercise, concept, paper
	ItemID      string     `json:"item_id" db:"item_id"`
	Interval    int        `json:"interval" db:"interval"`
	EaseFactor  float64    `json:"ease_factor" db:"ease_factor"`
	Repetitions int        `json:"repetitions" db:"repetitions"`
	NextReview  time.Time  `json:"next_review" db:"next_review"`
	LastReview  *time.Time `json:"last_review" db:"last_review"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

// SM-2 update: quality 0-5
func (c *ReviewCard) UpdateSM2(quality int) {
	if quality < 3 {
		c.Repetitions = 0
		c.Interval = 1
	} else {
		switch c.Repetitions {
		case 0:
			c.Interval = 1
		case 1:
			c.Interval = 6
		default:
			c.Interval = int(float64(c.Interval) * c.EaseFactor)
		}
		c.Repetitions++
	}
	ef := c.EaseFactor + 0.1 - float64(5-quality)*(0.08+float64(5-quality)*0.02)
	if ef < 1.3 {
		ef = 1.3
	}
	c.EaseFactor = ef
	now := time.Now()
	c.LastReview = &now
	c.NextReview = now.AddDate(0, 0, c.Interval)
}

// ─── Concept ─────────────────────────────────────────────────────────────────

type Concept struct {
	ID          string `json:"id" db:"id"`
	Label       string `json:"label" db:"label"`
	Domain      string `json:"domain" db:"domain"`
	Phase       int    `json:"phase" db:"phase"`
	Description string `json:"description" db:"description"`
}

type ConceptEdge struct {
	ConceptID      string `json:"concept_id" db:"concept_id"`
	PrerequisiteID string `json:"prerequisite_id" db:"prerequisite_id"`
}

type UserConcept struct {
	UserID      uuid.UUID `json:"user_id" db:"user_id"`
	ConceptID   string    `json:"concept_id" db:"concept_id"`
	MasteryLevel int      `json:"mastery_level" db:"mastery_level"` // 0=not_started, 1=learning, 2=mastered
}

// ─── Project ─────────────────────────────────────────────────────────────────

type Project struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	UserID      uuid.UUID  `json:"user_id" db:"user_id"`
	Name        string     `json:"name" db:"name"`
	Description string     `json:"description" db:"description"`
	Phase       int        `json:"phase" db:"phase"`
	Status      string     `json:"status" db:"status"` // concept, prototype, production, published
	GitHubRepo  string     `json:"github_repo" db:"github_repo"`
	Milestones  JSONBSlice `json:"milestones" db:"milestones"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at" db:"updated_at"`
}

// ─── Paper ───────────────────────────────────────────────────────────────────

type Paper struct {
	ID        string     `json:"id" db:"id"` // arXiv ID or custom
	Title     string     `json:"title" db:"title"`
	Authors   string     `json:"authors" db:"authors"`
	Year      int        `json:"year" db:"year"`
	Venue     string     `json:"venue" db:"venue"`
	Tags      JSONBSlice `json:"tags" db:"tags"`
	Abstract  string     `json:"abstract" db:"abstract"`
	URL       string     `json:"url" db:"url"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

type UserPaper struct {
	UserID             uuid.UUID `json:"user_id" db:"user_id"`
	PaperID            string    `json:"paper_id" db:"paper_id"`
	Status             string    `json:"status" db:"status"` // queued, reading, read
	Notes              string    `json:"notes" db:"notes"`
	Questions          string    `json:"questions" db:"questions"`
	ImplementationLink string    `json:"implementation_link" db:"implementation_link"`
}

// ─── Contribution ────────────────────────────────────────────────────────────

type Contribution struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	UserID         uuid.UUID  `json:"user_id" db:"user_id"`
	Repo           string     `json:"repo" db:"repo"`
	Title          string     `json:"title" db:"title"`
	Description    string     `json:"description" db:"description"`
	URL            string     `json:"url" db:"url"`
	Status         string     `json:"status" db:"status"` // open, merged, closed
	Skills         JSONBSlice `json:"skills" db:"skills"`
	ContributedAt  *time.Time `json:"contributed_at" db:"contributed_at"`
	SyncedAt       *time.Time `json:"synced_at" db:"synced_at"`
}

// ─── Hypothesis ──────────────────────────────────────────────────────────────

type Hypothesis struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	UserID      uuid.UUID  `json:"user_id" db:"user_id"`
	Title       string     `json:"title" db:"title"`
	Description string     `json:"description" db:"description"`
	Status      string     `json:"status" db:"status"` // exploring, validated, rejected
	Experiments JSONBSlice `json:"experiments" db:"experiments"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

// ─── Experiment ──────────────────────────────────────────────────────────────

type Experiment struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	UserID         uuid.UUID  `json:"user_id" db:"user_id"`
	HypothesisID   *uuid.UUID `json:"hypothesis_id" db:"hypothesis_id"`
	Title          string     `json:"title" db:"title"`
	Description    string     `json:"description" db:"description"`
	Result         string     `json:"result" db:"result"`
	Status         string     `json:"status" db:"status"` // running, complete, failed
	ConductedAt    *time.Time `json:"conducted_at" db:"conducted_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
}

// ─── UserActivity ─────────────────────────────────────────────────────────────

type UserActivity struct {
	ID           uuid.UUID `json:"id" db:"id"`
	UserID       uuid.UUID `json:"user_id" db:"user_id"`
	ActivityType string    `json:"activity_type" db:"activity_type"`
	ActivityData JSONB     `json:"activity_data" db:"activity_data"`
	XPGained     int       `json:"xp_gained" db:"xp_gained"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// ─── Code Execution ──────────────────────────────────────────────────────────

type ExecutionRequest struct {
	Code     string `json:"code" binding:"required"`
	TestCode string `json:"test_code"`
}

type ExecutionResult struct {
	Stdout   string        `json:"stdout"`
	Stderr   string        `json:"stderr"`
	ExitCode int           `json:"exit_code"`
	Duration time.Duration `json:"duration_ms"`
	Passed   bool          `json:"passed"`
	Error    string        `json:"error,omitempty"`
}

// ─── Auth ────────────────────────────────────────────────────────────────────

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// ─── Progress ────────────────────────────────────────────────────────────────

type ProgressSummary struct {
	XP                  int            `json:"xp"`
	Streak              int            `json:"streak"`
	TotalDone           int            `json:"total_done"`
	PhaseProgress       map[int]int    `json:"phase_progress"` // phase -> % done
	DomainProgress      map[string]int `json:"domain_progress"`
	MasteredConcepts    int            `json:"mastered_concepts"`
	PapersRead          int            `json:"papers_read"`
	Contributions       int            `json:"contributions"`
	ReviewsDue          int            `json:"reviews_due"`
}

// ─── Migration (localStorage → PostgreSQL) ───────────────────────────────────

type LocalStorageExport struct {
	XP                  int               `json:"xp"`
	Streak              int               `json:"streak"`
	CompletedExercises  []int             `json:"completed"`
	InProgressExercises []int             `json:"inProgress"`
	MasteredConcepts    []string          `json:"concepts"`
	ExerciseCodes       map[string]string `json:"codes"`   // "code_1" -> code
	ExerciseNotes       map[string]string `json:"notes"`   // "notes_1" -> notes
	SRCards             map[string]SRCard `json:"sr"`      // "sr_1" -> card
}

type SRCard struct {
	Interval    int     `json:"interval"`
	EaseFactor  float64 `json:"easeFactor"`
	Repetitions int     `json:"repetitions"`
	NextReview  int64   `json:"nextReview"` // Unix ms
}
