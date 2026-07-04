package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lucciano/bastion/internal/auth"
	"github.com/lucciano/bastion/internal/executor"
	"github.com/lucciano/bastion/internal/github"
	"github.com/lucciano/bastion/internal/models"
	"github.com/lucciano/bastion/internal/service"
	"github.com/lucciano/bastion/pkg/config"
	"golang.org/x/time/rate"
)

// ─── Rate Limiter ─────────────────────────────────────────────────────────────

type ipRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	r        rate.Limit
	b        int
}

func newIPRateLimiter(r rate.Limit, b int) *ipRateLimiter {
	return &ipRateLimiter{limiters: make(map[string]*rate.Limiter), r: r, b: b}
}

func (l *ipRateLimiter) get(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()
	if lim, ok := l.limiters[ip]; ok {
		return lim
	}
	lim := rate.NewLimiter(l.r, l.b)
	l.limiters[ip] = lim
	return lim
}

// ─── Auth Handler ─────────────────────────────────────────────────────────────

type AuthHandler struct {
	userSvc *service.UserService
	cfg     *config.Config
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.userSvc.Register(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.userSvc.Login(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID, ok := auth.UserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	user, err := h.userSvc.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *AuthHandler) UpdateMe(c *gin.Context) {
	userID, ok := auth.UserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var body map[string]string
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.userSvc.Update(c.Request.Context(), userID, body); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ─── Exercise Handler ─────────────────────────────────────────────────────────

type ExerciseHandler struct {
	svc     *service.ExerciseService
	exec    executor.Executor
	limiter *ipRateLimiter
}

func (h *ExerciseHandler) List(c *gin.Context) {
	phase := 0
	if p := c.Query("phase"); p != "" {
		phase, _ = strconv.Atoi(p)
	}
	domain := c.Query("domain")
	userID, _ := auth.UserIDFromContext(c)
	exercises, err := h.svc.List(c.Request.Context(), userID, phase, domain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, exercises)
}

func (h *ExerciseHandler) Get(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	userID, _ := auth.UserIDFromContext(c)
	ex, err := h.svc.GetWithProgress(c.Request.Context(), userID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "exercise not found"})
		return
	}
	c.JSON(http.StatusOK, ex)
}

func (h *ExerciseHandler) RunCode(c *gin.Context) {
	if !h.limiter.get(c.ClientIP()).Allow() {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded — max 10 executions/min"})
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))
	var req models.ExecutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ex, err := h.svc.GetExercise(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "exercise not found"})
		return
	}
	result, err := h.exec.Execute(c.Request.Context(), req.Code, ex.StarterCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ExerciseHandler) Submit(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	userID, _ := auth.UserIDFromContext(c)
	var body struct {
		Code  string `json:"code"`
		Notes string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.SaveProgress(c.Request.Context(), userID, id, "in_progress", body.Code, body.Notes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *ExerciseHandler) MarkComplete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	userID, _ := auth.UserIDFromContext(c)
	if err := h.svc.MarkComplete(c.Request.Context(), userID, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "xp_gained": 100})
}

