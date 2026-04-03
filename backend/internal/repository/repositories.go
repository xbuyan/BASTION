package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lucciano/prds/internal/db"
	"github.com/lucciano/prds/internal/models"
)

// ─── UserRepo ─────────────────────────────────────────────────────────────────

type UserRepo struct{ db *db.DB }

func NewUserRepo(database *db.DB) *UserRepo { return &UserRepo{db: database} }

func (r *UserRepo) Create(ctx context.Context, u *models.User) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id,username,email,password_hash,xp,streak,created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		u.ID, u.Username, u.Email, u.PasswordHash, u.XP, u.Streak, u.CreatedAt,
	)
	return err
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	u := &models.User{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id,username,email,password_hash,xp,streak,last_active,created_at FROM users WHERE email=$1`, email,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.XP, &u.Streak, &u.LastActive, &u.CreatedAt)
	return u, err
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	u := &models.User{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id,username,email,xp,streak,last_active,created_at FROM users WHERE id=$1`, id,
	).Scan(&u.ID, &u.Username, &u.Email, &u.XP, &u.Streak, &u.LastActive, &u.CreatedAt)
	return u, err
}

func (r *UserRepo) AddXP(ctx context.Context, id uuid.UUID, amount int) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET xp = xp + $1 WHERE id = $2`, amount, id)
	return err
}

func (r *UserRepo) Update(ctx context.Context, id uuid.UUID, fields map[string]string) error {
	if username, ok := fields["username"]; ok {
		_, err := r.db.ExecContext(ctx, `UPDATE users SET username=$1 WHERE id=$2`, username, id)
		return err
	}
	return nil
}

func (r *UserRepo) UpdateLastActive(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET last_active = NOW() WHERE id = $1`, id)
	return err
}

// ─── ExerciseRepo ─────────────────────────────────────────────────────────────

type ExerciseRepo struct{ db *db.DB }

func NewExerciseRepo(database *db.DB) *ExerciseRepo { return &ExerciseRepo{db: database} }

func (r *ExerciseRepo) List(ctx context.Context, phase int, domain string) ([]models.Exercise, error) {
	query := `SELECT id,phase,num,title,domain,difficulty,description,starter_code,time_complexity,space_complexity,created_at FROM exercises WHERE 1=1`
	args := []interface{}{}
	n := 1
	if phase > 0 {
		query += fmt.Sprintf(" AND phase=$%d", n)
		args = append(args, phase)
		n++
	}
	if domain != "" {
		query += fmt.Sprintf(" AND domain=$%d", n)
		args = append(args, domain)
	}
	query += " ORDER BY phase, id"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exercises []models.Exercise
	for rows.Next() {
		var ex models.Exercise
		if err := rows.Scan(&ex.ID, &ex.Phase, &ex.Num, &ex.Title, &ex.Domain, &ex.Difficulty,
			&ex.Description, &ex.StarterCode, &ex.TimeComplexity, &ex.SpaceComplexity, &ex.CreatedAt); err != nil {
			continue
		}
		exercises = append(exercises, ex)
	}
	return exercises, nil
}

func (r *ExerciseRepo) GetByID(ctx context.Context, id int) (*models.Exercise, error) {
	ex := &models.Exercise{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id,phase,num,title,domain,difficulty,description,starter_code,solution_explanation,
		 time_complexity,space_complexity,created_at FROM exercises WHERE id=$1`, id,
	).Scan(&ex.ID, &ex.Phase, &ex.Num, &ex.Title, &ex.Domain, &ex.Difficulty,
		&ex.Description, &ex.StarterCode, &ex.SolutionExplanation,
		&ex.TimeComplexity, &ex.SpaceComplexity, &ex.CreatedAt)
	return ex, err
}

func (r *ExerciseRepo) GetUserProgress(ctx context.Context, userID uuid.UUID) (map[int]*models.UserExercise, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT exercise_id,status,code,notes,completed_at FROM user_exercises WHERE user_id=$1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[int]*models.UserExercise)
	for rows.Next() {
		ue := &models.UserExercise{}
		rows.Scan(&ue.ExerciseID, &ue.Status, &ue.Code, &ue.Notes, &ue.CompletedAt)
		result[ue.ExerciseID] = ue
	}
	return result, nil
}

func (r *ExerciseRepo) GetUserExercise(ctx context.Context, userID uuid.UUID, exerciseID int) (*models.UserExercise, error) {
	ue := &models.UserExercise{}
	err := r.db.QueryRowContext(ctx,
		`SELECT user_id,exercise_id,status,code,notes,completed_at FROM user_exercises WHERE user_id=$1 AND exercise_id=$2`,
		userID, exerciseID,
	).Scan(&ue.UserID, &ue.ExerciseID, &ue.Status, &ue.Code, &ue.Notes, &ue.CompletedAt)
	return ue, err
}

func (r *ExerciseRepo) UpsertUserExercise(ctx context.Context, ue *models.UserExercise) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO user_exercises (user_id,exercise_id,status,code,notes,completed_at)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 ON CONFLICT (user_id,exercise_id) DO UPDATE SET
		   status=EXCLUDED.status,
		   code=COALESCE(NULLIF(EXCLUDED.code,''), user_exercises.code),
		   notes=COALESCE(NULLIF(EXCLUDED.notes,''), user_exercises.notes),
		   completed_at=COALESCE(EXCLUDED.completed_at, user_exercises.completed_at)`,
		ue.UserID, ue.ExerciseID, ue.Status, ue.Code, ue.Notes, ue.CompletedAt,
	)
	return err
}

