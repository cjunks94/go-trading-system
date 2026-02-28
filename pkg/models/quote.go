package models

import "time"

// Quote represents a market data quote for a symbol
type Quote struct {
	Symbol    string    `json:"symbol"`
	Bid       float64   `json:"bid"`
	Ask       float64   `json:"ask"`
	Last      float64   `json:"last"`
	Volume    int64     `json:"volume"`
	Timestamp time.Time `json:"timestamp"`
}

// Spread returns the bid-ask spread
func (q *Quote) Spread() float64 {
	return q.Ask - q.Bid
}

// Mid returns the mid-price
func (q *Quote) Mid() float64 {
	return (q.Bid + q.Ask) / 2
}
