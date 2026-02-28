package api

import (
	"log"
	"net/http"
	"time"

	"github.com/cjunker/go-trading-system/internal/market"
	"github.com/cjunker/go-trading-system/internal/trading"
	"github.com/cjunker/go-trading-system/internal/websocket"
	"github.com/gorilla/mux"
)

// Server represents the HTTP server
type Server struct {
	router   *mux.Router
	handlers *Handlers
	hub      *websocket.Hub
	engine   *trading.Engine
	feed     market.Feed
}

// NewServer creates a new API server
func NewServer(engine *trading.Engine, feed market.Feed, hub *websocket.Hub) *Server {
	s := &Server{
		router:   mux.NewRouter(),
		handlers: NewHandlers(engine, feed, hub),
		hub:      hub,
		engine:   engine,
		feed:     feed,
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// Apply middleware
	s.router.Use(loggingMiddleware)
	s.router.Use(corsMiddleware)

	// Health check
	s.router.HandleFunc("/health", s.handlers.HealthCheck).Methods("GET")

	// Market data
	s.router.HandleFunc("/api/quotes", s.handlers.GetQuotes).Methods("GET")
	s.router.HandleFunc("/api/quotes/{symbol}", s.handlers.GetQuote).Methods("GET")

	// Orders
	s.router.HandleFunc("/api/orders", s.handlers.SubmitOrder).Methods("POST")
	s.router.HandleFunc("/api/orders/{id}", s.handlers.GetOrder).Methods("GET")
	s.router.HandleFunc("/api/orders/{id}", s.handlers.CancelOrder).Methods("DELETE")

	// Positions
	s.router.HandleFunc("/api/positions", s.handlers.GetPositions).Methods("GET")
	s.router.HandleFunc("/api/positions/{symbol}", s.handlers.GetPosition).Methods("GET")

	// P&L
	s.router.HandleFunc("/api/pnl", s.handlers.GetPnL).Methods("GET")

	// WebSocket
	s.router.HandleFunc("/ws", s.handlers.WebSocket)

	// Serve static files
	s.router.PathPrefix("/").Handler(http.FileServer(http.Dir("./web")))
}

// Router returns the router for the server
func (s *Server) Router() *mux.Router {
	return s.router
}

// loggingMiddleware logs HTTP requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[HTTP] %s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// corsMiddleware adds CORS headers
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