func (h *ExerciseHandler) UpdateProgress(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	userID, _ := auth.UserIDFromContext(c)
	var body struct {
		Status string `json:"status"`
		Code   string `json:"code"`
		Notes  string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.SaveProgress(c.Request.Context(), userID, id, body.Status, body.Code, body.Notes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ─── Review Handler ───────────────────────────────────────────────────────────

type ReviewHandler struct {
	svc *service.ReviewService
}

func (h *ReviewHandler) GetQueue(c *gin.Context) {
	userID, _ := auth.UserIDFromContext(c)
	queue, err := h.svc.GetDueCards(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, queue)
}

func (h *ReviewHandler) SubmitRating(c *gin.Context) {
	cardID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid card id"})
		return
	}
	userID, _ := auth.UserIDFromContext(c)
	var body struct {
		Quality int `json:"quality" binding:"required,min=0,max=5"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.SubmitRating(c.Request.Context(), userID, cardID, body.Quality); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *ReviewHandler) GetStats(c *gin.Context) {
	userID, _ := auth.UserIDFromContext(c)
	stats, err := h.svc.GetStats(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// ─── Concept Handler ──────────────────────────────────────────────────────────

type ConceptHandler struct {
	svc *service.ConceptService
}

func (h *ConceptHandler) List(c *gin.Context) {
	concepts, err := h.svc.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, concepts)
}

func (h *ConceptHandler) GetGraph(c *gin.Context) {
	userID, _ := auth.UserIDFromContext(c)
	graph, err := h.svc.GetGraph(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, graph)
}

func (h *ConceptHandler) UpdateMastery(c *gin.Context) {
	userID, _ := auth.UserIDFromContext(c)
	conceptID := c.Param("id")
	var body struct {
		Level int `json:"level" binding:"min=0,max=2"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.UpdateMastery(c.Request.Context(), userID, conceptID, body.Level); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ─── Project Handler ──────────────────────────────────────────────────────────

type ProjectHandler struct {
	svc *service.ProjectService
}

func (h *ProjectHandler) List(c *gin.Context) {
	userID, _ := auth.UserIDFromContext(c)
	projects, err := h.svc.List(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, projects)
}

func (h *ProjectHandler) Create(c *gin.Context) {
	userID, _ := auth.UserIDFromContext(c)
	var proj models.Project
	if err := c.ShouldBindJSON(&proj); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	proj.UserID = userID
	if err := h.svc.Create(c.Request.Context(), &proj); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, proj)
}

func (h *ProjectHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	proj, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, proj)
}

func (h *ProjectHandler) Update(c *gin.Context) {
	id, _ := uuid.Parse(c.Param("id"))
	userID, _ := auth.UserIDFromContext(c)
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.Update(c.Request.Context(), userID, id, body); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *ProjectHandler) ToggleMilestone(c *gin.Context) {
	id, _ := uuid.Parse(c.Param("id"))
	userID, _ := auth.UserIDFromContext(c)
	var body struct {
		Index int `json:"index"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.ToggleMilestone(c.Request.Context(), userID, id, body.Index); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "xp_gained": 25})
}

func (h *ProjectHandler) AdvanceStage(c *gin.Context) {
	id, _ := uuid.Parse(c.Param("id"))
	userID, _ := auth.UserIDFromContext(c)
	newStatus, err := h.svc.AdvanceStage(c.Request.Context(), userID, id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": newStatus, "xp_gained": 200})
}

// ─── Paper Handler ────────────────────────────────────────────────────────────

type PaperHandler struct {
	svc *service.PaperService
}

func (h *PaperHandler) List(c *gin.Context) {
	userID, _ := auth.UserIDFromContext(c)
	papers, err := h.svc.ListWithStatus(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, papers)
}

func (h *PaperHandler) Create(c *gin.Context) {
	userID, _ := auth.UserIDFromContext(c)
	var paper models.Paper
	if err := c.ShouldBindJSON(&paper); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.Create(c.Request.Context(), userID, &paper); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, paper)
}

func (h *PaperHandler) Get(c *gin.Context) {
	id := c.Param("id")
	userID, _ := auth.UserIDFromContext(c)
	paper, err := h.svc.GetWithUserData(c.Request.Context(), userID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, paper)
}

func (h *PaperHandler) UpdateUserPaper(c *gin.Context) {
	id := c.Param("id")
	userID, _ := auth.UserIDFromContext(c)
	var up models.UserPaper
	if err := c.ShouldBindJSON(&up); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	up.PaperID = id
	up.UserID = userID
	if err := h.svc.UpdateUserPaper(c.Request.Context(), &up); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *PaperHandler) ImportArxiv(c *gin.Context) {
	userID, _ := auth.UserIDFromContext(c)
	var body struct {
		ArxivID string `json:"arxiv_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	paper, err := h.svc.ImportFromArxiv(c.Request.Context(), userID, body.ArxivID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, paper)
}

// ─── Contrib Handler ──────────────────────────────────────────────────────────

type ContribHandler struct {
	svc   *service.ContribService
	ghSvc *github.Service
}

func (h *ContribHandler) List(c *gin.Context) {
	userID, _ := auth.UserIDFromContext(c)
	contribs, err := h.svc.List(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, contribs)
}

func (h *ContribHandler) Create(c *gin.Context) {
	userID, _ := auth.UserIDFromContext(c)
	var contrib models.Contribution
	if err := c.ShouldBindJSON(&contrib); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	contrib.UserID = userID
	if err := h.svc.Create(c.Request.Context(), &contrib); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, contrib)
}

func (h *ContribHandler) GetHeatmap(c *gin.Context) {
	userID, _ := auth.UserIDFromContext(c)
	heatmap, err := h.svc.GetHeatmap(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, heatmap)
}

func (h *ContribHandler) GitHubAuth(c *gin.Context) {
	url := h.ghSvc.AuthURL()
	c.Redirect(http.StatusTemporaryRedirect, url)
}

func (h *ContribHandler) GitHubCallback(c *gin.Context) {
	code := c.Query("code")
	token, err := h.ghSvc.ExchangeCode(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "github auth failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"github_token": token})
}

func (h *ContribHandler) SyncGitHub(c *gin.Context) {
	userID, _ := auth.UserIDFromContext(c)
	var body struct {
		GitHubToken string `json:"github_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	synced, err := h.svc.SyncFromGitHub(c.Request.Context(), userID, body.GitHubToken, h.ghSvc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"synced": synced})
}

func (h *ContribHandler) GitHubStats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "provide github_token in /github/sync first"})
}

// ─── Research Handler ─────────────────────────────────────────────────────────

type ResearchHandler struct {
	svc *service.ResearchService
}

func (h *ResearchHandler) ListHypotheses(c *gin.Context) {
	userID, _ := auth.UserIDFromContext(c)
	items, err := h.svc.ListHypotheses(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *ResearchHandler) CreateHypothesis(c *gin.Context) {
	userID, _ := auth.UserIDFromContext(c)
	var h2 models.Hypothesis
	if err := c.ShouldBindJSON(&h2); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h2.UserID = userID
	if err := h.svc.CreateHypothesis(c.Request.Context(), &h2); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, h2)
}

func (h *ResearchHandler) UpdateHypothesis(c *gin.Context) {
	id, _ := uuid.Parse(c.Param("id"))
	userID, _ := auth.UserIDFromContext(c)
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.UpdateHypothesis(c.Request.Context(), userID, id, body); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *ResearchHandler) ListExperiments(c *gin.Context) {
	userID, _ := auth.UserIDFromContext(c)
	items, err := h.svc.ListExperiments(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *ResearchHandler) CreateExperiment(c *gin.Context) {
	userID, _ := auth.UserIDFromContext(c)
	var exp models.Experiment
	if err := c.ShouldBindJSON(&exp); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	exp.UserID = userID
	if err := h.svc.CreateExperiment(c.Request.Context(), &exp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, exp)
}

func (h *ResearchHandler) UpdateExperiment(c *gin.Context) {
	id, _ := uuid.Parse(c.Param("id"))
	userID, _ := auth.UserIDFromContext(c)
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.UpdateExperiment(c.Request.Context(), userID, id, body); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ─── Progress Handler ─────────────────────────────────────────────────────────

type ProgressHandler struct {
	userSvc     *service.UserService
	exerciseSvc *service.ExerciseService
	reviewSvc   *service.ReviewService
}

func (h *ProgressHandler) GetSummary(c *gin.Context) {
	userID, _ := auth.UserIDFromContext(c)
	summary, err := h.exerciseSvc.GetProgressSummary(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *ProgressHandler) LearningVelocity(c *gin.Context) {
	userID, _ := auth.UserIDFromContext(c)
	data, err := h.exerciseSvc.LearningVelocity(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *ProgressHandler) SkillGrowth(c *gin.Context) {
	userID, _ := auth.UserIDFromContext(c)
	data, err := h.exerciseSvc.SkillGrowth(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *ProgressHandler) ActivityHeatmap(c *gin.Context) {
	userID, _ := auth.UserIDFromContext(c)
	data, err := h.exerciseSvc.ActivityHeatmap(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *ProgressHandler) ExportJSON(c *gin.Context) {
	userID, _ := auth.UserIDFromContext(c)
	data, err := h.exerciseSvc.ExportJSON(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Header("Content-Disposition", "attachment; filename=prds-export.json")
	c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, data)
}

func (h *ProgressHandler) ExportMarkdown(c *gin.Context) {
	userID, _ := auth.UserIDFromContext(c)
	md, err := h.exerciseSvc.ExportMarkdown(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Header("Content-Disposition", "attachment; filename=prds-progress.md")
	c.Header("Content-Type", "text/markdown")
	c.String(http.StatusOK, md)
}

// ─── Migration Handler ────────────────────────────────────────────────────────

type MigrationHandler struct {
	userSvc     *service.UserService
	exerciseSvc *service.ExerciseService
	reviewSvc   *service.ReviewService
}

func (h *MigrationHandler) MigrateLocalStorage(c *gin.Context) {
	userID, _ := auth.UserIDFromContext(c)
	var export models.LocalStorageExport
	if err := c.ShouldBindJSON(&export); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	count := 0
	ctx := c.Request.Context()

	// Migrate completed exercises
	for _, id := range export.CompletedExercises {
		code := export.ExerciseCodes["code_"+strconv.Itoa(id)]
		notes := export.ExerciseNotes["notes_"+strconv.Itoa(id)]
		h.exerciseSvc.SaveProgress(ctx, userID, id, "completed", code, notes)
		count++
	}

	// Migrate in-progress
	for _, id := range export.InProgressExercises {
		code := export.ExerciseCodes["code_"+strconv.Itoa(id)]
		notes := export.ExerciseNotes["notes_"+strconv.Itoa(id)]
		h.exerciseSvc.SaveProgress(ctx, userID, id, "in_progress", code, notes)
		count++
	}

	// Migrate XP
	if export.XP > 0 {
		h.userSvc.AddXP(ctx, userID, export.XP)
	}

	// Migrate SR cards
	for key, card := range export.SRCards {
		var itemID string
		fmt.Sscanf(key, "sr_%s", &itemID)
		nextReview := time.UnixMilli(card.NextReview)
		h.reviewSvc.UpsertCard(ctx, userID, "exercise", itemID, card.Interval, card.EaseFactor, card.Repetitions, nextReview)
	}

	c.JSON(http.StatusOK, gin.H{
		"migrated": count,
		"xp":       export.XP,
		"sr_cards": len(export.SRCards),
	})
}

// fmt needed for Sscanf
var _ = json.Marshal
var _ = fmt.Sprintf
var _ = time.Now
