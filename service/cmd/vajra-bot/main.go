package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yourorg/vajra-bot/service/internal/api"
	"github.com/yourorg/vajra-bot/service/internal/config"
	"github.com/yourorg/vajra-bot/service/internal/db"

	log "github.com/sirupsen/logrus"
)

func main() {
	// Parse flags
	var standalone bool
	var serverURL string

	flag.BoolVar(&standalone, "standalone", false, "Run in standalone mode with private server")
	flag.StringVar(&serverURL, "server", "", "Connect to specific server URL")
	flag.Parse()

	// Initialize config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	dbPath := cfg.Service.DataDir
	if err := db.Init(dbPath); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize API
	apiRouter := api.NewRouter(cfg, db, standalone)

	// Determine server address
	addr := fmt.Sprintf(":%d", cfg.Service.Port)
	if serverURL != "" {
		addr = serverURL
	}

	// Start HTTP server
	server := &http.Server{
		Addr:         addr,
		Handler:      apiRouter,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Graceful shutdown
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		log.Infof("Starting vajra Bot service on %s", addr)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-ch
	log.Println("Shutting down server...")
}