func (r *ExerciseRepo) LearningVelocity(ctx context.Context, userID uuid.UUID) (interface{}, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT DATE(completed_at) as day, COUNT(*) FROM user_exercises
		 WHERE user_id=$1 AND status='completed' AND completed_at IS NOT NULL
		 GROUP BY day ORDER BY day DESC LIMIT 30`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type DayCount struct {
		Day   string `json:"day"`
		Count int    `json:"count"`
	}
	var result []DayCount
	for rows.Next() {
		var dc DayCount
		rows.Scan(&dc.Day, &dc.Count)
		result = append(result, dc)
	}
	return result, nil
}

func (r *ExerciseRepo) SkillGrowth(ctx context.Context, userID uuid.UUID) (interface{}, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT e.domain, COUNT(*) FILTER (WHERE ue.status='completed') as done, COUNT(*) as total
		 FROM exercises e LEFT JOIN user_exercises ue ON e.id=ue.exercise_id AND ue.user_id=$1
		 GROUP BY e.domain`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type DomainSkill struct {
		Domain string `json:"domain"`
		Done   int    `json:"done"`
		Total  int    `json:"total"`
		Pct    int    `json:"pct"`
	}
	var result []DomainSkill
	for rows.Next() {
		var ds DomainSkill
		rows.Scan(&ds.Domain, &ds.Done, &ds.Total)
		if ds.Total > 0 {
			ds.Pct = int(float64(ds.Done) / float64(ds.Total) * 100)
		}
		result = append(result, ds)
	}
	return result, nil
}

func (r *ExerciseRepo) ActivityHeatmap(ctx context.Context, userID uuid.UUID) (interface{}, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT DATE(completed_at) as day, COUNT(*) FROM user_exercises
		 WHERE user_id=$1 AND status='completed' AND completed_at > NOW()-INTERVAL '1 year'
		 GROUP BY day`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]int)
	for rows.Next() {
		var day string
		var count int
		rows.Scan(&day, &count)
		result[day] = count
	}
	return result, nil
}

func (r *ExerciseRepo) ExportAll(ctx context.Context, userID uuid.UUID) (interface{}, error) {
	progress, err := r.GetUserProgress(ctx, userID)
	if err != nil {
		return nil, err
	}
	exercises, _ := r.List(ctx, 0, "")
	return map[string]interface{}{
		"exercises": exercises,
		"progress":  progress,
		"exported_at": time.Now(),
	}, nil
}

// ─── ReviewRepo ───────────────────────────────────────────────────────────────

type ReviewRepo struct{ db *db.DB }

func NewReviewRepo(database *db.DB) *ReviewRepo { return &ReviewRepo{db: database} }

func (r *ReviewRepo) GetDue(ctx context.Context, userID uuid.UUID) ([]*models.ReviewCard, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id,user_id,item_type,item_id,interval,ease_factor,repetitions,next_review,last_review,created_at
		 FROM review_cards WHERE user_id=$1 AND next_review <= NOW() ORDER BY next_review LIMIT 50`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cards []*models.ReviewCard
	for rows.Next() {
		c := &models.ReviewCard{}
		rows.Scan(&c.ID, &c.UserID, &c.ItemType, &c.ItemID, &c.Interval, &c.EaseFactor, &c.Repetitions,
			&c.NextReview, &c.LastReview, &c.CreatedAt)
		cards = append(cards, c)
	}
	return cards, nil
}

