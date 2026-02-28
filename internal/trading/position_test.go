package trading

import (
	"testing"

	"github.com/cjunker/go-trading-system/pkg/models"
)

func TestOpenLongPosition(t *testing.T) {
	pm := NewPositionManager()

	trade := models.Trade{
		ID:       "trade-1",
		OrderID:  "order-1",
		Symbol:   "AAPL",
		Side:     models.Buy,
		Quantity: 100,
		Price:    150.00,
	}

	pos := pm.UpdateFromTrade(trade)

	if pos.Symbol != "AAPL" {
		t.Errorf("expected symbol AAPL, got %s", pos.Symbol)
	}
	if pos.Quantity != 100 {
		t.Errorf("expected quantity 100, got %d", pos.Quantity)
	}
	if pos.AvgCost != 150.00 {
		t.Errorf("expected avg cost 150.00, got %f", pos.AvgCost)
	}
}

func TestCloseLongPosition(t *testing.T) {
	pm := NewPositionManager()

	// Open position
	pm.UpdateFromTrade(models.Trade{
		ID:       "trade-1",
		Symbol:   "AAPL",
		Side:     models.Buy,
		Quantity: 100,
		Price:    150.00,
	})

	// Close position at profit
	pos := pm.UpdateFromTrade(models.Trade{
		ID:       "trade-2",
		Symbol:   "AAPL",
		Side:     models.Sell,
		Quantity: 100,
		Price:    160.00,
	})

	if pos.Quantity != 0 {
		t.Errorf("expected quantity 0, got %d", pos.Quantity)
	}

	// Realized P&L should be $10 * 100 = $1000
	expectedPL := 1000.0
	if pos.RealizedPL != expectedPL {
		t.Errorf("expected realized P&L %f, got %f", expectedPL, pos.RealizedPL)
	}
}

func TestPartialClose(t *testing.T) {
	pm := NewPositionManager()

	// Open position
	pm.UpdateFromTrade(models.Trade{
		Symbol:   "AAPL",
		Side:     models.Buy,
		Quantity: 100,
		Price:    150.00,
	})

	// Partial close
	pos := pm.UpdateFromTrade(models.Trade{
		Symbol:   "AAPL",
		Side:     models.Sell,
		Quantity: 50,
		Price:    160.00,
	})

	if pos.Quantity != 50 {
		t.Errorf("expected quantity 50, got %d", pos.Quantity)
	}

	// Realized P&L should be $10 * 50 = $500
	expectedPL := 500.0
	if pos.RealizedPL != expectedPL {
		t.Errorf("expected realized P&L %f, got %f", expectedPL, pos.RealizedPL)
	}
}

func TestShortPosition(t *testing.T) {
	pm := NewPositionManager()

	// Open short
	pos := pm.UpdateFromTrade(models.Trade{
		Symbol:   "AAPL",
		Side:     models.Sell,
		Quantity: 100,
		Price:    150.00,
	})

	if pos.Quantity != -100 {
		t.Errorf("expected quantity -100, got %d", pos.Quantity)
	}

	// Close short at profit (price went down)
	pos = pm.UpdateFromTrade(models.Trade{
		Symbol:   "AAPL",
		Side:     models.Buy,
		Quantity: 100,
		Price:    140.00,
	})

	if pos.Quantity != 0 {
		t.Errorf("expected quantity 0, got %d", pos.Quantity)
	}

	// Realized P&L should be $10 * 100 = $1000
	expectedPL := 1000.0
	if pos.RealizedPL != expectedPL {
		t.Errorf("expected realized P&L %f, got %f", expectedPL, pos.RealizedPL)
	}
}

func TestUpdateMarketValue(t *testing.T) {
	pm := NewPositionManager()

	// Open position
	pm.UpdateFromTrade(models.Trade{
		Symbol:   "AAPL",
		Side:     models.Buy,
		Quantity: 100,
		Price:    150.00,
	})

	// Update market value
	pos := pm.UpdateMarketValue("AAPL", 160.00)

	if pos.MarketValue != 16000.00 {
		t.Errorf("expected market value 16000, got %f", pos.MarketValue)
	}

	// Unrealized P&L should be $10 * 100 = $1000
	expectedPL := 1000.0
	if pos.UnrealizedPL != expectedPL {
		t.Errorf("expected unrealized P&L %f, got %f", expectedPL, pos.UnrealizedPL)
	}
}

func TestPnLSummary(t *testing.T) {
	pm := NewPositionManager()

	// Open and close position with profit
	pm.UpdateFromTrade(models.Trade{Symbol: "AAPL", Side: models.Buy, Quantity: 100, Price: 150.00})
	pm.UpdateFromTrade(models.Trade{Symbol: "AAPL", Side: models.Sell, Quantity: 100, Price: 160.00})

	// Open another position
	pm.UpdateFromTrade(models.Trade{Symbol: "GOOGL", Side: models.Buy, Quantity: 50, Price: 140.00})
	pm.UpdateMarketValue("GOOGL", 145.00) // $5 * 50 = $250 unrealized

	summary := pm.GetPnLSummary()

	if summary.TotalRealizedPL != 1000.0 {
		t.Errorf("expected realized P&L 1000, got %f", summary.TotalRealizedPL)
	}

	if summary.TotalUnrealizedPL != 250.0 {
		t.Errorf("expected unrealized P&L 250, got %f", summary.TotalUnrealizedPL)
	}

	if summary.TotalPL != 1250.0 {
		t.Errorf("expected total P&L 1250, got %f", summary.TotalPL)
	}
}
