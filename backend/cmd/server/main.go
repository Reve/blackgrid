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
	e.Use(middleware.CORS())

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

	v1 := e.Group("/api/v1")

	v1.GET("/health", h.Health)

	// Sites
	v1.GET("/sites", h.GetSites)
	v1.GET("/sites/:id", h.GetSite)
	v1.POST("/sites", h.CreateSite)
	v1.PUT("/sites/:id", h.UpdateSite)
	v1.DELETE("/sites/:id", h.DeleteSite)

	// Vlans
	v1.GET("/vlans", h.GetVlans)
	v1.GET("/vlans/:id", h.GetVlan)
	v1.POST("/vlans", h.CreateVlan)
	v1.PUT("/vlans/:id", h.UpdateVlan)
	v1.DELETE("/vlans/:id", h.DeleteVlan)

	// Prefixes
	v1.GET("/prefixes", h.GetPrefixes)
	v1.GET("/prefixes/:id", h.GetPrefix)
	v1.GET("/prefixes/:id/next-ip", h.GetNextAvailableIP)
	v1.POST("/prefixes", h.CreatePrefix)
	v1.PUT("/prefixes/:id", h.UpdatePrefix)
	v1.PUT("/prefixes/:id/scan-config", h.UpdatePrefixScanConfig)
	v1.DELETE("/prefixes/:id", h.DeletePrefix)

	// IP Addresses
	v1.GET("/ip-addresses", h.GetIPAddresses)
	v1.GET("/ip-addresses/:id", h.GetIPAddress)
	v1.POST("/ip-addresses", h.CreateIPAddress)
	v1.PUT("/ip-addresses/:id", h.UpdateIPAddress)
	v1.DELETE("/ip-addresses/:id", h.DeleteIPAddress)

	// Devices
	v1.GET("/devices", h.GetDevices)
	v1.GET("/devices/:id", h.GetDevice)
	v1.POST("/devices", h.CreateDevice)
	v1.PUT("/devices/:id", h.UpdateDevice)
	v1.DELETE("/devices/:id", h.DeleteDevice)

	// Discovery
	v1.GET("/discovery/scans", h.GetScans)
	v1.POST("/discovery/scans", h.StartScan)
	v1.GET("/discovery/scans/:id", h.GetScan)
	v1.GET("/discovery/results", h.GetDiscoveryResults)
	v1.POST("/discovery/results/:id/accept", h.AcceptDiscoveryResult)
	v1.POST("/discovery/results/:id/ignore", h.IgnoreDiscoveryResult)
	v1.POST("/prefixes/:id/scan", h.StartPrefixScan)

	// Monitors
	v1.GET("/monitors", monitorHandler.GetMonitors)
	v1.GET("/monitors/:id", monitorHandler.GetMonitor)
	v1.POST("/monitors", monitorHandler.CreateMonitor)
	v1.PATCH("/monitors/:id", monitorHandler.UpdateMonitor)
	v1.DELETE("/monitors/:id", monitorHandler.DeleteMonitor)
	v1.POST("/monitors/:id/pause", monitorHandler.PauseMonitor)
	v1.POST("/monitors/:id/resume", monitorHandler.ResumeMonitor)
	v1.POST("/monitors/:id/test", monitorHandler.TestMonitor)
	v1.GET("/monitors/:id/results", monitorHandler.GetMonitorResults)

	// Incidents
	v1.GET("/incidents", incidentHandler.ListIncidents)
	v1.GET("/incidents/counts", incidentHandler.IncidentCounts)
	v1.GET("/incidents/:id", incidentHandler.GetIncident)
	v1.POST("/incidents/:id/acknowledge", incidentHandler.AcknowledgeIncident)
	v1.POST("/incidents/:id/resolve", incidentHandler.ResolveIncident)

	// Notification channels
	v1.GET("/notification-channels", notificationHandler.ListChannels)
	v1.POST("/notification-channels", notificationHandler.CreateChannel)
	v1.GET("/notification-channels/:id", notificationHandler.GetChannel)
	v1.PATCH("/notification-channels/:id", notificationHandler.UpdateChannel)
	v1.DELETE("/notification-channels/:id", notificationHandler.DeleteChannel)
	v1.POST("/notification-channels/:id/test", notificationHandler.TestChannel)

	// Status pages (admin)
	v1.GET("/status-pages", statusPageHandler.ListStatusPages)
	v1.POST("/status-pages", statusPageHandler.CreateStatusPage)
	v1.GET("/status-pages/:id", statusPageHandler.GetStatusPage)
	v1.PATCH("/status-pages/:id", statusPageHandler.UpdateStatusPage)
	v1.DELETE("/status-pages/:id", statusPageHandler.DeleteStatusPage)
	v1.POST("/status-pages/:id/monitors", statusPageHandler.AttachMonitor)
	v1.PATCH("/status-pages/:id/monitors/:monitor_id", statusPageHandler.UpdateAttachedMonitor)
	v1.DELETE("/status-pages/:id/monitors/:monitor_id", statusPageHandler.RemoveAttachedMonitor)
	v1.POST("/status-pages/:id/monitors/reorder", statusPageHandler.ReorderMonitors)

	// Public status endpoint (outside /api/v1)
	e.GET("/status/:slug", statusPageHandler.PublicStatusPage)

	// Graceful shutdown
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
