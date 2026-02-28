package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cjunker/go-trading-system/internal/api"
	"github.com/cjunker/go-trading-system/internal/config"
	"github.com/cjunker/go-trading-system/internal/market"
	"github.com/cjunker/go-trading-system/internal/trading"
	"github.com/cjunker/go-trading-system/internal/websocket"
	"github.com/cjunker/go-trading-system/pkg/models"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting Go Trading System...")

	// Load configuration
	cfg := config.Load()
	log.Printf("Config: Port=%s, Symbols=%v, Workers=%d", cfg.Port, cfg.Symbols, cfg.Workers)

	// Initialize components
	feed := market.NewSimulatedFeed(cfg.Symbols)
	hub := websocket.NewHub()

	limits := models.RiskLimits{
		MaxPositionSize: cfg.MaxPositionSize,
		MaxOrderSize:    cfg.MaxOrderSize,
		MaxDailyLoss:    cfg.MaxDailyLoss,
	}
	engine := trading.NewEngine(feed, limits)

	// Start components
	if err := feed.Start(); err != nil {
		log.Fatalf("Failed to start feed: %v", err)
	}

	if err := engine.Start(); err != nil {
		log.Fatalf("Failed to start engine: %v", err)
	}

	go hub.Run()

	// Start quote broadcaster (sends quotes to WebSocket clients)
	go broadcastQuotes(feed, hub)

	// Setup HTTP server
	server := api.NewServer(engine, feed, hub)
	httpServer := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      server.Router(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("HTTP server listening on :%s", cfg.Port)
		log.Printf("Dashboard: http://localhost:%s", cfg.Port)
		log.Printf("WebSocket: ws://localhost:%s/ws", cfg.Port)
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	if err := engine.Stop(); err != nil {
		log.Printf("Engine stop error: %v", err)
	}
	hub.Stop()
	if err := feed.Stop(); err != nil {
		log.Printf("Feed stop error: %v", err)
	}

	log.Println("Server stopped")
}

// broadcastQuotes sends market data to all WebSocket clients
func broadcastQuotes(feed market.Feed, hub *websocket.Hub) {
	quotes := feed.Subscribe()

	for quote := range quotes {
		if err := hub.Broadcast(websocket.NewQuoteMessage(quote)); err != nil {
			// Best-effort broadcast, continue on error
			log.Printf("Broadcast error: %v", err)
		}
	}
}
