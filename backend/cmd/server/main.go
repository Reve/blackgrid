package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"blackgrid/internal/api/handlers"
	"blackgrid/internal/db"
	"blackgrid/internal/events"
	"blackgrid/internal/monitor"
	"blackgrid/internal/service"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// 1. Structured Logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = handlers.CustomHTTPErrorHandler

	// 2. Middleware
	e.Use(middleware.RequestID())
	e.Use(handlers.StructuredLogger(logger))
	e.Use(middleware.Recover())
	e.Use(handlers.SecurityHeaders())

	// 3. CORS Configuration
	allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:5173,http://localhost:3000"
	}
	origins := strings.Split(allowedOrigins, ",")

	// Default true so the dev frontend on localhost:5173 can carry the
	// session cookie. Operators deploying to production with a wildcard
	// origin must set CORS_ALLOW_CREDENTIALS=false (the browser will
	// reject "*" + credentials anyway). See docs/deployment.md.
	allowCreds := getEnvBool("CORS_ALLOW_CREDENTIALS", true)
	for _, o := range origins {
		if strings.TrimSpace(o) == "*" && allowCreds {
			log.Printf("WARNING: CORS_ALLOWED_ORIGINS contains '*' and CORS_ALLOW_CREDENTIALS=true; browsers will reject this combination. Set explicit origins or disable credentials.")
		}
	}

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     origins,
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAuthorization, echo.HeaderXRequestID},
		AllowCredentials: allowCreds,
	}))

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://blackgrid:blackgrid@localhost:5432/blackgrid?sslmode=disable"
	}

	// 4. Run Migrations
	if err := db.RunMigrations(context.Background(), dbURL); err != nil {
		log.Fatalf("Failed to run migrations: %v\n", err)
	}

	// 5. Database Pool Configuration

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Fatalf("Unable to parse DATABASE_URL: %v\n", err)
	}

	// pgxpool models pool capacity differently from database/sql:
	//   - MaxConns       caps total open connections (DB_MAX_OPEN_CONNS).
	//   - MaxConnLifetime caps a connection's age (DB_CONN_MAX_LIFETIME_MINUTES).
	//   - MaxConnIdleTime caps how long an idle connection lingers
	//     before being closed (DB_CONN_MAX_IDLE_TIME_MINUTES).
	// pgxpool has no direct equivalent of database/sql's "max idle
	// connections" cap — DB_MAX_IDLE_CONNS is therefore unused. The
	// previous assignment of an integer to MaxConnIdleTime (a Duration)
	// silently set it to nanoseconds; that bug is fixed here.
	config.MaxConns = int32(getEnvInt("DB_MAX_OPEN_CONNS", 25))
	config.MaxConnLifetime = time.Duration(getEnvInt("DB_CONN_MAX_LIFETIME_MINUTES", 30)) * time.Minute
	config.MaxConnIdleTime = time.Duration(getEnvInt("DB_CONN_MAX_IDLE_TIME_MINUTES", 10)) * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer pool.Close()

	queries := db.New(pool)
	bus := events.NewEventBus()

	// ---- Services ----
	auditSvc := service.NewAuditService(queries, bus)
	authCfg := service.AuthConfigFromEnv()
	authSvc := service.NewAuthService(queries, authCfg, auditSvc, bus)

	siteSvc := service.NewSiteService(queries, bus)
	vlanSvc := service.NewVlanService(queries, bus)
	prefixSvc := service.NewPrefixService(queries, bus)
	ipSvc := service.NewIPAddressService(queries, bus)
	deviceSvc := service.NewDeviceService(queries, bus)
	discoverySvc := service.NewDiscoveryService(queries, bus)

	// 5. Data Retention Service
	retentionSvc := service.NewRetentionService(queries)
	retentionCfg := service.RetentionConfig{
		MonitorResultsDays:         getEnvInt("RETENTION_MONITOR_RESULTS_DAYS", 90),
		NotificationDeliveriesDays: getEnvInt("RETENTION_NOTIFICATION_DELIVERIES_DAYS", 30),
		AuditLogDays:               getEnvInt("RETENTION_AUDIT_LOG_DAYS", 365),
		DiscoveryResultsDays:       getEnvInt("RETENTION_DISCOVERY_RESULTS_DAYS", 90),
		DiscoveryScansDays:         getEnvInt("RETENTION_DISCOVERY_SCANS_DAYS", 90),
		IntervalHours:              getEnvInt("RETENTION_CLEANUP_INTERVAL_HOURS", 24),
	}
	retentionCtx, cancelRetention := context.WithCancel(context.Background())
	go retentionSvc.Start(retentionCtx, retentionCfg)

	// Start background scheduled scans
	schedCtx, cancelScheduler := context.WithCancel(context.Background())
	schedDone := make(chan struct{})
	go func() {
		defer close(schedDone)
		discoverySvc.RunScheduledScans(schedCtx)
	}()

	h := handlers.New(siteSvc, vlanSvc, prefixSvc, ipSvc, deviceSvc, discoverySvc, auditSvc)

	incidentSvc := service.NewIncidentService(queries, bus)
	notificationSvc := service.NewNotificationService(queries, bus)
	incidentSvc.SetNotifier(notificationSvc)

	monitorRunner := monitor.NewRunner(queries)
	monitorScheduler := monitor.NewScheduler(queries, monitorRunner, getEnvInt("MONITOR_WORKERS", 10), bus)
	incidentHook := service.NewIncidentHook(incidentSvc)
	monitorScheduler.SetIncidentHook(incidentHook)
	monitorScheduler.Start()

	monitorHandler := handlers.NewMonitorHandler(queries, monitorRunner, auditSvc, bus)
	monitorHandler.SetIncidentHook(incidentHook)
	incidentHandler := handlers.NewIncidentHandler(incidentSvc)
	notificationHandler := handlers.NewNotificationHandler(notificationSvc)
	notificationHandler.SetAuditService(auditSvc)

	statusPageSvc := service.NewStatusPageService(queries, bus)
	statusPageHandler := handlers.NewStatusPageHandler(statusPageSvc)
	statusPageHandler.SetAuditService(auditSvc)

	eventHandler := handlers.NewEventHandler(bus)

	authHandler := handlers.NewAuthHandler(authSvc, auditSvc, authCfg)

	// ---- Auth middleware ----
	authMW := handlers.AuthMiddleware(authSvc)
	adminMW := handlers.RequireAdmin()
	operatorMW := handlers.RequireOperator()

	// 6. Rate Limiting
	loginLimiter := handlers.RateLimitMiddleware(2.0/300.0, 10, handlers.ErrCodeRateLimited, "Too many login attempts") // 10 per 5m
	setupLimiter := handlers.RateLimitMiddleware(1.0/120.0, 5, handlers.ErrCodeRateLimited, "Too many setup attempts") // 5 per 10m
	apiTokenLimiter := handlers.UserRateLimitMiddleware(30.0/3600.0, 30, handlers.ErrCodeRateLimited, "API token creation limit exceeded")
	notifTestLimiter := handlers.UserRateLimitMiddleware(20.0/3600.0, 20, handlers.ErrCodeRateLimited, "Notification test limit exceeded")
	monitorTestLimiter := handlers.UserRateLimitMiddleware(60.0/3600.0, 60, handlers.ErrCodeRateLimited, "Monitor test limit exceeded")
	pushLimiter := handlers.RateLimitMiddleware(2.0, 120, handlers.ErrCodeRateLimited, "Push rate limit exceeded") // 120 per min

	// ---- Public status endpoint (no auth required) ----
	e.GET("/status/:slug", statusPageHandler.PublicStatusPage)

	// ---- Push heartbeat endpoint (token-authenticated, no user auth) ----
	e.GET("/push/:token", monitorHandler.ReceivePushHeartbeat, pushLimiter)
	e.POST("/push/:token", monitorHandler.ReceivePushHeartbeat, pushLimiter)

	// 7. Metrics endpoint
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	// ---- API v1 group ----
	v1 := e.Group("/api/v1")

	// Public routes (no auth required)
	v1.GET("/health", h.Health)
	v1.GET("/ready", func(c echo.Context) error {
		if err := pool.Ping(c.Request().Context()); err != nil {
			return handlers.Error(c, handlers.ErrCodeServiceUnavailable, "Database unavailable", nil)
		}
		return c.JSON(http.StatusOK, map[string]string{
			"status":   "ok",
			"database": "ok",
			"time":     time.Now().Format(time.RFC3339),
		})
	})

	v1.GET("/setup/status", authHandler.SetupStatus)
	v1.POST("/setup/admin", authHandler.SetupAdmin, setupLimiter)
	v1.POST("/auth/login", authHandler.Login, loginLimiter)
	v1.POST("/auth/logout", authHandler.Logout)

	// Protected routes — require authentication
	api := v1.Group("", authMW)

	// Current user
	api.GET("/auth/me", authHandler.Me)

	// Events stream
	api.GET("/events/stream", eventHandler.StreamEvents)

	// Sites
	api.GET("/sites", h.GetSites)
	api.GET("/sites/:id", h.GetSite)
	api.POST("/sites", h.CreateSite, operatorMW)
	api.PUT("/sites/:id", h.UpdateSite, operatorMW)
	api.DELETE("/sites/:id", h.DeleteSite, operatorMW)

	// Vlans
	api.GET("/vlans", h.GetVlans)
	api.GET("/vlans/:id", h.GetVlan)
	api.POST("/vlans", h.CreateVlan, operatorMW)
	api.PUT("/vlans/:id", h.UpdateVlan, operatorMW)
	api.DELETE("/vlans/:id", h.DeleteVlan, operatorMW)

	// Prefixes
	api.GET("/prefixes", h.GetPrefixes)
	api.GET("/prefixes/:id", h.GetPrefix)
	api.GET("/prefixes/:id/next-ip", h.GetNextAvailableIP)
	api.GET("/prefixes/:id/next-available", h.GetNextAvailableIP) // alias matching the planned spec
	api.GET("/prefixes/:id/addresses", h.GetPrefixAddresses)
	api.GET("/prefixes/:id/utilization", h.GetPrefixUtilization)
	api.POST("/prefixes", h.CreatePrefix, operatorMW)
	api.PUT("/prefixes/:id", h.UpdatePrefix, operatorMW)
	api.PUT("/prefixes/:id/scan-config", h.UpdatePrefixScanConfig, operatorMW)
	api.DELETE("/prefixes/:id", h.DeletePrefix, operatorMW)
	api.POST("/prefixes/:id/scan", h.StartPrefixScan, operatorMW)

	// IP Addresses
	api.GET("/ip-addresses", h.GetIPAddresses)
	api.GET("/ip-addresses/:id", h.GetIPAddress)
	api.POST("/ip-addresses", h.CreateIPAddress, operatorMW)
	api.PUT("/ip-addresses/:id", h.UpdateIPAddress, operatorMW)
	api.POST("/ip-addresses/:id/reserve", h.ReserveIPAddress, operatorMW)
	api.POST("/ip-addresses/:id/assign", h.AssignIPAddress, operatorMW)
	api.POST("/ip-addresses/:id/release", h.ReleaseIPAddress, operatorMW)
	api.DELETE("/ip-addresses/:id", h.DeleteIPAddress, operatorMW)

	// Devices
	api.GET("/devices", h.GetDevices)
	api.GET("/devices/:id", h.GetDevice)
	api.POST("/devices", h.CreateDevice, operatorMW)
	api.PUT("/devices/:id", h.UpdateDevice, operatorMW)
	api.DELETE("/devices/:id", h.DeleteDevice, operatorMW)

	// Discovery
	api.GET("/discovery/scans", h.GetScans)
	api.POST("/discovery/scans", h.StartScan, operatorMW)
	api.GET("/discovery/scans/:id", h.GetScan)
	api.GET("/discovery/results", h.GetDiscoveryResults)
	api.POST("/discovery/results/:id/accept", h.AcceptDiscoveryResult, operatorMW)
	api.POST("/discovery/results/:id/ignore", h.IgnoreDiscoveryResult, operatorMW)

	// Monitors
	api.GET("/monitors", monitorHandler.GetMonitors)
	api.GET("/monitors/:id", monitorHandler.GetMonitor)
	api.POST("/monitors", monitorHandler.CreateMonitor, operatorMW)
	api.PATCH("/monitors/:id", monitorHandler.UpdateMonitor, operatorMW)
	api.DELETE("/monitors/:id", monitorHandler.DeleteMonitor, operatorMW)
	api.POST("/monitors/:id/pause", monitorHandler.PauseMonitor, operatorMW)
	api.POST("/monitors/:id/resume", monitorHandler.ResumeMonitor, operatorMW)
	api.POST("/monitors/:id/test", monitorHandler.TestMonitor, operatorMW, monitorTestLimiter)
	api.GET("/monitors/:id/results", monitorHandler.GetMonitorResults)
	api.POST("/monitors/:id/rotate-push-token", monitorHandler.RotatePushToken, operatorMW)

	// Incidents
	api.GET("/incidents", incidentHandler.ListIncidents)
	api.GET("/incidents/counts", incidentHandler.IncidentCounts)
	api.GET("/incidents/:id", incidentHandler.GetIncident)
	api.POST("/incidents/:id/acknowledge", incidentHandler.AcknowledgeIncident, operatorMW)
	api.POST("/incidents/:id/resolve", incidentHandler.ResolveIncident, operatorMW)

	// Notification channels
	api.GET("/notification-channels", notificationHandler.ListChannels)
	api.POST("/notification-channels", notificationHandler.CreateChannel, adminMW)
	api.GET("/notification-channels/:id", notificationHandler.GetChannel)
	api.PATCH("/notification-channels/:id", notificationHandler.UpdateChannel, adminMW)
	api.DELETE("/notification-channels/:id", notificationHandler.DeleteChannel, adminMW)
	api.POST("/notification-channels/:id/test", notificationHandler.TestChannel, adminMW, notifTestLimiter)

	// Status pages
	api.GET("/status-pages", statusPageHandler.ListStatusPages)
	api.POST("/status-pages", statusPageHandler.CreateStatusPage, operatorMW)
	api.GET("/status-pages/:id", statusPageHandler.GetStatusPage)
	api.PATCH("/status-pages/:id", statusPageHandler.UpdateStatusPage, operatorMW)
	api.DELETE("/status-pages/:id", statusPageHandler.DeleteStatusPage, operatorMW)
	api.POST("/status-pages/:id/monitors", statusPageHandler.AttachMonitor, operatorMW)
	api.PATCH("/status-pages/:id/monitors/:monitor_id", statusPageHandler.UpdateAttachedMonitor, operatorMW)
	api.DELETE("/status-pages/:id/monitors/:monitor_id", statusPageHandler.RemoveAttachedMonitor, operatorMW)
	api.POST("/status-pages/:id/monitors/reorder", statusPageHandler.ReorderMonitors, operatorMW)

	// Users
	api.GET("/users", authHandler.ListUsers, adminMW)
	api.POST("/users", authHandler.CreateUser, adminMW)
	api.GET("/users/:id", authHandler.GetUser, adminMW)
	api.PATCH("/users/:id", authHandler.UpdateUser, adminMW)
	api.DELETE("/users/:id", authHandler.DeleteUser, adminMW)

	// API tokens
	api.GET("/api-tokens", authHandler.ListAPITokens, adminMW)
	api.POST("/api-tokens", authHandler.CreateAPIToken, adminMW, apiTokenLimiter)
	api.DELETE("/api-tokens/:id", authHandler.DeleteAPIToken, adminMW)

	// Audit log
	api.GET("/audit-log", authHandler.ListAuditLog, adminMW)

	// Admin actions
	admin := api.Group("/admin", adminMW)
	admin.POST("/retention/run", func(c echo.Context) error {
		retentionSvc.Run(c.Request().Context(), retentionCfg)
		return c.JSON(http.StatusOK, map[string]string{"status": "scheduled"})
	})

	// ---- Server start ----
	go func() {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		if err := e.Start(":" + port); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal("shutting down the server")
		}
	}()

	// 8. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Blackgrid...")
	
	monitorScheduler.Stop()
	cancelScheduler()
	cancelRetention()

	select {
	case <-schedDone:
		log.Println("Discovery scheduler stopped")
	case <-time.After(5 * time.Second):
		log.Println("Discovery scheduler did not stop within 5s")
	}

	shutdownTimeout := time.Duration(getEnvInt("SHUTDOWN_TIMEOUT_SECONDS", 20)) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}

	// Close SSE subscribers cleanly so any blocked Publish callers return
	// and clients see EOF instead of a hung connection.
	bus.Shutdown()
	
	log.Println("Blackgrid exited cleanly")
}

func getEnvBool(key string, def bool) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "":
		return def
	case "1", "t", "true", "yes", "on":
		return true
	case "0", "f", "false", "no", "off":
		return false
	default:
		return def
	}
}

func getEnvInt(key string, def int) int {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return def
	}
	return n
}
