package api

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/lucciano/prds/internal/auth"
	"github.com/lucciano/prds/internal/db"
	"github.com/lucciano/prds/internal/executor"
	"github.com/lucciano/prds/internal/github"
	"github.com/lucciano/prds/internal/repository"
	"github.com/lucciano/prds/internal/service"
	"github.com/lucciano/prds/pkg/config"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// NewRouter sets up the full Gin router with all routes and middleware.
func NewRouter(cfg *config.Config, database *db.DB, redisClient *redis.Client, logger *zap.Logger) http.Handler {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(zapLogger(logger))

	// CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.FrontendURL, "http://localhost:3000", "http://localhost:80"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Repos
	userRepo := repository.NewUserRepo(database)
	exerciseRepo := repository.NewExerciseRepo(database)
	reviewRepo := repository.NewReviewRepo(database)
	projectRepo := repository.NewProjectRepo(database)
	paperRepo := repository.NewPaperRepo(database)
	contribRepo := repository.NewContribRepo(database)
	conceptRepo := repository.NewConceptRepo(database)
	researchRepo := repository.NewResearchRepo(database)

	// Services
	userSvc := service.NewUserService(userRepo, cfg.JWTSecret)
	exerciseSvc := service.NewExerciseService(exerciseRepo, userRepo, redisClient)
	reviewSvc := service.NewReviewService(reviewRepo, exerciseRepo)
	projectSvc := service.NewProjectService(projectRepo)
	paperSvc := service.NewPaperService(paperRepo)
	contribSvc := service.NewContribService(contribRepo)
	conceptSvc := service.NewConceptService(conceptRepo)
	researchSvc := service.NewResearchService(researchRepo)
	ghSvc := github.NewService(cfg)

	// Code executor
	var exec executor.Executor
	if cfg.DockerEnabled {
		exec = executor.NewDockerExecutor(logger)
	} else {
		exec = executor.NewLocalExecutor(logger)
	}

	// Rate limiter (10 code executions/min per IP)
	execLimiter := newIPRateLimiter(rate.Limit(10.0/60), 3)

	// Handlers
	authH := &AuthHandler{userSvc: userSvc, cfg: cfg}
	exerciseH := &ExerciseHandler{svc: exerciseSvc, exec: exec, limiter: execLimiter}
	reviewH := &ReviewHandler{svc: reviewSvc}
	projectH := &ProjectHandler{svc: projectSvc}
	paperH := &PaperHandler{svc: paperSvc}
	contribH := &ContribHandler{svc: contribSvc, ghSvc: ghSvc}
	conceptH := &ConceptHandler{svc: conceptSvc}
	researchH := &ResearchHandler{svc: researchSvc}
	progressH := &ProgressHandler{userSvc: userSvc, exerciseSvc: exerciseSvc, reviewSvc: reviewSvc}
	migrationH := &MigrationHandler{userSvc: userSvc, exerciseSvc: exerciseSvc, reviewSvc: reviewSvc}

	// ─── Public routes ───────────────────────────────────────────────
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "ts": time.Now().Unix()})
	})

	v1 := r.Group("/api/v1")
	{
		// Auth
		v1.POST("/auth/register", authH.Register)
		v1.POST("/auth/login", authH.Login)

		// GitHub OAuth
		v1.GET("/github/auth", contribH.GitHubAuth)
		v1.GET("/github/callback", contribH.GitHubCallback)
	}

	// ─── Authenticated routes ────────────────────────────────────────
	private := v1.Group("/")
	private.Use(auth.Middleware(cfg.JWTSecret))
	{
		// Current user
		private.GET("/me", authH.Me)
		private.PATCH("/me", authH.UpdateMe)

		// Progress summary
		private.GET("/progress", progressH.GetSummary)

		// Exercises
		private.GET("/exercises", exerciseH.List)
		private.GET("/exercises/:id", exerciseH.Get)
		private.POST("/exercises/:id/run", exerciseH.RunCode)
		private.POST("/exercises/:id/submit", exerciseH.Submit)
		private.POST("/exercises/:id/complete", exerciseH.MarkComplete)
		private.POST("/exercises/:id/progress", exerciseH.UpdateProgress)

		// Spaced Repetition
		private.GET("/review/queue", reviewH.GetQueue)
		private.POST("/review/:id", reviewH.SubmitRating)
		private.GET("/review/stats", reviewH.GetStats)

		// Knowledge Graph
		private.GET("/concepts", conceptH.List)
		private.GET("/concepts/graph", conceptH.GetGraph)
		private.PATCH("/concepts/:id/mastery", conceptH.UpdateMastery)

		// Projects
		private.GET("/projects", projectH.List)
		private.POST("/projects", projectH.Create)
		private.GET("/projects/:id", projectH.Get)
		private.PATCH("/projects/:id", projectH.Update)
		private.POST("/projects/:id/milestone", projectH.ToggleMilestone)
		private.PATCH("/projects/:id/advance", projectH.AdvanceStage)

		// Papers
		private.GET("/papers", paperH.List)
		private.POST("/papers", paperH.Create)
		private.GET("/papers/:id", paperH.Get)
		private.PATCH("/papers/:id", paperH.UpdateUserPaper)
		private.POST("/papers/import/arxiv", paperH.ImportArxiv)

		// Contributions
		private.GET("/contributions", contribH.List)
		private.POST("/contributions", contribH.Create)
		private.GET("/contributions/heatmap", contribH.GetHeatmap)
		private.POST("/github/sync", contribH.SyncGitHub)
		private.GET("/github/stats", contribH.GitHubStats)

		// Research Workspace
		private.GET("/hypotheses", researchH.ListHypotheses)
		private.POST("/hypotheses", researchH.CreateHypothesis)
		private.PATCH("/hypotheses/:id", researchH.UpdateHypothesis)
		private.GET("/experiments", researchH.ListExperiments)
		private.POST("/experiments", researchH.CreateExperiment)
		private.PATCH("/experiments/:id", researchH.UpdateExperiment)

		// Analytics
		private.GET("/analytics/velocity", progressH.LearningVelocity)
		private.GET("/analytics/skills", progressH.SkillGrowth)
		private.GET("/analytics/activity", progressH.ActivityHeatmap)

		// Export / Backup
		private.GET("/export/json", progressH.ExportJSON)
		private.GET("/export/markdown", progressH.ExportMarkdown)

		// Migration (localStorage → PostgreSQL)
		private.POST("/migrate/localstorage", migrationH.MigrateLocalStorage)
	}

	return r
}

// zapLogger is a Gin middleware that logs requests using zap.
func zapLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		logger.Info("request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
			zap.String("ip", c.ClientIP()),
		)
	}
}
