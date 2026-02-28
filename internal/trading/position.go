package trading

import (
	"sync"

	"github.com/cjunker/go-trading-system/pkg/models"
)

// PositionManager tracks positions and P&L for all symbols
type PositionManager struct {
	positions map[string]*models.Position
	mu        sync.RWMutex
}

// NewPositionManager creates a new position manager
func NewPositionManager() *PositionManager {
	return &PositionManager{
		positions: make(map[string]*models.Position),
	}
}

// UpdateFromTrade updates positions based on an executed trade
func (pm *PositionManager) UpdateFromTrade(trade models.Trade) *models.Position {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pos, exists := pm.positions[trade.Symbol]
	if !exists {
		pos = &models.Position{Symbol: trade.Symbol}
		pm.positions[trade.Symbol] = pos
	}

	// Calculate position change
	qty := trade.Quantity
	if trade.Side == models.Sell {
		qty = -qty
	}

	// Check if this trade is closing or opening position
	if (pos.Quantity > 0 && qty < 0) || (pos.Quantity < 0 && qty > 0) {
		// Closing position - realize P&L
		closeQty := min(abs(pos.Quantity), abs(qty))
		if pos.Quantity > 0 {
			// Closing long position
			pos.RealizedPL += float64(closeQty) * (trade.Price - pos.AvgCost)
		} else {
			// Closing short position
			pos.RealizedPL += float64(closeQty) * (pos.AvgCost - trade.Price)
		}
	}

	// Update position
	oldQty := pos.Quantity
	newQty := oldQty + qty

	if newQty == 0 {
		pos.Quantity = 0
		pos.AvgCost = 0
	} else if (oldQty >= 0 && qty > 0) || (oldQty <= 0 && qty < 0) {
		// Adding to position - update average cost
		pos.AvgCost = (pos.AvgCost*float64(abs(oldQty)) + trade.Price*float64(abs(qty))) / float64(abs(newQty))
		pos.Quantity = newQty
	} else {
		// Partial close with flip
		pos.Quantity = newQty
		if (oldQty > 0 && newQty < 0) || (oldQty < 0 && newQty > 0) {
			pos.AvgCost = trade.Price // New position at current price
		}
	}

	return pos
}

// UpdateMarketValue updates unrealized P&L based on current price
func (pm *PositionManager) UpdateMarketValue(symbol string, currentPrice float64) *models.Position {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pos, exists := pm.positions[symbol]
	if !exists || pos.Quantity == 0 {
		return nil
	}

	pos.MarketValue = float64(abs(pos.Quantity)) * currentPrice

	if pos.Quantity > 0 {
		pos.UnrealizedPL = float64(pos.Quantity) * (currentPrice - pos.AvgCost)
	} else {
		pos.UnrealizedPL = float64(-pos.Quantity) * (pos.AvgCost - currentPrice)
	}

	return pos
}

// GetPosition returns a copy of the position for a symbol
func (pm *PositionManager) GetPosition(symbol string) *models.Position {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if pos, exists := pm.positions[symbol]; exists {
		copy := *pos
		return &copy
	}
	return nil
}

// GetAllPositions returns copies of all positions
func (pm *PositionManager) GetAllPositions() []models.Position {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make([]models.Position, 0, len(pm.positions))
	for _, pos := range pm.positions {
		if pos.Quantity != 0 {
			result = append(result, *pos)
		}
	}
	return result
}

// GetPnLSummary returns aggregated P&L information
func (pm *PositionManager) GetPnLSummary() models.PnLSummary {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var summary models.PnLSummary
	for _, pos := range pm.positions {
		summary.TotalRealizedPL += pos.RealizedPL
		summary.TotalUnrealizedPL += pos.UnrealizedPL
	}
	summary.TotalPL = summary.TotalRealizedPL + summary.TotalUnrealizedPL
	summary.DailyPL = summary.TotalPL // Simplified - in production would track daily

	return summary
}

func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