func (r *ReviewRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.ReviewCard, error) {
	c := &models.ReviewCard{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id,user_id,item_type,item_id,interval,ease_factor,repetitions,next_review,last_review,created_at
		 FROM review_cards WHERE id=$1`, id,
	).Scan(&c.ID, &c.UserID, &c.ItemType, &c.ItemID, &c.Interval, &c.EaseFactor, &c.Repetitions,
		&c.NextReview, &c.LastReview, &c.CreatedAt)
	return c, err
}

func (r *ReviewRepo) Upsert(ctx context.Context, card *models.ReviewCard) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO review_cards (id,user_id,item_type,item_id,interval,ease_factor,repetitions,next_review,last_review,created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		 ON CONFLICT (id) DO UPDATE SET
		   interval=EXCLUDED.interval, ease_factor=EXCLUDED.ease_factor,
		   repetitions=EXCLUDED.repetitions, next_review=EXCLUDED.next_review, last_review=EXCLUDED.last_review`,
		card.ID, card.UserID, card.ItemType, card.ItemID,
		card.Interval, card.EaseFactor, card.Repetitions,
		card.NextReview, card.LastReview, card.CreatedAt,
	)
	return err
}

func (r *ReviewRepo) GetStats(ctx context.Context, userID uuid.UUID) (interface{}, error) {
	var total, due, mastered int
	r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM review_cards WHERE user_id=$1`, userID).Scan(&total)
	r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM review_cards WHERE user_id=$1 AND next_review <= NOW()`, userID).Scan(&due)
	r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM review_cards WHERE user_id=$1 AND repetitions >= 5`, userID).Scan(&mastered)
	return map[string]int{"total": total, "due": due, "mastered": mastered}, nil
}

// ─── ProjectRepo ──────────────────────────────────────────────────────────────

type ProjectRepo struct{ db *db.DB }

func NewProjectRepo(database *db.DB) *ProjectRepo { return &ProjectRepo{db: database} }

func (r *ProjectRepo) List(ctx context.Context, userID uuid.UUID) ([]*models.Project, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id,user_id,name,description,phase,status,github_repo,milestones,created_at FROM projects WHERE user_id=$1 ORDER BY phase`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var projects []*models.Project
	for rows.Next() {
		p := &models.Project{}
		rows.Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.Phase, &p.Status, &p.GitHubRepo, &p.Milestones, &p.CreatedAt)
		projects = append(projects, p)
	}
	return projects, nil
}

func (r *ProjectRepo) Create(ctx context.Context, p *models.Project) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO projects (id,user_id,name,description,phase,status,github_repo,milestones,created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		p.ID, p.UserID, p.Name, p.Description, p.Phase, p.Status, p.GitHubRepo, p.Milestones, p.CreatedAt,
	)
	return err
}

