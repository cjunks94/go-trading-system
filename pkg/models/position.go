package models

// Position represents holdings in a symbol
type Position struct {
	Symbol       string  `json:"symbol"`
	Quantity     int64   `json:"quantity"`     // positive = long, negative = short
	AvgCost      float64 `json:"avgCost"`      // average entry price
	MarketValue  float64 `json:"marketValue"`  // current market value
	UnrealizedPL float64 `json:"unrealizedPL"` // unrealized P&L
	RealizedPL   float64 `json:"realizedPL"`   // realized P&L from closed trades
}

// TotalPL returns total P&L (realized + unrealized)
func (p *Position) TotalPL() float64 {
	return p.RealizedPL + p.UnrealizedPL
}

// IsLong returns true if position is long
func (p *Position) IsLong() bool {
	return p.Quantity > 0
}

// IsShort returns true if position is short
func (p *Position) IsShort() bool {
	return p.Quantity < 0
}

// IsFlat returns true if no position
func (p *Position) IsFlat() bool {
	return p.Quantity == 0
}

// RiskLimits defines trading constraints
type RiskLimits struct {
	MaxPositionSize int64   `json:"maxPositionSize"` // per symbol
	MaxOrderSize    int64   `json:"maxOrderSize"`    // single order limit
	MaxDailyLoss    float64 `json:"maxDailyLoss"`    // stop trading if exceeded
}

// PnLSummary provides overall P&L information
type PnLSummary struct {
	TotalRealizedPL   float64 `json:"totalRealizedPL"`
	TotalUnrealizedPL float64 `json:"totalUnrealizedPL"`
	TotalPL           float64 `json:"totalPL"`
	DailyPL           float64 `json:"dailyPL"`
}
