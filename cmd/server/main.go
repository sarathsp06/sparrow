package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/sarathsp06/sparrow"
	"github.com/sarathsp06/sparrow/internal/config"
	"github.com/sarathsp06/sparrow/server"
)

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	migrateOnly := flag.Bool("migrate", false, "run database migrations and exit (for a pre-deploy job)")
	flag.Parse()
	if *showVersion {
		fmt.Println("sparrow-server", sparrow.Version)
		return
	}

	// Load .env file if present, but only outside production.
	// In production containers a .env file should not exist, but if one
	// is accidentally present it could silently override critical env vars
	// (DATABASE_URL, SPARROW_API_KEY, SPARROW_ENCRYPTION_KEYS).
	// We check SPARROW_ENVIRONMENT from the real OS env first (before godotenv)
	// to avoid the chicken-and-egg problem.
	if !config.IsProductionEnv(os.Getenv("SPARROW_ENVIRONMENT")) {
		_ = godotenv.Load()
	}

	ctx := context.Background()

	if *migrateOnly {
		if err := server.Migrate(ctx); err != nil {
			log.Fatalf("%v", err)
		}
		return
	}

	srv, err := server.New(ctx)
	if err != nil {
		log.Fatalf("%v", err)
	}
	if err := srv.Start(ctx); err != nil {
		log.Fatalf("%v", err)
	}

	httpServer := &http.Server{
		Addr:              srv.Addr(),
		Handler:           srv.Handler(),
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 10 * time.Second, // Mitigate slowloris attacks
		MaxHeaderBytes:    1 << 20,          // 1 MB max header size
	}

	// Start HTTP server in a goroutine
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to serve HTTP: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	fmt.Println("   Press Ctrl+C to stop...")
	<-sigChan

	fmt.Printf("\n🛑 Shutting down (up to %s)...\n", srv.ShutdownTimeout())

	// Graceful shutdown: one budget for everything. Readiness goes first so
	// load balancers stop sending traffic while in-flight work finishes.
	srv.Drain()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), srv.ShutdownTimeout())
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Shutdown error: %v", err)
	}
	fmt.Println("👋 Shutdown complete")
}