func (r *ProjectRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Project, error) {
	p := &models.Project{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id,user_id,name,description,phase,status,github_repo,milestones,created_at FROM projects WHERE id=$1`, id,
	).Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.Phase, &p.Status, &p.GitHubRepo, &p.Milestones, &p.CreatedAt)
	return p, err
}

func (r *ProjectRepo) Update(ctx context.Context, userID, id uuid.UUID, fields map[string]interface{}) error {
	_, err := r.db.ExecContext(ctx, `UPDATE projects SET updated_at=NOW() WHERE id=$1 AND user_id=$2`, id, userID)
	return err
}

func (r *ProjectRepo) ToggleMilestone(ctx context.Context, userID, id uuid.UUID, idx int) error {
	// Read, toggle, write milestones JSON
	p, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if idx < 0 || idx >= len(p.Milestones) {
		return fmt.Errorf("milestone index out of range")
	}
	ms, ok := p.Milestones[idx].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid milestone format")
	}
	ms["done"] = !toBool(ms["done"])
	p.Milestones[idx] = ms
	_, err = r.db.ExecContext(ctx, `UPDATE projects SET milestones=$1, updated_at=NOW() WHERE id=$2`, p.Milestones, id)
	return err
}

func toBool(v interface{}) bool {
	b, ok := v.(bool)
	return ok && b
}

// ─── PaperRepo ────────────────────────────────────────────────────────────────

type PaperRepo struct{ db *db.DB }

func NewPaperRepo(database *db.DB) *PaperRepo { return &PaperRepo{db: database} }

func (r *PaperRepo) ListWithStatus(ctx context.Context, userID uuid.UUID) (interface{}, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT p.id,p.title,p.authors,p.year,p.venue,p.abstract,
		 COALESCE(up.status,'queued') as status, COALESCE(up.notes,'') as notes
		 FROM papers p LEFT JOIN user_papers up ON p.id=up.paper_id AND up.user_id=$1
		 ORDER BY p.year DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type PaperRow struct {
		ID       string `json:"id"`
		Title    string `json:"title"`
		Authors  string `json:"authors"`
		Year     int    `json:"year"`
		Venue    string `json:"venue"`
		Abstract string `json:"abstract"`
		Status   string `json:"status"`
		Notes    string `json:"notes"`
	}
	var papers []PaperRow
	for rows.Next() {
		var p PaperRow
		rows.Scan(&p.ID, &p.Title, &p.Authors, &p.Year, &p.Venue, &p.Abstract, &p.Status, &p.Notes)
		papers = append(papers, p)
	}
	return papers, nil
}

func (r *PaperRepo) Create(ctx context.Context, userID uuid.UUID, p *models.Paper) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO papers (id,title,authors,year,venue,abstract,url,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		 ON CONFLICT (id) DO NOTHING`,
		p.ID, p.Title, p.Authors, p.Year, p.Venue, p.Abstract, p.URL, p.CreatedAt,
	)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO user_papers (user_id,paper_id,status) VALUES ($1,$2,'queued') ON CONFLICT DO NOTHING`,
		userID, p.ID,
	)
	return err
}

func (r *PaperRepo) GetWithUserData(ctx context.Context, userID uuid.UUID, id string) (interface{}, error) {
	type Result struct {
		models.Paper
		Status   string `json:"status"`
		Notes    string `json:"notes"`
		Questions string `json:"questions"`
	}
	var res Result
	err := r.db.QueryRowContext(ctx,
		`SELECT p.id,p.title,p.authors,p.year,p.venue,p.abstract,p.url,
		 COALESCE(up.status,'queued'), COALESCE(up.notes,''), COALESCE(up.questions,'')
		 FROM papers p LEFT JOIN user_papers up ON p.id=up.paper_id AND up.user_id=$1 WHERE p.id=$2`,
		userID, id,
	).Scan(&res.ID, &res.Title, &res.Authors, &res.Year, &res.Venue, &res.Abstract, &res.URL,
		&res.Status, &res.Notes, &res.Questions)
	return res, err
}

func (r *PaperRepo) UpsertUserPaper(ctx context.Context, up *models.UserPaper) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO user_papers (user_id,paper_id,status,notes,questions,implementation_link)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 ON CONFLICT (user_id,paper_id) DO UPDATE SET
		   status=EXCLUDED.status, notes=EXCLUDED.notes,
		   questions=EXCLUDED.questions, implementation_link=EXCLUDED.implementation_link`,
		up.UserID, up.PaperID, up.Status, up.Notes, up.Questions, up.ImplementationLink,
	)
	return err
}

// ─── ContribRepo ──────────────────────────────────────────────────────────────

type ContribRepo struct{ db *db.DB }

func NewContribRepo(database *db.DB) *ContribRepo { return &ContribRepo{db: database} }

func (r *ContribRepo) List(ctx context.Context, userID uuid.UUID) ([]*models.Contribution, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id,user_id,repo,title,description,url,status,contributed_at FROM contributions WHERE user_id=$1 ORDER BY contributed_at DESC`,
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var contribs []*models.Contribution
	for rows.Next() {
		c := &models.Contribution{}
		rows.Scan(&c.ID, &c.UserID, &c.Repo, &c.Title, &c.Description, &c.URL, &c.Status, &c.ContributedAt)
		contribs = append(contribs, c)
	}
	return contribs, nil
}

func (r *ContribRepo) Create(ctx context.Context, c *models.Contribution) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO contributions (id,user_id,repo,title,description,url,status,contributed_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		c.ID, c.UserID, c.Repo, c.Title, c.Description, c.URL, c.Status, c.ContributedAt,
	)
	return err
}

func (r *ContribRepo) GetHeatmap(ctx context.Context, userID uuid.UUID) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT DATE(contributed_at) as day, COUNT(*) FROM contributions
		 WHERE user_id=$1 AND contributed_at > NOW()-INTERVAL '1 year' GROUP BY day`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]int)
	for rows.Next() {
		var day string
		var count int
		rows.Scan(&day, &count)
		result[day] = count
	}
	return result, nil
}

