package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"blackgrid/internal/api/handlers"
	"blackgrid/internal/db"
	"blackgrid/internal/monitor"
	"blackgrid/internal/service"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:3000"},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAuthorization},
		AllowCredentials: true, // required for session cookies cross-origin
	}))

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://blackgrid:blackgrid@localhost:5432/blackgrid?sslmode=disable"
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer pool.Close()

	queries := db.New(pool)

	// ---- Services ----
	auditSvc := service.NewAuditService(queries)
	authCfg := service.AuthConfigFromEnv()
	authSvc := service.NewAuthService(queries, authCfg, auditSvc)

	siteSvc := service.NewSiteService(queries)
	vlanSvc := service.NewVlanService(queries)
	prefixSvc := service.NewPrefixService(queries)
	ipSvc := service.NewIPAddressService(queries)
	deviceSvc := service.NewDeviceService(queries)
	discoverySvc := service.NewDiscoveryService(queries)

	// Start background scheduled scans with a cancelable context for graceful shutdown.
	schedCtx, cancelScheduler := context.WithCancel(context.Background())
	schedDone := make(chan struct{})
	go func() {
		defer close(schedDone)
		discoverySvc.RunScheduledScans(schedCtx)
	}()

	h := handlers.New(siteSvc, vlanSvc, prefixSvc, ipSvc, deviceSvc, discoverySvc)

	incidentSvc := service.NewIncidentService(queries)
	notificationSvc := service.NewNotificationService(queries)
	incidentSvc.SetNotifier(notificationSvc)

	monitorRunner := monitor.NewRunner(queries)
	monitorScheduler := monitor.NewScheduler(queries, monitorRunner, 10)
	monitorScheduler.SetIncidentHook(service.NewIncidentHook(incidentSvc))
	monitorScheduler.Start()

	monitorHandler := handlers.NewMonitorHandler(queries, monitorRunner)
	incidentHandler := handlers.NewIncidentHandler(incidentSvc)
	notificationHandler := handlers.NewNotificationHandler(notificationSvc)

	statusPageSvc := service.NewStatusPageService(queries)
	statusPageHandler := handlers.NewStatusPageHandler(statusPageSvc)

	authHandler := handlers.NewAuthHandler(authSvc, auditSvc, authCfg)

	// ---- Auth middleware ----
	authMW := handlers.AuthMiddleware(authSvc)
	adminMW := handlers.RequireAdmin()
	operatorMW := handlers.RequireOperator()

	// ---- Public status endpoint (no auth required) ----
	e.GET("/status/:slug", statusPageHandler.PublicStatusPage)

	// ---- API v1 group ----
	v1 := e.Group("/api/v1")

	// Public routes (no auth required)
	v1.GET("/health", h.Health)
	v1.GET("/setup/status", authHandler.SetupStatus)
	v1.POST("/setup/admin", authHandler.SetupAdmin)
	v1.POST("/auth/login", authHandler.Login)
	v1.POST("/auth/logout", authHandler.Logout)

	// Protected routes — require authentication
	api := v1.Group("", authMW)

	// Current user
	api.GET("/auth/me", authHandler.Me)

	// Sites — viewer can GET, operator+ can mutate
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
	api.POST("/monitors/:id/test", monitorHandler.TestMonitor, operatorMW)
	api.GET("/monitors/:id/results", monitorHandler.GetMonitorResults)

	// Incidents
	api.GET("/incidents", incidentHandler.ListIncidents)
	api.GET("/incidents/counts", incidentHandler.IncidentCounts)
	api.GET("/incidents/:id", incidentHandler.GetIncident)
	api.POST("/incidents/:id/acknowledge", incidentHandler.AcknowledgeIncident, operatorMW)
	api.POST("/incidents/:id/resolve", incidentHandler.ResolveIncident, operatorMW)

	// Notification channels — admin only for mutation
	api.GET("/notification-channels", notificationHandler.ListChannels)
	api.POST("/notification-channels", notificationHandler.CreateChannel, adminMW)
	api.GET("/notification-channels/:id", notificationHandler.GetChannel)
	api.PATCH("/notification-channels/:id", notificationHandler.UpdateChannel, adminMW)
	api.DELETE("/notification-channels/:id", notificationHandler.DeleteChannel, adminMW)
	api.POST("/notification-channels/:id/test", notificationHandler.TestChannel, adminMW)

	// Status pages (admin)
	api.GET("/status-pages", statusPageHandler.ListStatusPages)
	api.POST("/status-pages", statusPageHandler.CreateStatusPage, operatorMW)
	api.GET("/status-pages/:id", statusPageHandler.GetStatusPage)
	api.PATCH("/status-pages/:id", statusPageHandler.UpdateStatusPage, operatorMW)
	api.DELETE("/status-pages/:id", statusPageHandler.DeleteStatusPage, operatorMW)
	api.POST("/status-pages/:id/monitors", statusPageHandler.AttachMonitor, operatorMW)
	api.PATCH("/status-pages/:id/monitors/:monitor_id", statusPageHandler.UpdateAttachedMonitor, operatorMW)
	api.DELETE("/status-pages/:id/monitors/:monitor_id", statusPageHandler.RemoveAttachedMonitor, operatorMW)
	api.POST("/status-pages/:id/monitors/reorder", statusPageHandler.ReorderMonitors, operatorMW)

	// Users — admin only
	api.GET("/users", authHandler.ListUsers, adminMW)
	api.POST("/users", authHandler.CreateUser, adminMW)
	api.GET("/users/:id", authHandler.GetUser, adminMW)
	api.PATCH("/users/:id", authHandler.UpdateUser, adminMW)
	api.DELETE("/users/:id", authHandler.DeleteUser, adminMW)

	// API tokens — admin only
	api.GET("/api-tokens", authHandler.ListAPITokens, adminMW)
	api.POST("/api-tokens", authHandler.CreateAPIToken, adminMW)
	api.DELETE("/api-tokens/:id", authHandler.DeleteAPIToken, adminMW)

	// Audit log — admin only
	api.GET("/audit-log", authHandler.ListAuditLog, adminMW)

	// ---- Server start ----
	go func() {
		if err := e.Start(":8080"); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal("shutting down the server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	monitorScheduler.Stop()

	cancelScheduler()
	select {
	case <-schedDone:
	case <-time.After(5 * time.Second):
		log.Println("discovery scheduler did not stop within 5s")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
}
