package main

import (
	"context"
	"log"
	"os"

	"blackgrid/internal/api/handlers"
	"blackgrid/internal/db"
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

	h := handlers.New(siteSvc, vlanSvc, prefixSvc, ipSvc, deviceSvc)

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

	e.Logger.Fatal(e.Start(":8080"))
}
