package orderbook

import (
	"container/heap"
	"sync"
	"time"

	"github.com/cjunker/go-trading-system/pkg/models"
	"github.com/google/uuid"
)

// OrderBook manages orders for a single symbol with price-time priority
type OrderBook struct {
	symbol string
	bids   *orderHeap // Max-heap for bids (highest price first)
	asks   *orderHeap // Min-heap for asks (lowest price first)
	orders map[string]*models.Order
	mu     sync.RWMutex
}

// NewOrderBook creates a new order book for a symbol
func NewOrderBook(symbol string) *OrderBook {
	bids := &orderHeap{orders: make([]*models.Order, 0), isMaxHeap: true}
	asks := &orderHeap{orders: make([]*models.Order, 0), isMaxHeap: false}
	heap.Init(bids)
	heap.Init(asks)

	return &OrderBook{
		symbol: symbol,
		bids:   bids,
		asks:   asks,
		orders: make(map[string]*models.Order),
	}
}

// Symbol returns the order book's symbol
func (ob *OrderBook) Symbol() string {
	return ob.symbol
}

// AddOrder adds a limit order to the book
func (ob *OrderBook) AddOrder(order *models.Order) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	ob.orders[order.ID] = order

	if order.Side == models.Buy {
		heap.Push(ob.bids, order)
	} else {
		heap.Push(ob.asks, order)
	}
}

// CancelOrder removes an order from the book
func (ob *OrderBook) CancelOrder(orderID string) bool {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	order, exists := ob.orders[orderID]
	if !exists {
		return false
	}

	order.Status = models.Canceled
	order.UpdatedAt = time.Now()
	delete(ob.orders, orderID)
	return true
}

// Match attempts to match an incoming order against the book
// Returns executed trades
func (ob *OrderBook) Match(order *models.Order) []models.Trade {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	var trades []models.Trade
	var oppositeBook *orderHeap

	if order.Side == models.Buy {
		oppositeBook = ob.asks
	} else {
		oppositeBook = ob.bids
	}

	for order.RemainingQty() > 0 && oppositeBook.Len() > 0 {
		bestOrder := oppositeBook.Peek()

		// Skip cancelled orders
		if bestOrder.Status == models.Canceled {
			heap.Pop(oppositeBook)
			continue
		}

		// Check price compatibility
		if order.Type == models.Limit {
			if order.Side == models.Buy && order.Price < bestOrder.Price {
				break // No match possible
			}
			if order.Side == models.Sell && order.Price > bestOrder.Price {
				break // No match possible
			}
		}

		// Execute trade at the resting order's price
		execPrice := bestOrder.Price
		execQty := min(order.RemainingQty(), bestOrder.RemainingQty())

		trade := models.Trade{
			ID:         uuid.New().String(),
			OrderID:    order.ID,
			Symbol:     order.Symbol,
			Side:       order.Side,
			Quantity:   execQty,
			Price:      execPrice,
			ExecutedAt: time.Now(),
		}
		trades = append(trades, trade)

		// Update order fills
		order.FilledQty += execQty
		order.AvgPrice = updateAvgPrice(order.AvgPrice, order.FilledQty-execQty, execPrice, execQty)
		order.UpdatedAt = time.Now()

		bestOrder.FilledQty += execQty
		bestOrder.AvgPrice = updateAvgPrice(bestOrder.AvgPrice, bestOrder.FilledQty-execQty, execPrice, execQty)
		bestOrder.UpdatedAt = time.Now()

		// Update statuses
		if order.IsFilled() {
			order.Status = models.Filled
		} else {
			order.Status = models.Partial
		}

		if bestOrder.IsFilled() {
			bestOrder.Status = models.Filled
			heap.Pop(oppositeBook)
			delete(ob.orders, bestOrder.ID)
		} else {
			bestOrder.Status = models.Partial
		}
	}

	// If order is not fully filled and is a limit order, add remainder to book
	if !order.IsFilled() && order.Type == models.Limit {
		ob.orders[order.ID] = order
		if order.Side == models.Buy {
			heap.Push(ob.bids, order)
		} else {
			heap.Push(ob.asks, order)
		}
	}

	return trades
}

