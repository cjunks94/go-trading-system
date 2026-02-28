package trading

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/cjunker/go-trading-system/internal/market"
	"github.com/cjunker/go-trading-system/internal/orderbook"
	"github.com/cjunker/go-trading-system/pkg/models"
	"github.com/google/uuid"
)

var (
	ErrSymbolNotFound = errors.New("symbol not found")
	ErrOrderRejected  = errors.New("order rejected")
)

// Engine orchestrates the trading system
type Engine struct {
	feed        market.Feed
	orderBooks  map[string]*orderbook.OrderBook
	positions   *PositionManager
	risk        *RiskManager
	orders      map[string]*models.Order
	tradesChan  chan models.Trade // Published trades
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// NewEngine creates a new trading engine
func NewEngine(feed market.Feed, limits models.RiskLimits) *Engine {
	orderBooks := make(map[string]*orderbook.OrderBook)
	for _, symbol := range feed.Symbols() {
		orderBooks[symbol] = orderbook.NewOrderBook(symbol)
	}

	return &Engine{
		feed:       feed,
		orderBooks: orderBooks,
		positions:  NewPositionManager(),
		risk:       NewRiskManager(limits),
		orders:     make(map[string]*models.Order),
		tradesChan: make(chan models.Trade, 1000),
	}
}

// Start begins the trading engine
func (e *Engine) Start() error {
	e.ctx, e.cancel = context.WithCancel(context.Background())

	// Start market data processing
	e.wg.Add(1)
	go e.processMarketData()

	// Start P&L updater
	e.wg.Add(1)
	go e.updatePnL()

	log.Println("[Engine] Trading engine started")
	return nil
}

// Stop gracefully shuts down the engine
func (e *Engine) Stop() error {
	if e.cancel != nil {
		e.cancel()
	}
	e.wg.Wait()
	close(e.tradesChan)
	log.Println("[Engine] Trading engine stopped")
	return nil
}

// SubmitOrder validates and processes a new order
func (e *Engine) SubmitOrder(req *models.OrderRequest) (*models.Order, []models.Trade, error) {
	// Validate order request
	if err := req.Validate(); err != nil {
		return nil, nil, err
	}

	// Get order book for symbol
	e.mu.RLock()
	book, exists := e.orderBooks[req.Symbol]
	e.mu.RUnlock()

	if !exists {
		return nil, nil, ErrSymbolNotFound
	}

	// Check risk limits
	position := e.positions.GetPosition(req.Symbol)
	order := &models.Order{
		ID:        uuid.New().String(),
		Symbol:    req.Symbol,
		Side:      req.Side,
		Type:      req.Type,
		Quantity:  req.Quantity,
		Price:     req.Price,
		Status:    models.Pending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := e.risk.CheckOrder(order, position); err != nil {
		order.Status = models.Rejected
		return order, nil, err
	}

	// For market orders, use current market price if not specified
	if order.Type == models.Market {
		quote, ok := e.feed.GetQuote(req.Symbol)
		if ok {
			if order.Side == models.Buy {
				order.Price = quote.Ask
			} else {
				order.Price = quote.Bid
			}
		}
	}

	// Store order
	e.mu.Lock()
	e.orders[order.ID] = order
	e.mu.Unlock()

	// Match order
	trades := book.Match(order)

	// Process trades
	for _, trade := range trades {
		e.positions.UpdateFromTrade(trade)

		// Publish trade
		select {
		case e.tradesChan <- trade:
		default:
			log.Println("[Engine] Trade channel full, dropping trade")
		}
	}

	// Update order status if not fully filled
	if !order.IsFilled() && order.Type == models.Market {
		order.Status = models.Partial
	}

	log.Printf("[Engine] Order %s: %s %s %d @ %.2f - Status: %s, Filled: %d",
		order.ID[:8], order.Side, order.Symbol, order.Quantity, order.Price, order.Status, order.FilledQty)

	return order, trades, nil
}

// CancelOrder cancels a pending order
func (e *Engine) CancelOrder(orderID string) error {
	e.mu.Lock()
	order, exists := e.orders[orderID]
	if !exists {
		e.mu.Unlock()
		return errors.New("order not found")
	}

	book := e.orderBooks[order.Symbol]
	e.mu.Unlock()

	if book.CancelOrder(orderID) {
		order.Status = models.Cancelled
		order.UpdatedAt = time.Now()
		return nil
	}

	return errors.New("failed to cancel order")
}

// GetOrder returns an order by ID
func (e *Engine) GetOrder(orderID string) (*models.Order, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	order, exists := e.orders[orderID]
	if !exists {
		return nil, errors.New("order not found")
	}
	return order, nil
}

// GetPositions returns all positions
func (e *Engine) GetPositions() []models.Position {
	return e.positions.GetAllPositions()
}

// GetPosition returns position for a symbol
func (e *Engine) GetPosition(symbol string) *models.Position {
	return e.positions.GetPosition(symbol)
}

// GetPnL returns P&L summary
func (e *Engine) GetPnL() models.PnLSummary {
	return e.positions.GetPnLSummary()
}

// Trades returns the channel for executed trades
func (e *Engine) Trades() <-chan models.Trade {
	return e.tradesChan
}

// GetRiskStatus returns current risk status
func (e *Engine) GetRiskStatus() map[string]interface{} {
	return map[string]interface{}{
		"breached": e.risk.IsBreached(),
		"dailyPnL": e.risk.GetDailyPnL(),
		"limits":   e.risk.GetLimits(),
	}
}

// processMarketData updates positions with current prices
func (e *Engine) processMarketData() {
	defer e.wg.Done()

	quotes := e.feed.Subscribe()

	for {
		select {
		case <-e.ctx.Done():
			return
		case quote, ok := <-quotes:
			if !ok {
				return
			}
			// Update position market value
			e.positions.UpdateMarketValue(quote.Symbol, quote.Last)
		}
	}
}

// updatePnL periodically updates risk manager with current P&L
func (e *Engine) updatePnL() {
	defer e.wg.Done()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			summary := e.positions.GetPnLSummary()
			e.risk.UpdateDailyPnL(summary.TotalPL)
		}
	}
}
