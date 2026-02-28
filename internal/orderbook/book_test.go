package orderbook

import (
	"testing"
	"time"

	"github.com/cjunker/go-trading-system/pkg/models"
)

func TestNewOrderBook(t *testing.T) {
	book := NewOrderBook("AAPL")
	if book.Symbol() != "AAPL" {
		t.Errorf("expected symbol AAPL, got %s", book.Symbol())
	}
}

func TestAddLimitOrder(t *testing.T) {
	book := NewOrderBook("AAPL")

	order := &models.Order{
		ID:        "order-1",
		Symbol:    "AAPL",
		Side:      models.Buy,
		Type:      models.Limit,
		Price:     150.00,
		Quantity:  100,
		Status:    models.Pending,
		CreatedAt: time.Now(),
	}

	book.AddOrder(order)

	price, qty := book.GetBestBid()
	if price != 150.00 {
		t.Errorf("expected best bid 150.00, got %f", price)
	}
	if qty != 100 {
		t.Errorf("expected qty 100, got %d", qty)
	}
}

func TestMatchMarketBuyOrder(t *testing.T) {
	book := NewOrderBook("AAPL")

	// Add sell limit order to the book
	sellOrder := &models.Order{
		ID:        "sell-1",
		Symbol:    "AAPL",
		Side:      models.Sell,
		Type:      models.Limit,
		Price:     150.00,
		Quantity:  100,
		Status:    models.Pending,
		CreatedAt: time.Now(),
	}
	book.AddOrder(sellOrder)

	// Submit market buy order
	buyOrder := &models.Order{
		ID:        "buy-1",
		Symbol:    "AAPL",
		Side:      models.Buy,
		Type:      models.Market,
		Quantity:  50,
		Status:    models.Pending,
		CreatedAt: time.Now(),
	}

	trades := book.Match(buyOrder)

	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}

	if trades[0].Quantity != 50 {
		t.Errorf("expected trade quantity 50, got %d", trades[0].Quantity)
	}

	if trades[0].Price != 150.00 {
		t.Errorf("expected trade price 150.00, got %f", trades[0].Price)
	}

	if buyOrder.FilledQty != 50 {
		t.Errorf("expected filled qty 50, got %d", buyOrder.FilledQty)
	}

	if buyOrder.Status != models.Filled {
		t.Errorf("expected status FILLED, got %s", buyOrder.Status)
	}
}

func TestMatchLimitBuyOrder(t *testing.T) {
	book := NewOrderBook("AAPL")

	// Add sell limit order
	sellOrder := &models.Order{
		ID:        "sell-1",
		Symbol:    "AAPL",
		Side:      models.Sell,
		Type:      models.Limit,
		Price:     150.00,
		Quantity:  100,
		Status:    models.Pending,
		CreatedAt: time.Now(),
	}
	book.AddOrder(sellOrder)

	// Submit limit buy at matching price
	buyOrder := &models.Order{
		ID:        "buy-1",
		Symbol:    "AAPL",
		Side:      models.Buy,
		Type:      models.Limit,
		Price:     150.00,
		Quantity:  100,
		Status:    models.Pending,
		CreatedAt: time.Now(),
	}

	trades := book.Match(buyOrder)

	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}

	if trades[0].Quantity != 100 {
		t.Errorf("expected trade quantity 100, got %d", trades[0].Quantity)
	}
}

func TestNoMatchWhenPricesDontCross(t *testing.T) {
	book := NewOrderBook("AAPL")

	// Add sell limit order at 150
	sellOrder := &models.Order{
		ID:        "sell-1",
		Symbol:    "AAPL",
		Side:      models.Sell,
		Type:      models.Limit,
		Price:     150.00,
		Quantity:  100,
		Status:    models.Pending,
		CreatedAt: time.Now(),
	}
	book.AddOrder(sellOrder)

	// Submit limit buy at lower price (no match)
	buyOrder := &models.Order{
		ID:        "buy-1",
		Symbol:    "AAPL",
		Side:      models.Buy,
		Type:      models.Limit,
		Price:     149.00,
		Quantity:  100,
		Status:    models.Pending,
		CreatedAt: time.Now(),
	}

	trades := book.Match(buyOrder)

	if len(trades) != 0 {
		t.Errorf("expected 0 trades, got %d", len(trades))
	}

	// Buy order should be added to book
	price, qty := book.GetBestBid()
	if price != 149.00 || qty != 100 {
		t.Errorf("expected bid 149.00/100, got %f/%d", price, qty)
	}
}

