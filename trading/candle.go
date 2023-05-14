package trading

import (
	"time"

	"github.com/ericlagergren/decimal"
	"github.com/mintthemoon/currents/token"
)

type (
	Candle struct {
		BaseAsset   string      `json:"base_asset"`
		QuoteAsset  string      `json:"quote_asset"`
		BaseVolume  decimal.Big `json:"base_volume"`
		QuoteVolume decimal.Big `json:"quote_volume"`
		High        decimal.Big `json:"high"`
		Low         decimal.Big `json:"low"`
		Open        decimal.Big `json:"open"`
		Close       decimal.Big `json:"close"`
		Start       time.Time   `json:"start"`
		End         time.Time   `json:"end"`
	}

	Candles struct {
	}
)

func CandlesFromTrades(pair *token.Pair, trades []*Trade, start time.Time, end time.Time, interval time.Duration) []*Candle {
	numCandles := int(end.Sub(start) / interval)
	candles := make([]*Candle, numCandles)
	for i := 0; i < numCandles; i++ {
		candles[i] = &Candle{
			BaseAsset:  pair.Base,
			QuoteAsset: pair.Quote,
			Start:      start.Add(time.Duration(i) * interval),
			End:        start.Add(time.Duration(i+1) * interval),
		}
	}
	for _, trade := range trades {
		if trade.Timestamp().Before(start) {
			continue
		}
		if trade.Timestamp().After(end) || trade.Timestamp().Equal(end) {
			break
		}
		candles[int(trade.Timestamp().Sub(start)/interval)].AddTrade(trade)
	}
	return candles
}

func (c *Candle) AddTrade(t *Trade) {
	if (c.Open.Cmp(&decimal.Big{}) == 0) {
		c.Open = *t.Price()
		c.Low = *t.Price()
	}
	c.Close = *t.Price()
	if c.High.Cmp(t.Price()) < 0 {
		c.High = *t.Price()
	}
	if c.Low.Cmp(t.Price()) > 0 {
		c.Low = *t.Price()
	}
	c.BaseVolume.Add(&c.BaseVolume, t.BaseVolume())
	c.QuoteVolume.Add(&c.QuoteVolume, t.QuoteVolume())
}

func (c *Candle) Reversed() *Candle {
	r := Candle{
		BaseAsset:   c.QuoteAsset,
		QuoteAsset:  c.BaseAsset,
		BaseVolume:  c.QuoteVolume,
		QuoteVolume: c.BaseVolume,
		Start:       c.Start,
		End:         c.End,
	}
	zero := decimal.Big{}
	one := decimal.Big{}
	one.SetUint64(1)
	if c.Open.Cmp(&zero) != 0 {
		r.Open.Quo(&one, &c.Open)
	}
	if c.Close.Cmp(&zero) != 0 {
		r.Close.Quo(&one, &c.Close)
	}
	if c.Low.Cmp(&zero) != 0 {
		r.High.Quo(&one, &c.Low)
	}
	if c.High.Cmp(&zero) != 0 {
		r.Low.Quo(&one, &c.High)
	}
	return &r
}