// ─── ConceptRepo ──────────────────────────────────────────────────────────────

type ConceptRepo struct{ db *db.DB }

func NewConceptRepo(database *db.DB) *ConceptRepo { return &ConceptRepo{db: database} }

func (r *ConceptRepo) List(ctx context.Context) ([]*models.Concept, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,label,domain,phase,description FROM concepts ORDER BY phase`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var concepts []*models.Concept
	for rows.Next() {
		c := &models.Concept{}
		rows.Scan(&c.ID, &c.Label, &c.Domain, &c.Phase, &c.Description)
		concepts = append(concepts, c)
	}
	return concepts, nil
}

func (r *ConceptRepo) GetGraph(ctx context.Context, userID uuid.UUID) (interface{}, error) {
	nodes, _ := r.List(ctx)
	rows, err := r.db.QueryContext(ctx, `SELECT concept_id,prerequisite_id FROM concept_edges`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var edges []models.ConceptEdge
	for rows.Next() {
		e := models.ConceptEdge{}
		rows.Scan(&e.ConceptID, &e.PrerequisiteID)
		edges = append(edges, e)
	}
	return map[string]interface{}{"nodes": nodes, "edges": edges}, nil
}

func (r *ConceptRepo) UpdateMastery(ctx context.Context, uc *models.UserConcept) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO user_concepts (user_id,concept_id,mastery_level) VALUES ($1,$2,$3)
		 ON CONFLICT (user_id,concept_id) DO UPDATE SET mastery_level=EXCLUDED.mastery_level`,
		uc.UserID, uc.ConceptID, uc.MasteryLevel,
	)
	return err
}

// ─── ResearchRepo ─────────────────────────────────────────────────────────────

type ResearchRepo struct{ db *db.DB }

func NewResearchRepo(database *db.DB) *ResearchRepo { return &ResearchRepo{db: database} }

func (r *ResearchRepo) ListHypotheses(ctx context.Context, userID uuid.UUID) ([]*models.Hypothesis, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id,user_id,title,description,status,created_at FROM hypotheses WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*models.Hypothesis
	for rows.Next() {
		h := &models.Hypothesis{}
		rows.Scan(&h.ID, &h.UserID, &h.Title, &h.Description, &h.Status, &h.CreatedAt)
		items = append(items, h)
	}
	return items, nil
}

func (r *ResearchRepo) CreateHypothesis(ctx context.Context, h *models.Hypothesis) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO hypotheses (id,user_id,title,description,status,created_at) VALUES ($1,$2,$3,$4,$5,$6)`,
		h.ID, h.UserID, h.Title, h.Description, h.Status, h.CreatedAt,
	)
	return err
}

func (r *ResearchRepo) UpdateHypothesis(ctx context.Context, userID, id uuid.UUID, fields map[string]interface{}) error {
	if status, ok := fields["status"].(string); ok {
		_, err := r.db.ExecContext(ctx, `UPDATE hypotheses SET status=$1 WHERE id=$2 AND user_id=$3`, status, id, userID)
		return err
	}
	return nil
}

func (r *ResearchRepo) ListExperiments(ctx context.Context, userID uuid.UUID) ([]*models.Experiment, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id,user_id,title,description,result,status,conducted_at,created_at FROM experiments WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*models.Experiment
	for rows.Next() {
		e := &models.Experiment{}
		rows.Scan(&e.ID, &e.UserID, &e.Title, &e.Description, &e.Result, &e.Status, &e.ConductedAt, &e.CreatedAt)
		items = append(items, e)
	}
	return items, nil
}

func (r *ResearchRepo) CreateExperiment(ctx context.Context, exp *models.Experiment) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO experiments (id,user_id,title,description,result,status,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		exp.ID, exp.UserID, exp.Title, exp.Description, exp.Result, exp.Status, exp.CreatedAt,
	)
	return err
}

func (r *ResearchRepo) UpdateExperiment(ctx context.Context, userID, id uuid.UUID, fields map[string]interface{}) error {
	if result, ok := fields["result"].(string); ok {
		_, err := r.db.ExecContext(ctx, `UPDATE experiments SET result=$1 WHERE id=$2 AND user_id=$3`, result, id, userID)
		return err
	}
	return nil
}
