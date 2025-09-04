package main

import (
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
	defer repoManager.Close()
	
	// Initialize game engine
	log.Println("Starting game engine...")
	gameEngine := game.NewEngine(repoManager)
	
	// Initialize session handler
	sessionHandler := server.NewSessionHandler(repoManager, gameEngine)
	
	// Initialize connection manager
	connectionManager := server.NewConnectionManager(100, 30*time.Minute)
	connectionManager.SetHandler(sessionHandler)
	
	// Start server
	log.Printf("Starting DungeoGo server on %s", address)
	
	// Handle graceful shutdown
	go func() {
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
		<-sigchan
		
		log.Println("Shutting down server...")
		connectionManager.Stop()
		os.Exit(0)
	}()
	
	// Start accepting connections
	if err := connectionManager.Start(address); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
