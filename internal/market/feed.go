package market

import "github.com/cjunker/go-trading-system/pkg/models"

// Feed provides market data quotes
type Feed interface {
	// Start begins generating/receiving price updates
	Start() error

	// Stop gracefully shuts down the feed
	Stop() error

	// Subscribe returns a channel for receiving quotes
	Subscribe() <-chan models.Quote

	// GetQuote returns the latest quote for a symbol
	GetQuote(symbol string) (models.Quote, bool)

	// Symbols returns the list of symbols being tracked
	Symbols() []string
}
