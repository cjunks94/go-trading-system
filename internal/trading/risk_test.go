package trading

import (
	"testing"

	"github.com/cjunker/go-trading-system/pkg/models"
)

func TestRiskCheckOrderSize(t *testing.T) {
	rm := NewRiskManager(models.RiskLimits{
		MaxOrderSize:    100,
		MaxPositionSize: 1000,
		MaxDailyLoss:    10000,
	})

	// Order within limit
	order := &models.Order{
		Symbol:   "AAPL",
		Side:     models.Buy,
		Quantity: 50,
	}

	if err := rm.CheckOrder(order, nil); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Order exceeds limit
	order.Quantity = 150
	if err := rm.CheckOrder(order, nil); err != ErrOrderSizeExceeded {
		t.Errorf("expected ErrOrderSizeExceeded, got %v", err)
	}
}

func TestRiskCheckPositionLimit(t *testing.T) {
	rm := NewRiskManager(models.RiskLimits{
		MaxOrderSize:    1000,
		MaxPositionSize: 500,
		MaxDailyLoss:    10000,
	})

	currentPosition := &models.Position{
		Symbol:   "AAPL",
		Quantity: 400,
	}

	// Order within position limit
	order := &models.Order{
		Symbol:   "AAPL",
		Side:     models.Buy,
		Quantity: 50,
	}

	if err := rm.CheckOrder(order, currentPosition); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Order exceeds position limit
	order.Quantity = 150
	if err := rm.CheckOrder(order, currentPosition); err != ErrPositionLimitExceeded {
		t.Errorf("expected ErrPositionLimitExceeded, got %v", err)
	}
}

func TestRiskCheckDailyLoss(t *testing.T) {
	rm := NewRiskManager(models.RiskLimits{
		MaxOrderSize:    1000,
		MaxPositionSize: 1000,
		MaxDailyLoss:    1000,
	})

	order := &models.Order{
		Symbol:   "AAPL",
		Side:     models.Buy,
		Quantity: 50,
	}

	// Normal trading
	if err := rm.CheckOrder(order, nil); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Exceed daily loss
	rm.UpdateDailyPnL(-1500)

	if !rm.IsBreached() {
		t.Error("expected breached to be true")
	}

	if err := rm.CheckOrder(order, nil); err != ErrDailyLossExceeded {
		t.Errorf("expected ErrDailyLossExceeded, got %v", err)
	}
}

func TestRiskReset(t *testing.T) {
	rm := NewRiskManager(models.RiskLimits{
		MaxDailyLoss: 1000,
	})

	rm.UpdateDailyPnL(-1500)

	if !rm.IsBreached() {
		t.Error("expected breached to be true")
	}

	rm.Reset()

	if rm.IsBreached() {
		t.Error("expected breached to be false after reset")
	}

	if rm.GetDailyPnL() != 0 {
		t.Errorf("expected daily P&L 0, got %f", rm.GetDailyPnL())
	}
}
