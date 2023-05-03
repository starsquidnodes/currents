package trading

import (
	"time"
	
	"github.com/ericlagergren/decimal"
	"github.com/mintthemoon/chaindex/token"
)

type (
	Trade interface {
		BaseVolume() *decimal.Big
		QuoteVolume() *decimal.Big
		BaseAsset() string
		QuoteAsset() string
		Timestamp() time.Time
		Price() *decimal.Big
	}

	BasicTrade struct {
		Base token.Token
		Quote token.Token
		Time time.Time
	}
)

func (b *BasicTrade) BaseAsset() string {
	return b.Base.Symbol
}

func (b *BasicTrade) QuoteAsset() string {
	return b.Quote.Symbol
}

func (b *BasicTrade) BaseVolume() *decimal.Big {
	return &b.Base.Amount
}

func (b *BasicTrade) QuoteVolume() *decimal.Big {
	return &b.Quote.Amount
}

func (b *BasicTrade) Timestamp() time.Time {
	return b.Time
}

func (b *BasicTrade) Price() *decimal.Big {
	price := decimal.Big{}
	price.Quo(&b.Quote.Amount, &b.Base.Amount)
	return &price
}