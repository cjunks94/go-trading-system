package models

import "time"

// Trade represents an executed trade (fill)
type Trade struct {
	ID         string    `json:"id"`
	OrderID    string    `json:"orderId"`
	Symbol     string    `json:"symbol"`
	Side       OrderSide `json:"side"`
	Quantity   int64     `json:"quantity"`
	Price      float64   `json:"price"`
	ExecutedAt time.Time `json:"executedAt"`
}

// Value returns the notional value of the trade
func (t *Trade) Value() float64 {
	return float64(t.Quantity) * t.Price
}
