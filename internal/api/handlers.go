package api

import (
	"encoding/json"
	"net/http"

	"github.com/cjunker/go-trading-system/internal/market"
	"github.com/cjunker/go-trading-system/internal/trading"
	"github.com/cjunker/go-trading-system/internal/websocket"
	"github.com/cjunker/go-trading-system/pkg/models"
	"github.com/gorilla/mux"
)

// Handlers contains HTTP handler functions
type Handlers struct {
	engine *trading.Engine
	feed   market.Feed
	hub    *websocket.Hub
}

// NewHandlers creates new API handlers
func NewHandlers(engine *trading.Engine, feed market.Feed, hub *websocket.Hub) *Handlers {
	return &Handlers{
		engine: engine,
		feed:   feed,
		hub:    hub,
	}
}

// HealthCheck returns service health status
func (h *Handlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status":      "healthy",
		"wsClients":   h.hub.ClientCount(),
		"symbols":     h.feed.Symbols(),
		"riskStatus":  h.engine.GetRiskStatus(),
	}
	writeJSON(w, http.StatusOK, status)
}

// GetQuotes returns current quotes for all symbols
func (h *Handlers) GetQuotes(w http.ResponseWriter, r *http.Request) {
	quotes := make([]models.Quote, 0)
	for _, symbol := range h.feed.Symbols() {
		if quote, ok := h.feed.GetQuote(symbol); ok {
			quotes = append(quotes, quote)
		}
	}
	writeJSON(w, http.StatusOK, quotes)
}

// GetQuote returns quote for a specific symbol
func (h *Handlers) GetQuote(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	symbol := vars["symbol"]

	quote, ok := h.feed.GetQuote(symbol)
	if !ok {
		writeError(w, http.StatusNotFound, "symbol not found")
		return
	}
	writeJSON(w, http.StatusOK, quote)
}

// SubmitOrder handles new order submissions
func (h *Handlers) SubmitOrder(w http.ResponseWriter, r *http.Request) {
	var req models.OrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	order, trades, err := h.engine.SubmitOrder(&req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	response := map[string]interface{}{
		"order":  order,
		"trades": trades,
	}
	writeJSON(w, http.StatusCreated, response)

	// Broadcast order update (errors logged, not critical)
	if err := h.hub.Broadcast(websocket.NewOrderMessage(order)); err != nil {
		// WebSocket broadcast is best-effort, don't fail request
		_ = err
	}
	for _, trade := range trades {
		if err := h.hub.Broadcast(websocket.NewTradeMessage(trade)); err != nil {
			_ = err
		}
	}
}

// CancelOrder handles order cancellation
func (h *Handlers) CancelOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID := vars["id"]

	if err := h.engine.CancelOrder(orderID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	order, _ := h.engine.GetOrder(orderID)
	writeJSON(w, http.StatusOK, order)
}

// GetOrder returns a specific order
func (h *Handlers) GetOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID := vars["id"]

	order, err := h.engine.GetOrder(orderID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, order)
}

// GetPositions returns all positions
func (h *Handlers) GetPositions(w http.ResponseWriter, r *http.Request) {
	positions := h.engine.GetPositions()
	writeJSON(w, http.StatusOK, positions)
}

// GetPosition returns position for a symbol
func (h *Handlers) GetPosition(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	symbol := vars["symbol"]

	position := h.engine.GetPosition(symbol)
	if position == nil {
		writeJSON(w, http.StatusOK, models.Position{Symbol: symbol})
		return
	}
	writeJSON(w, http.StatusOK, position)
}

// GetPnL returns P&L summary
func (h *Handlers) GetPnL(w http.ResponseWriter, r *http.Request) {
	pnl := h.engine.GetPnL()
	writeJSON(w, http.StatusOK, pnl)
}

// WebSocket handles WebSocket upgrade
func (h *Handlers) WebSocket(w http.ResponseWriter, r *http.Request) {
	h.hub.ServeWs(w, r)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Client may have disconnected, log but don't fail
		http.Error(w, "encoding error", http.StatusInternalServerError)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
