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

	pos := pm.getOrCreatePosition(trade.Symbol)
	qty := pm.signedQuantity(trade)

	pm.realizeClosingPL(pos, qty, trade.Price)
	pm.updatePositionQuantity(pos, qty, trade.Price)

	return pos
}

// getOrCreatePosition returns existing position or creates new one
func (pm *PositionManager) getOrCreatePosition(symbol string) *models.Position {
	pos, exists := pm.positions[symbol]
	if !exists {
		pos = &models.Position{Symbol: symbol}
		pm.positions[symbol] = pos
	}
	return pos
}

// signedQuantity converts trade to signed quantity (negative for sells)
func (pm *PositionManager) signedQuantity(trade models.Trade) int64 {
	if trade.Side == models.Sell {
		return -trade.Quantity
	}
	return trade.Quantity
}

// realizeClosingPL calculates and adds realized P&L when closing positions
func (pm *PositionManager) realizeClosingPL(pos *models.Position, qty int64, price float64) {
	if !isClosingPosition(pos.Quantity, qty) {
		return
	}

	closeQty := min(abs(pos.Quantity), abs(qty))
	if pos.Quantity > 0 {
		pos.RealizedPL += float64(closeQty) * (price - pos.AvgCost)
	} else {
		pos.RealizedPL += float64(closeQty) * (pos.AvgCost - price)
	}
}

// updatePositionQuantity updates position quantity and average cost
func (pm *PositionManager) updatePositionQuantity(pos *models.Position, qty int64, price float64) {
	oldQty := pos.Quantity
	newQty := oldQty + qty

	switch {
	case newQty == 0:
		pos.Quantity = 0
		pos.AvgCost = 0
	case isAddingToPosition(oldQty, qty):
		pos.AvgCost = weightedAverage(pos.AvgCost, abs(oldQty), price, abs(qty))
		pos.Quantity = newQty
	default:
		pos.Quantity = newQty
		if isFlippingPosition(oldQty, newQty) {
			pos.AvgCost = price
		}
	}
}

// isClosingPosition returns true if trade reduces position size
func isClosingPosition(posQty, tradeQty int64) bool {
	return (posQty > 0 && tradeQty < 0) || (posQty < 0 && tradeQty > 0)
}

// isAddingToPosition returns true if trade increases position in same direction
func isAddingToPosition(oldQty, qty int64) bool {
	return (oldQty >= 0 && qty > 0) || (oldQty <= 0 && qty < 0)
}

// isFlippingPosition returns true if position changed from long to short or vice versa
func isFlippingPosition(oldQty, newQty int64) bool {
	return (oldQty > 0 && newQty < 0) || (oldQty < 0 && newQty > 0)
}

// weightedAverage calculates weighted average of two values
func weightedAverage(val1 float64, weight1 int64, val2 float64, weight2 int64) float64 {
	totalWeight := weight1 + weight2
	return (val1*float64(weight1) + val2*float64(weight2)) / float64(totalWeight)
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
