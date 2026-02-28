package models

import (
	"errors"
	"time"
)

type OrderSide string
type OrderType string
type OrderStatus string

const (
	Buy  OrderSide = "BUY"
	Sell OrderSide = "SELL"

	Market OrderType = "MARKET"
	Limit  OrderType = "LIMIT"

	Pending   OrderStatus = "PENDING"
	Filled    OrderStatus = "FILLED"
	Partial   OrderStatus = "PARTIAL"
	Canceled OrderStatus = "CANCELED"
	Rejected  OrderStatus = "REJECTED"
)

var (
	ErrInvalidSymbol   = errors.New("invalid symbol")
	ErrInvalidSide     = errors.New("invalid order side")
	ErrInvalidType     = errors.New("invalid order type")
	ErrInvalidQuantity = errors.New("invalid quantity")
	ErrInvalidPrice    = errors.New("invalid price for limit order")
)

// Order represents a trading order
type Order struct {
	ID        string      `json:"id"`
	Symbol    string      `json:"symbol"`
	Side      OrderSide   `json:"side"`
	Type      OrderType   `json:"type"`
	Quantity  int64       `json:"quantity"`
	Price     float64     `json:"price,omitempty"`
	FilledQty int64       `json:"filledQty"`
	AvgPrice  float64     `json:"avgPrice"`
	Status    OrderStatus `json:"status"`
	CreatedAt time.Time   `json:"createdAt"`
	UpdatedAt time.Time   `json:"updatedAt"`
}

// OrderRequest is the incoming order submission
type OrderRequest struct {
	Symbol   string    `json:"symbol"`
	Side     OrderSide `json:"side"`
	Type     OrderType `json:"type"`
	Quantity int64     `json:"quantity"`
	Price    float64   `json:"price,omitempty"`
}

// Validate checks if the order request is valid
func (r *OrderRequest) Validate() error {
	if r.Symbol == "" {
		return ErrInvalidSymbol
	}
	if r.Side != Buy && r.Side != Sell {
		return ErrInvalidSide
	}
	if r.Type != Market && r.Type != Limit {
		return ErrInvalidType
	}
	if r.Quantity <= 0 {
		return ErrInvalidQuantity
	}
	if r.Type == Limit && r.Price <= 0 {
		return ErrInvalidPrice
	}
	return nil
}

// RemainingQty returns unfilled quantity
func (o *Order) RemainingQty() int64 {
	return o.Quantity - o.FilledQty
}

// IsFilled returns true if order is completely filled
func (o *Order) IsFilled() bool {
	return o.FilledQty >= o.Quantity
}