// GetBestBid returns the best bid price and quantity
func (ob *OrderBook) GetBestBid() (float64, int64) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	for ob.bids.Len() > 0 {
		best := ob.bids.Peek()
		if best.Status != models.Canceled {
			return best.Price, best.RemainingQty()
		}
		heap.Pop(ob.bids)
	}
	return 0, 0
}

// GetBestAsk returns the best ask price and quantity
func (ob *OrderBook) GetBestAsk() (float64, int64) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	for ob.asks.Len() > 0 {
		best := ob.asks.Peek()
		if best.Status != models.Canceled {
			return best.Price, best.RemainingQty()
		}
		heap.Pop(ob.asks)
	}
	return 0, 0
}

// BookDepth represents order book depth at a price level
type BookDepth struct {
	Bids []PriceLevel `json:"bids"`
	Asks []PriceLevel `json:"asks"`
}

// PriceLevel represents aggregated orders at a price
type PriceLevel struct {
	Price    float64 `json:"price"`
	Quantity int64   `json:"quantity"`
	Count    int     `json:"count"`
}

// GetDepth returns aggregated order book depth
func (ob *OrderBook) GetDepth(levels int) BookDepth {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	depth := BookDepth{
		Bids: aggregateLevels(ob.bids, levels),
		Asks: aggregateLevels(ob.asks, levels),
	}
	return depth
}

func aggregateLevels(h *orderHeap, maxLevels int) []PriceLevel {
	levels := make(map[float64]*PriceLevel)
	var prices []float64

	for _, order := range h.orders {
		if order.Status == models.Canceled {
			continue
		}
		if pl, exists := levels[order.Price]; exists {
			pl.Quantity += order.RemainingQty()
			pl.Count++
		} else {
			levels[order.Price] = &PriceLevel{
				Price:    order.Price,
				Quantity: order.RemainingQty(),
				Count:    1,
			}
			prices = append(prices, order.Price)
		}
	}

	// Sort and limit
	result := make([]PriceLevel, 0, min(len(prices), maxLevels))
	for i := 0; i < len(prices) && i < maxLevels; i++ {
		result = append(result, *levels[prices[i]])
	}
	return result
}

func updateAvgPrice(currentAvg float64, currentQty int64, newPrice float64, newQty int64) float64 {
	totalQty := currentQty + newQty
	if totalQty == 0 {
		return 0
	}
	return (currentAvg*float64(currentQty) + newPrice*float64(newQty)) / float64(totalQty)
}

// orderHeap implements heap.Interface for price-time priority
type orderHeap struct {
	orders    []*models.Order
	isMaxHeap bool // true for bids (max), false for asks (min)
}

func (h orderHeap) Len() int { return len(h.orders) }

func (h orderHeap) Less(i, j int) bool {
	// Price priority first
	if h.orders[i].Price != h.orders[j].Price {
		if h.isMaxHeap {
			return h.orders[i].Price > h.orders[j].Price // Max heap: higher price first
		}
		return h.orders[i].Price < h.orders[j].Price // Min heap: lower price first
	}
	// Time priority (FIFO) - earlier orders first
	return h.orders[i].CreatedAt.Before(h.orders[j].CreatedAt)
}

func (h orderHeap) Swap(i, j int) {
	h.orders[i], h.orders[j] = h.orders[j], h.orders[i]
}

func (h *orderHeap) Push(x interface{}) {
	h.orders = append(h.orders, x.(*models.Order))
}

func (h *orderHeap) Pop() interface{} {
	old := h.orders
	n := len(old)
	order := old[n-1]
	h.orders = old[0 : n-1]
	return order
}

func (h *orderHeap) Peek() *models.Order {
	if len(h.orders) == 0 {
		return nil
	}
	return h.orders[0]
}
