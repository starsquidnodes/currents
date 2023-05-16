package trading

import (
	"time"

	"github.com/ericlagergren/decimal"
)

type Ticker struct {
	BaseAsset string `json:"base_asset"`
	QuoteAsset string `json:"quote_asset"`
	BaseVolume decimal.Big `json:"base_volume"`
	QuoteVolume decimal.Big `json:"quote_volume"`
	Price decimal.Big `json:"price"`
	Time time.Time `json:"time"`
}

func (t *Ticker) Reversed() *Ticker {
	r := &Ticker{
		BaseAsset: t.QuoteAsset,
		QuoteAsset: t.BaseAsset,
		BaseVolume: t.QuoteVolume,
		QuoteVolume: t.BaseVolume,
		Time: t.Time,
	}
	if t.Price.Cmp(&decimal.Big{}) != 0 {
		one := &decimal.Big{}
		one.SetUint64(1)
		r.Price.Quo(one, &t.Price)
	}
	return r
}