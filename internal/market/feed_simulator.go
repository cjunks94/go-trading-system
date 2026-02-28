package market

import (
	"context"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/cjunker/go-trading-system/pkg/models"
)

// SimulatedFeed generates realistic price movements using Brownian motion
type SimulatedFeed struct {
	symbols     []string
	quotes      chan models.Quote     // Internal quote aggregation
	subscribers []chan models.Quote   // Fan-out to subscribers
	latestQuote map[string]models.Quote // Latest quote per symbol
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// NewSimulatedFeed creates a new simulated market data feed
func NewSimulatedFeed(symbols []string) *SimulatedFeed {
	return &SimulatedFeed{
		symbols:     symbols,
		quotes:      make(chan models.Quote, 1000), // Buffered for backpressure
		subscribers: make([]chan models.Quote, 0),
		latestQuote: make(map[string]models.Quote),
	}
}

// Start begins generating price updates
func (f *SimulatedFeed) Start() error {
	f.ctx, f.cancel = context.WithCancel(context.Background())

	// Initialize prices for each symbol
	basePrices := map[string]float64{
		"AAPL": 175.0, "GOOGL": 140.0, "MSFT": 380.0,
		"AMZN": 180.0, "TSLA": 250.0, "NFLX": 450.0,
		"META": 500.0, "NVDA": 800.0,
	}

	// One goroutine per symbol - demonstrates concurrent producers
	for _, symbol := range f.symbols {
		basePrice := basePrices[symbol]
		if basePrice == 0 {
			basePrice = 100.0 + rand.Float64()*100
		}

		f.wg.Add(1)
		go f.generatePrices(symbol, basePrice)
	}

	// Broadcaster goroutine - fan-out to all subscribers
	f.wg.Add(1)
	go f.broadcast()

	log.Printf("[Feed] Started simulated feed for %d symbols", len(f.symbols))
	return nil
}

// Stop gracefully shuts down the feed
func (f *SimulatedFeed) Stop() error {
	if f.cancel != nil {
		f.cancel()
	}
	f.wg.Wait()

	// Close subscriber channels
	f.mu.Lock()
	for _, sub := range f.subscribers {
		close(sub)
	}
	f.subscribers = nil
	f.mu.Unlock()

	log.Println("[Feed] Stopped simulated feed")
	return nil
}

// Subscribe returns a channel for receiving quotes
func (f *SimulatedFeed) Subscribe() <-chan models.Quote {
	ch := make(chan models.Quote, 100) // Buffered to prevent blocking

	f.mu.Lock()
	f.subscribers = append(f.subscribers, ch)
	f.mu.Unlock()

	return ch
}

// GetQuote returns the latest quote for a symbol
func (f *SimulatedFeed) GetQuote(symbol string) (models.Quote, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	quote, ok := f.latestQuote[symbol]
	return quote, ok
}

// Symbols returns the list of symbols being tracked
func (f *SimulatedFeed) Symbols() []string {
	return f.symbols
}

// generatePrices simulates realistic price movements using Brownian motion
func (f *SimulatedFeed) generatePrices(symbol string, startPrice float64) {
	defer f.wg.Done()

	ticker := time.NewTicker(100 * time.Millisecond) // 10 updates/sec per symbol
	defer ticker.Stop()

	lastPrice := startPrice
	volume := int64(0)

	for {
		select {
		case <-f.ctx.Done():
			return
		case <-ticker.C:
			// Brownian motion for realistic price movement
			// volatility ~0.1% per tick
			change := (rand.Float64() - 0.5) * 0.002
			lastPrice *= (1 + change)

			// Ensure price stays positive
			if lastPrice < 1 {
				lastPrice = 1
			}

			// Spread is typically 0.01-0.05% of price
			spreadPct := 0.0001 + rand.Float64()*0.0004
			spread := lastPrice * spreadPct

			// Volume increment per tick
			volume += int64(rand.Intn(1000))

			quote := models.Quote{
				Symbol:    symbol,
				Bid:       lastPrice - spread/2,
				Ask:       lastPrice + spread/2,
				Last:      lastPrice,
				Volume:    volume,
				Timestamp: time.Now(),
			}

			// Store latest quote
			f.mu.Lock()
			f.latestQuote[symbol] = quote
			f.mu.Unlock()

			// Non-blocking send to aggregation channel (backpressure)
			select {
			case f.quotes <- quote:
			default:
				// Channel full, drop quote (acceptable for real-time data)
			}
		}
	}
}

// broadcast implements fan-out pattern to all subscribers
func (f *SimulatedFeed) broadcast() {
	defer f.wg.Done()

	for {
		select {
		case <-f.ctx.Done():
			return
		case quote := <-f.quotes:
			f.mu.RLock()
			for _, sub := range f.subscribers {
				// Non-blocking send to each subscriber
				select {
				case sub <- quote:
				default:
					// Subscriber too slow, skip (prevents blocking fast consumers)
				}
			}
			f.mu.RUnlock()
		}
	}
}
