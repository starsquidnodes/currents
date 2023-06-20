package trading

import (
	"time"

	"indexer/token"

	"github.com/ericlagergren/decimal"
)

type (
	Trade struct {
		Base  token.Token `json:"base"`
		Quote token.Token `json:"quote"`
		Time  time.Time   `json:"time"`
	}
)

func (t *Trade) Price() *decimal.Big {
	price := decimal.Big{}
	price.Quo(&t.Quote.Amount, &t.Base.Amount)
	return &price
}

func (t *Trade) Pair() *token.Pair {
	return &token.Pair{
		Base:  t.Base.Symbol,
		Quote: t.Quote.Symbol,
	}
}

func (t *Trade) Reversed() *Trade {
	return &Trade{
		Base:  t.Quote,
		Quote: t.Base,
		Time:  t.Time,
	}
}
