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
)

func CandlesFromTrades(pair *token.Pair, trades []*Trade, start time.Time, end time.Time, interval time.Duration) []*Candle {
	numCandles := int(end.Sub(start) / interval)
	candles := make([]*Candle, numCandles)
	for i := numCandles; i > 0; i-- {
		candles[numCandles - i] = &Candle{
			BaseAsset:  pair.Base,
			QuoteAsset: pair.Quote,
			Start:      start.Add(time.Duration(i - 1) * interval),
			End:        start.Add(time.Duration(i) * interval),
		}
	}
	if len(trades) == 0 {
		return candles
	}
	i := 0
	var trade *Trade
	for ; i < len(trades); i++ {
		trade = trades[i]
		if trade.Time.Before(candles[0].End) {
			break
		}
	}
	for j, candle := range candles {
		var high, low, open, close *decimal.Big
		baseVol := &decimal.Big{}
		quoteVol := &decimal.Big{}
		for ; i < len(trades); i++ {
			trade = trades[i]
			if trade.Time.Compare(candle.Start) <= 0 {
				break
			}
			open = trade.Price()
			if close == nil {
				high = open
				low = open
				close = open
			} else if open.Cmp(high) > 0 {
				high = open
			} else if open.Cmp(low) < 0 {
				low = open
			}
			baseVol.Add(baseVol, &trade.Base.Amount)
			quoteVol.Add(quoteVol, &trade.Quote.Amount)
		}
		if close == nil {
			continue
		}
		candles[j].High = *high
		candles[j].Low = *low
		candles[j].Open = *open
		candles[j].Close = *close
		candles[j].BaseVolume = *baseVol
		candles[j].QuoteVolume = *quoteVol
	}
	return candles
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
