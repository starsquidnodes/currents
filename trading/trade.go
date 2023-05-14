package trading

import (
	"time"

	"github.com/ericlagergren/decimal"
	"github.com/mintthemoon/currents/token"
)

type (
	Trade struct {
		Base  token.Token `json:"base"`
		Quote token.Token `json:"quote"`
		Time  time.Time   `json:"time"`
	}
)

func (b *Trade) BaseAsset() string {
	return b.Base.Symbol
}

func (b *Trade) QuoteAsset() string {
	return b.Quote.Symbol
}

func (b *Trade) BaseVolume() *decimal.Big {
	return &b.Base.Amount
}

func (b *Trade) QuoteVolume() *decimal.Big {
	return &b.Quote.Amount
}

func (b *Trade) Timestamp() time.Time {
	return b.Time
}

func (b *Trade) Price() *decimal.Big {
	price := decimal.Big{}
	price.Quo(&b.Quote.Amount, &b.Base.Amount)
	return &price
}
