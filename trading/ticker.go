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

func TickerFromCandles(candles []*Candle, start time.Time, end time.Time) *Ticker {
	if len(candles) == 0 {
		return &Ticker{}
	}
	ticker := &Ticker{
		BaseAsset: candles[0].BaseAsset,
		QuoteAsset: candles[0].QuoteAsset,
		Time: candles[0].End,
	}
	zero := &decimal.Big{}
	for _, candle := range candles {
		if candle.End.After(end) {
			continue
		}
		if candle.Start.Before(start) {
			break
		}
		if ticker.Price.Cmp(zero) == 0 && candle.Close.Cmp(zero) != 0 {
			ticker.Price = candle.Close
		}
		ticker.BaseVolume.Add(&ticker.BaseVolume, &candle.BaseVolume)
		ticker.QuoteVolume.Add(&ticker.QuoteVolume, &candle.QuoteVolume)
	}
	return ticker
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