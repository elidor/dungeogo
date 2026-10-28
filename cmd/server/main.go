package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/elidor/dungeogo/config"
	"github.com/elidor/dungeogo/pkg/game"
	"github.com/elidor/dungeogo/pkg/persistence/postgres"
	"github.com/elidor/dungeogo/pkg/server"
)

func main() {
	cfg := config.NewConfig(config.NewFileProvider(".env"))

	// Get configuration values
	port := cfg.GetValue(config.Port)
	if port == "" {
		port = "8080"
	}

	bindAddress := cfg.GetValue(config.BindAddress)
	if bindAddress == "" {
		bindAddress = "localhost"
	}

	databaseURL := cfg.GetValue(config.DatabaseURL)
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	address := fmt.Sprintf("%s:%s", bindAddress, port)

	// Initialize database connection
	log.Println("Connecting to database...")
	repoManager, err := postgres.NewPostgreSQLRepositoryManager(databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	startingRoom := cfg.GetValue(config.StartingRoom)
	if startingRoom == "" {
		startingRoom = postgres.DefaultStartingRoom
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := repoManager.Bootstrap(ctx, startingRoom); err != nil {
		cancel()
		repoManager.Close()
		log.Fatalf("Failed to initialize world: %v", err)
	}
	cancel()

	// Initialize game engine
	log.Println("Starting game engine...")
	gameEngine := game.NewEngine(repoManager)

	// Initialize session handler
	sessionHandler := server.NewSessionHandler(repoManager, gameEngine)
	sessionHandler.SetStartingRoom(startingRoom)

	// Initialize connection manager
	connectionManager := server.NewConnectionManager(100, 30*time.Minute)
	connectionManager.SetHandler(sessionHandler)

	os.Exit(serve(connectionManager, address, repoManager))
}

// serve runs the server until it fails or receives SIGINT/SIGTERM, then shuts
// down in order: stop accepting, close sessions and wait for their in-flight
// commands, then close the database. It returns the process exit code; any
// failure along the way is logged and makes it nonzero.
func serve(connections *server.ConnectionManager, address string, repositories interface{ Close() error }) int {
	// Catch signals before serving, so one that arrives during startup still
	// shuts down in order instead of killing the process.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)

	log.Printf("Starting DungeoGo server on %s", address)
	served := make(chan error, 1)
	go func() { served <- connections.Start(address) }()

	code := 0
	select {
	case err := <-served:
		log.Printf("Server stopped: %v", err)
		code = 1
		if err := connections.Stop(); err != nil {
			log.Printf("Shutdown failed: %v", err)
		}
	case sig := <-signals:
		log.Printf("Received %s; shutting down...", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		if err := connections.Shutdown(ctx); err != nil {
			log.Printf("Shutdown failed: %v", err)
			code = 1
		}
		cancel()
		if err := <-served; err != nil {
			log.Printf("Server stopped with error: %v", err)
			code = 1
		}
	}
	if err := repositories.Close(); err != nil {
		log.Printf("Closing database failed: %v", err)
		code = 1
	}
	if code == 0 {
		log.Println("Shutdown complete.")
	}
	return code
}
