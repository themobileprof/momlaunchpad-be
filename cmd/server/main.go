package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/themobileprof/momlaunchpad-be/internal/api"
	"github.com/themobileprof/momlaunchpad-be/internal/api/middleware"
	"github.com/themobileprof/momlaunchpad-be/internal/calendar"
	"github.com/themobileprof/momlaunchpad-be/internal/classifier"
	"github.com/themobileprof/momlaunchpad-be/internal/db"
	"github.com/themobileprof/momlaunchpad-be/internal/language"
	"github.com/themobileprof/momlaunchpad-be/internal/memory"
	"github.com/themobileprof/momlaunchpad-be/internal/prompt"
	"github.com/themobileprof/momlaunchpad-be/internal/ws"
	"github.com/themobileprof/momlaunchpad-be/pkg/deepseek"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Get configuration from environment
	port := getEnv("PORT", "8080")
	databaseURL := getEnv("DATABASE_URL", "")
	deepseekAPIKey := getEnv("DEEPSEEK_API_KEY", "")
	jwtSecret := getEnv("JWT_SECRET", "")

	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	if deepseekAPIKey == "" {
		log.Fatal("DEEPSEEK_API_KEY is required")
	}
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	// Initialize database
	database, err := db.NewFromURL(databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	log.Println("✅ Database connected")

	// Initialize components
	cls := classifier.NewClassifier()
	memMgr := memory.NewMemoryManager(10) // Keep last 10 messages
	deepseekClient := deepseek.NewHTTPClient(deepseek.Config{
		APIKey: deepseekAPIKey,
	})
	promptBuilder := prompt.NewBuilder()
	calSuggester := calendar.NewSuggester()
	langMgr := language.NewManager()

	// Load enabled languages from database
	ctx := context.Background()
	languages, err := database.GetEnabledLanguages(ctx)
	if err != nil {
		log.Printf("Warning: Failed to load languages: %v", err)
	} else {
		for _, lang := range languages {
			langMgr.AddLanguage(language.LanguageInfo{
				Code:           lang.Code,
				Name:           lang.Name,
				NativeName:     lang.NativeName,
				IsEnabled:      lang.IsEnabled,
				IsExperimental: lang.IsExperimental,
			})
		}
		log.Printf("✅ Loaded %d languages", len(languages))
	}

	// Initialize handlers
	authHandler := api.NewAuthHandler(database, jwtSecret)
	calendarHandler := api.NewCalendarHandler(database)
	savingsHandler := api.NewSavingsHandler(database)
	chatHandler := ws.NewChatHandler(
		cls,
		memMgr,
		promptBuilder,
		deepseekClient,
		calSuggester,
		langMgr,
		database,
		jwtSecret,
	)

	// Setup Gin router
	router := gin.Default()

	// Apply CORS middleware
	router.Use(middleware.CORS())

	// Apply global rate limiting (100 req/sec per IP, burst of 200)
	router.Use(middleware.PerIP(100.0/60.0, 200)) // ~1.67 req/sec = 100/min

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"time":   time.Now().Unix(),
		})
	})

	// Auth routes (public)
	auth := router.Group("/api/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.GET("/me", middleware.JWTAuth(jwtSecret), authHandler.Me)
	}

	// Calendar routes (protected + per-user rate limiting)
	calendarGroup := router.Group("/api/reminders")
	calendarGroup.Use(middleware.JWTAuth(jwtSecret))
	calendarGroup.Use(middleware.PerUser(500.0/3600.0, 100)) // 500/hour per user
	{
		calendarGroup.GET("", calendarHandler.GetReminders)
		calendarGroup.POST("", calendarHandler.CreateReminder)
		calendarGroup.PUT("/:id", calendarHandler.UpdateReminder)
		calendarGroup.DELETE("/:id", calendarHandler.DeleteReminder)
	}

	// Savings routes (protected + per-user rate limiting)
	savingsGroup := router.Group("/api/savings")
	savingsGroup.Use(middleware.JWTAuth(jwtSecret))
	savingsGroup.Use(middleware.PerUser(500.0/3600.0, 100)) // 500/hour per user
	{
		savingsGroup.GET("/summary", savingsHandler.GetSavingsSummary)
		savingsGroup.GET("/entries", savingsHandler.GetSavingsEntries)
		savingsGroup.POST("/entries", savingsHandler.CreateSavingsEntry)
		savingsGroup.PUT("/edd", savingsHandler.UpdateEDD)
		savingsGroup.PUT("/goal", savingsHandler.UpdateSavingsGoal)
	}

	// WebSocket chat route (protected via query param/header)
	router.GET("/ws/chat", chatHandler.HandleChat)

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		log.Printf("🚀 Server starting on http://localhost:%s", port)
		log.Printf("📝 API endpoints:")
		log.Printf("   POST   /api/auth/register")
		log.Printf("   POST   /api/auth/login")
		log.Printf("   GET    /api/auth/me")
		log.Printf("   GET    /api/reminders")
		log.Printf("   POST   /api/reminders")
		log.Printf("   PUT    /api/reminders/:id")
		log.Printf("   DELETE /api/reminders/:id")
		log.Printf("   GET    /api/savings/summary")
		log.Printf("   GET    /api/savings/entries")
		log.Printf("   POST   /api/savings/entries")
		log.Printf("   PUT    /api/savings/edd")
		log.Printf("   PUT    /api/savings/goal")
		log.Printf("   WS     /ws/chat")
		log.Printf("")
		log.Printf("Press Ctrl+C to stop")

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