func TestPartialFill(t *testing.T) {
	book := NewOrderBook("AAPL")

	// Add sell order for 50 shares
	sellOrder := &models.Order{
		ID:        "sell-1",
		Symbol:    "AAPL",
		Side:      models.Sell,
		Type:      models.Limit,
		Price:     150.00,
		Quantity:  50,
		Status:    models.Pending,
		CreatedAt: time.Now(),
	}
	book.AddOrder(sellOrder)

	// Submit buy order for 100 shares
	buyOrder := &models.Order{
		ID:        "buy-1",
		Symbol:    "AAPL",
		Side:      models.Buy,
		Type:      models.Limit,
		Price:     150.00,
		Quantity:  100,
		Status:    models.Pending,
		CreatedAt: time.Now(),
	}

	trades := book.Match(buyOrder)

	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}

	if trades[0].Quantity != 50 {
		t.Errorf("expected trade quantity 50, got %d", trades[0].Quantity)
	}

	if buyOrder.FilledQty != 50 {
		t.Errorf("expected filled qty 50, got %d", buyOrder.FilledQty)
	}

	if buyOrder.Status != models.Partial {
		t.Errorf("expected status PARTIAL, got %s", buyOrder.Status)
	}

	// Remaining 50 should be on the book
	price, qty := book.GetBestBid()
	if price != 150.00 || qty != 50 {
		t.Errorf("expected bid 150.00/50, got %f/%d", price, qty)
	}
}

func TestPriceTimePriority(t *testing.T) {
	book := NewOrderBook("AAPL")

	// Add first sell order
	sell1 := &models.Order{
		ID:        "sell-1",
		Symbol:    "AAPL",
		Side:      models.Sell,
		Type:      models.Limit,
		Price:     150.00,
		Quantity:  50,
		Status:    models.Pending,
		CreatedAt: time.Now(),
	}
	book.AddOrder(sell1)

	// Add second sell order at same price (later time)
	time.Sleep(time.Millisecond)
	sell2 := &models.Order{
		ID:        "sell-2",
		Symbol:    "AAPL",
		Side:      models.Sell,
		Type:      models.Limit,
		Price:     150.00,
		Quantity:  50,
		Status:    models.Pending,
		CreatedAt: time.Now(),
	}
	book.AddOrder(sell2)

	// Submit buy order
	buyOrder := &models.Order{
		ID:        "buy-1",
		Symbol:    "AAPL",
		Side:      models.Buy,
		Type:      models.Market,
		Quantity:  50,
		Status:    models.Pending,
		CreatedAt: time.Now(),
	}

	trades := book.Match(buyOrder)

	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}

	// Should match with first order (time priority)
	if trades[0].Quantity != 50 {
		t.Errorf("expected trade quantity 50, got %d", trades[0].Quantity)
	}

	if sell1.FilledQty != 50 {
		t.Errorf("expected sell1 filled, got %d", sell1.FilledQty)
	}

	if sell2.FilledQty != 0 {
		t.Errorf("expected sell2 not filled, got %d", sell2.FilledQty)
	}
}

func TestCancelOrder(t *testing.T) {
	book := NewOrderBook("AAPL")

	order := &models.Order{
		ID:        "order-1",
		Symbol:    "AAPL",
		Side:      models.Buy,
		Type:      models.Limit,
		Price:     150.00,
		Quantity:  100,
		Status:    models.Pending,
		CreatedAt: time.Now(),
	}
	book.AddOrder(order)

	if !book.CancelOrder("order-1") {
		t.Error("expected cancel to succeed")
	}

	if order.Status != models.Cancelled {
		t.Errorf("expected status CANCELLED, got %s", order.Status)
	}
}

// Benchmark tests
func BenchmarkOrderMatching(b *testing.B) {
	book := NewOrderBook("AAPL")

	// Pre-populate with orders
	for i := 0; i < 1000; i++ {
		book.AddOrder(&models.Order{
			ID:        string(rune(i)),
			Symbol:    "AAPL",
			Side:      models.Sell,
			Type:      models.Limit,
			Price:     150.00 + float64(i)*0.01,
			Quantity:  100,
			Status:    models.Pending,
			CreatedAt: time.Now(),
		})
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		book.Match(&models.Order{
			ID:        "buy",
			Symbol:    "AAPL",
			Side:      models.Buy,
			Type:      models.Market,
			Quantity:  10,
			Status:    models.Pending,
			CreatedAt: time.Now(),
		})
	}
}

func BenchmarkAddOrder(b *testing.B) {
	book := NewOrderBook("AAPL")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		book.AddOrder(&models.Order{
			ID:        string(rune(i)),
			Symbol:    "AAPL",
			Side:      models.Buy,
			Type:      models.Limit,
			Price:     150.00,
			Quantity:  100,
			Status:    models.Pending,
			CreatedAt: time.Now(),
		})
	}
}
