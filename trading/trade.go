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

func (b *Trade) Price() *decimal.Big {
	price := decimal.Big{}
	price.Quo(&b.Quote.Amount, &b.Base.Amount)
	return &price
}
