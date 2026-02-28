package trading

import (
	"errors"
	"sync/atomic"

	"github.com/cjunker/go-trading-system/pkg/models"
)

var (
	ErrPositionLimitExceeded = errors.New("position limit exceeded")
	ErrOrderSizeExceeded     = errors.New("order size exceeds limit")
	ErrDailyLossExceeded     = errors.New("daily loss limit exceeded - trading halted")
)

// RiskManager enforces trading risk limits
type RiskManager struct {
	limits   models.RiskLimits
	dailyPnL atomic.Value // float64
	breached atomic.Bool  // Trading halted if true
}

// NewRiskManager creates a new risk manager with specified limits
func NewRiskManager(limits models.RiskLimits) *RiskManager {
	rm := &RiskManager{limits: limits}
	rm.dailyPnL.Store(0.0)
	return rm
}

// CheckOrder validates an order against risk limits
func (rm *RiskManager) CheckOrder(order *models.Order, currentPosition *models.Position) error {
	// Check if trading is halted
	if rm.breached.Load() {
		return ErrDailyLossExceeded
	}

	// Check order size limit
	if order.Quantity > rm.limits.MaxOrderSize {
		return ErrOrderSizeExceeded
	}

	// Check position limit
	currentQty := int64(0)
	if currentPosition != nil {
		currentQty = currentPosition.Quantity
	}

	var newPosition int64
	if order.Side == models.Buy {
		newPosition = currentQty + order.Quantity
	} else {
		newPosition = currentQty - order.Quantity
	}

	if abs(newPosition) > rm.limits.MaxPositionSize {
		return ErrPositionLimitExceeded
	}

	return nil
}

// UpdateDailyPnL updates the daily P&L and checks loss limits
func (rm *RiskManager) UpdateDailyPnL(pnl float64) {
	rm.dailyPnL.Store(pnl)

	if pnl < -rm.limits.MaxDailyLoss {
		rm.breached.Store(true)
	}
}

// IsBreached returns whether risk limits have been breached
func (rm *RiskManager) IsBreached() bool {
	return rm.breached.Load()
}

// GetDailyPnL returns current daily P&L
func (rm *RiskManager) GetDailyPnL() float64 {
	return rm.dailyPnL.Load().(float64)
}

// Reset resets daily limits (typically called at start of trading day)
func (rm *RiskManager) Reset() {
	rm.dailyPnL.Store(0.0)
	rm.breached.Store(false)
}

// GetLimits returns current risk limits
func (rm *RiskManager) GetLimits() models.RiskLimits {
	return rm.limits
}
