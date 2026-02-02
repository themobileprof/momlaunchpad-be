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
	"github.com/themobileprof/momlaunchpad-be/internal/chat"
	"github.com/themobileprof/momlaunchpad-be/internal/classifier"
	"github.com/themobileprof/momlaunchpad-be/internal/db"
	"github.com/themobileprof/momlaunchpad-be/internal/language"
	"github.com/themobileprof/momlaunchpad-be/internal/memory"
	"github.com/themobileprof/momlaunchpad-be/internal/prompt"
	"github.com/themobileprof/momlaunchpad-be/internal/subscription"
	"github.com/themobileprof/momlaunchpad-be/internal/ws"
	"github.com/themobileprof/momlaunchpad-be/pkg/deepseek"
	"github.com/themobileprof/momlaunchpad-be/pkg/gemini"
	"github.com/themobileprof/momlaunchpad-be/pkg/llm"
	"github.com/themobileprof/momlaunchpad-be/pkg/twilio"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Get configuration from environment
	port := getEnv("PORT", "8080")
	databaseURL := getEnv("DATABASE_URL", "")
	llmProvider := getEnv("LLM_PROVIDER", "deepseek") // Default to deepseek
	deepseekAPIKey := getEnv("DEEPSEEK_API_KEY", "")
	geminiAPIKey := getEnv("GEMINI_API_KEY", "")
	jwtSecret := getEnv("JWT_SECRET", "")
	twilioAccountSID := getEnv("TWILIO_ACCOUNT_SID", "")
	twilioAuthToken := getEnv("TWILIO_AUTH_TOKEN", "")
	twilioPhoneNumber := getEnv("TWILIO_PHONE_NUMBER", "")

	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	if llmProvider == "deepseek" && deepseekAPIKey == "" {
		log.Fatal("DEEPSEEK_API_KEY is required for deepseek provider")
	}
	if llmProvider == "gemini" && geminiAPIKey == "" {
		log.Fatal("GEMINI_API_KEY is required for gemini provider")
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
	memAdapter := db.NewMemoryAdapter(database)
	memMgr := memory.NewMemoryManager(10, memAdapter) // Keep last 10 messages, load from DB

	// Initialize LLM client
	var llmClient llm.Client
	switch llmProvider {
	case "gemini":
		llmClient = gemini.NewHTTPClient(gemini.Config{
			APIKey: geminiAPIKey,
		})
		log.Println("✅ Initialize Gemini LLM client")
	case "deepseek":
		fallthrough
	default:
		llmClient = deepseek.NewHTTPClient(deepseek.Config{
			APIKey: deepseekAPIKey,
		})
		log.Println("✅ Initialize DeepSeek LLM client")
	}

	promptBuilder := prompt.NewBuilder()
	calSuggester := calendar.NewSuggester()
	langMgr := language.NewManager()
	subMgr := subscription.NewManager(database.DB)

	// Initialize Twilio client (optional - only if credentials provided)
	var twilioClient *twilio.VoiceClient
	if twilioAccountSID != "" && twilioAuthToken != "" {
		twilioClient = twilio.NewVoiceClient(twilio.VoiceConfig{
			AccountSID:  twilioAccountSID,
			AuthToken:   twilioAuthToken,
			PhoneNumber: twilioPhoneNumber,
		})
		log.Println("✅ Twilio Voice initialized")
	}

	// Initialize chat engine (shared between WebSocket and Voice)
	chatEngine := chat.NewEngine(
		cls,
		memMgr,
		promptBuilder,
		llmClient,
		calSuggester,
		langMgr,
		database,
	)

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
	oauthHandler := api.NewOAuthHandler(database)
	calendarHandler := api.NewCalendarHandler(database)
	savingsHandler := api.NewSavingsHandler(database)
	subscriptionHandler := api.NewSubscriptionHandler(subMgr)
	symptomHandler := api.NewSymptomHandler(database)
	chatHandler := ws.NewChatHandler(
		chatEngine,
		database,
		jwtSecret,
		subMgr,
	)

	// Initialize voice handler (if Twilio configured)
	var voiceHandler *api.VoiceHandler
	if twilioClient != nil {
		voiceHandler = api.NewVoiceHandler(twilioClient, chatEngine, database)
		log.Println("✅ Voice handler initialized")
	}

	// Setup Gin router
	router := gin.Default()

	// Apply security headers first
	router.Use(middleware.SecurityHeaders())

	// Apply CORS middleware
	router.Use(middleware.CORS())

	// Apply global rate limiting - generous for development/mobile apps
	router.Use(middleware.PerIP(10.0, 50)) // 10 req/sec per IP, burst of 50

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

		// OAuth routes - Web flow (browser redirects)
		auth.GET("/google", oauthHandler.GoogleLogin)
		auth.GET("/google/callback", oauthHandler.GoogleCallback)

		// OAuth routes - Mobile flow (ID token verification)
		auth.POST("/google/token", oauthHandler.GoogleTokenAuth)

		// Apple OAuth (future)
		auth.GET("/apple", oauthHandler.AppleLogin)
		auth.GET("/apple/callback", oauthHandler.AppleCallback)
	}

	// Calendar routes (protected + feature gate + per-user rate limiting)
	calendarGroup := router.Group("/api/reminders")
	calendarGroup.Use(middleware.JWTAuth(jwtSecret))
	calendarGroup.Use(middleware.RequireFeature(subMgr, "calendar"))
	calendarGroup.Use(middleware.PerUser(500.0/3600.0, 100)) // 500/hour per user
	{
		calendarGroup.GET("", calendarHandler.GetReminders)
		calendarGroup.POST("", calendarHandler.CreateReminder)
		calendarGroup.PUT("/:id", calendarHandler.UpdateReminder)
		calendarGroup.DELETE("/:id", calendarHandler.DeleteReminder)
	}

	// Savings routes (protected + feature gate + per-user rate limiting)
	savingsGroup := router.Group("/api/savings")
	savingsGroup.Use(middleware.JWTAuth(jwtSecret))
	savingsGroup.Use(middleware.RequireFeature(subMgr, "savings"))
	savingsGroup.Use(middleware.PerUser(500.0/3600.0, 100)) // 500/hour per user
	{
		savingsGroup.GET("/summary", savingsHandler.GetSavingsSummary)
		savingsGroup.GET("/entries", savingsHandler.GetSavingsEntries)
		savingsGroup.POST("/entries", savingsHandler.CreateSavingsEntry)
		savingsGroup.PUT("/edd", savingsHandler.UpdateEDD)
		savingsGroup.PUT("/goal", savingsHandler.UpdateSavingsGoal)
	}

	// Subscription routes (protected)
	subscriptionGroup := router.Group("/api/subscription")
	subscriptionGroup.Use(middleware.JWTAuth(jwtSecret))
	{
		subscriptionGroup.GET("/me", subscriptionHandler.GetMySubscription)
		subscriptionGroup.GET("/features", subscriptionHandler.GetMyFeatures)
		subscriptionGroup.GET("/quota/:feature", subscriptionHandler.GetMyQuota)
	}

	// Symptom tracking routes (protected)
	symptomGroup := router.Group("/api/symptoms")
	symptomGroup.Use(middleware.JWTAuth(jwtSecret))
	{
		symptomGroup.GET("/history", symptomHandler.GetSymptomHistory)
		symptomGroup.GET("/recent", symptomHandler.GetRecentSymptoms)
		symptomGroup.GET("/stats", symptomHandler.GetSymptomStats)
		symptomGroup.PUT("/:id/resolve", symptomHandler.MarkSymptomResolved)
	}

	// Initialize admin handler
	adminHandler := api.NewAdminHandler(database, langMgr)

	// Admin routes (protected + admin only)
	adminGroup := router.Group("/api/admin")
	adminGroup.Use(middleware.JWTAuth(jwtSecret))
	adminGroup.Use(middleware.AdminOnly()) // Enforce admin role
	{
		// Plan management (CRUD)
		adminGroup.GET("/plans", subscriptionHandler.ListAllPlans)
		adminGroup.POST("/plans", adminHandler.CreatePlan)
		adminGroup.PUT("/plans/:planId", adminHandler.UpdatePlan)
		adminGroup.DELETE("/plans/:planId", adminHandler.DeletePlan)
		adminGroup.GET("/plans/:planId/features", adminHandler.GetPlanFeatures)
		adminGroup.POST("/plans/:planId/features/:featureId", adminHandler.AssignFeatureToPlan)
		adminGroup.DELETE("/plans/:planId/features/:featureId", adminHandler.RemoveFeatureFromPlan)

		// Feature management (CRUD)
		adminGroup.GET("/features", adminHandler.ListFeatures)
		adminGroup.POST("/features", adminHandler.CreateFeature)
		adminGroup.PUT("/features/:featureId", adminHandler.UpdateFeature)
		adminGroup.DELETE("/features/:featureId", adminHandler.DeleteFeature)

		// Language management (CRUD)
		adminGroup.GET("/languages", adminHandler.ListLanguages)
		adminGroup.POST("/languages", adminHandler.CreateLanguage)
		adminGroup.PUT("/languages/:code", adminHandler.UpdateLanguage)
		adminGroup.DELETE("/languages/:code", adminHandler.DeleteLanguage)

		// User subscription management
		adminGroup.GET("/users/:userId/subscription", subscriptionHandler.GetUserSubscription)
		adminGroup.PUT("/users/:userId/plan", subscriptionHandler.UpdateUserPlan)

		// Quota management
		adminGroup.GET("/users/:userId/quota/:feature", subscriptionHandler.GetUserQuotaUsage)
		adminGroup.POST("/users/:userId/quota/:feature/reset", subscriptionHandler.ResetUserQuota)
		adminGroup.GET("/quota/stats", subscriptionHandler.GetQuotaStats)

		// Feature grants to users
		adminGroup.POST("/users/:userId/features", subscriptionHandler.GrantFeature)

		// Analytics
		adminGroup.GET("/analytics/topics", adminHandler.GetChatAnalytics)
		adminGroup.GET("/analytics/users", adminHandler.GetUserStats)
		adminGroup.GET("/analytics/calls", adminHandler.GetCallHistory)

		// System settings management
		adminGroup.GET("/settings", adminHandler.GetSystemSettings)
		adminGroup.GET("/settings/:key", adminHandler.GetSystemSetting)
		adminGroup.PUT("/settings/:key", adminHandler.UpdateSystemSetting)
	}

	// WebSocket chat route (protected via query param/header)
	router.GET("/ws/chat", chatHandler.HandleChat)

	// Twilio Voice routes (public webhooks, but user lookup enforces subscription)
	if voiceHandler != nil {
		voice := router.Group("/api/voice")
		{
			voice.POST("/incoming", voiceHandler.HandleIncoming) // Initial call webhook
			voice.POST("/gather", voiceHandler.HandleGather)     // Speech recognition callback
			voice.POST("/status", voiceHandler.HandleStatus)     // Call status updates
		}
		log.Println("✅ Voice routes registered")
	}

	// Create HTTP server
	// Bind to 0.0.0.0 to accept connections from all network interfaces
	srv := &http.Server{
		Addr:    "0.0.0.0:" + port,
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		log.Printf("🚀 Server starting on 0.0.0.0:%s", port)
		log.Printf("📡 Accessible at http://localhost:%s (local)", port)
		log.Printf("📡 Accessible at http://<your-ip>:%s (network)", port)
		log.Printf("📝 API endpoints:")
		log.Printf("   POST   /api/auth/register")
		log.Printf("   POST   /api/auth/login")
		log.Printf("   GET    /api/auth/me")
		log.Printf("   GET    /api/auth/google (web)")
		log.Printf("   GET    /api/auth/google/callback (web)")
		log.Printf("   POST   /api/auth/google/token (mobile)")
		log.Printf("   GET    /api/auth/apple (coming soon)")
		log.Printf("   GET    /api/auth/apple/callback (coming soon)")
		log.Printf("   GET    /api/reminders")
		log.Printf("   POST   /api/reminders")
		log.Printf("   PUT    /api/reminders/:id")
		log.Printf("   DELETE /api/reminders/:id")
		log.Printf("   GET    /api/savings/summary")
		log.Printf("   GET    /api/savings/entries")
		log.Printf("   POST   /api/savings/entries")
		log.Printf("   PUT    /api/savings/edd")
		log.Printf("   PUT    /api/savings/goal")
		log.Printf("   GET    /api/subscription/me")
		log.Printf("   GET    /api/subscription/features")
		log.Printf("   GET    /api/subscription/quota/:feature")
		log.Printf("   GET    /api/admin/plans")
		log.Printf("   GET    /api/admin/users/:userId/subscription")
		log.Printf("   PUT    /api/admin/users/:userId/plan")
		log.Printf("   GET    /api/admin/users/:userId/quota/:feature")
		log.Printf("   POST   /api/admin/users/:userId/quota/:feature/reset")
		log.Printf("   GET    /api/admin/quota/stats")
		log.Printf("   POST   /api/admin/users/:userId/features")
		log.Printf("   WS     /ws/chat")
		if voiceHandler != nil {
			log.Printf("   POST   /api/voice/incoming (Twilio webhook)")
			log.Printf("   POST   /api/voice/gather (Twilio webhook)")
			log.Printf("   POST   /api/voice/status (Twilio webhook)")
		}
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
