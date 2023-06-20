package trading

import (
	"fmt"
	"time"

	"indexer/math"
	"indexer/token"

	"github.com/ericlagergren/decimal"
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
		Pair     token.Pair
		interval time.Duration
		period   time.Duration
		candles  []Candle
		cutoff   time.Time
	}
)

func NewCandles(pair *token.Pair, trades []*Trade, interval time.Duration, period time.Duration, end time.Time) (*Candles, error) {
	size := int(period/interval) + 1
	candles := &Candles{
		interval: interval,
		period:   period,
		Pair:     *pair,
		candles:  make([]Candle, size),
	}
	candles.Reset(end)
	return candles, candles.SetTrades(trades)
}

func (c *Candles) Reset(end time.Time) {
	for i := range c.candles {
		c.candles[i].BaseAsset = c.Pair.Base
		c.candles[i].QuoteAsset = c.Pair.Quote
		c.candles[i].BaseVolume.Set(math.Zero)
		c.candles[i].QuoteVolume.Set(math.Zero)
		c.candles[i].High.Set(math.Zero)
		c.candles[i].Low.Set(math.Zero)
		c.candles[i].Open.Set(math.Zero)
		c.candles[i].Close.Set(math.Zero)
		c.candles[i].Start = end.Add(-time.Duration(i+1) * c.interval)
		c.candles[i].End = c.candles[i].Start.Add(c.interval)
	}
	c.cutoff = c.candles[0].Start
}

func (c *Candles) shift(n int) {
	end := len(c.candles) - 1
	if n <= 0 || n >= end {
		return
	}
	for i := end; i >= n; i-- {
		candle := &c.candles[i]
		shifted := &c.candles[i-n]
		candle.BaseVolume.Set(&shifted.BaseVolume)
		candle.QuoteVolume.Set(&shifted.QuoteVolume)
		candle.High.Set(&shifted.High)
		candle.Low.Set(&shifted.Low)
		candle.Open.Set(&shifted.Open)
		candle.Close.Set(&shifted.Close)
		candle.Start = shifted.Start
		candle.End = shifted.End
	}
	c.cutoff = c.candles[n].Start.Add(time.Duration(n) * c.interval)
	for i := 0; i < n; i++ {
		candle := &c.candles[i]
		candle.BaseVolume.Set(math.Zero)
		candle.QuoteVolume.Set(math.Zero)
		candle.High.Set(math.Zero)
		candle.Low.Set(math.Zero)
		candle.Open.Set(math.Zero)
		candle.Close.Set(math.Zero)
		candle.Start = c.cutoff.Add(-time.Duration(i) * c.interval)
		candle.End = candle.Start.Add(c.interval)
	}
}

func (c *Candles) Extend(end time.Time) {
	if end.Before(c.candles[0].End) {
		return
	}
	n := int(end.Sub(c.candles[0].End) / c.interval)
	c.shift(n)
}

func (c *Candles) SetTrades(trades []*Trade) error {
	if len(trades) == 0 {
		return nil
	}
	c.cutoff = trades[0].Time
	end := c.candles[0].End
	for _, trade := range trades {
		if trade.Time.After(c.cutoff) {
			return fmt.Errorf("trades list out of order")
		}
		c.cutoff = trade.Time
		i := int(end.Sub(trade.Time) / c.interval)
		if i < 0 {
			continue
		}
		if i >= len(c.candles) {
			break
		}
		candle := &c.candles[i]
		candle.BaseVolume.Add(&candle.BaseVolume, &trade.Base.Amount)
		candle.QuoteVolume.Add(&candle.QuoteVolume, &trade.Quote.Amount)
		candle.Open.Set(trade.Price())
		if candle.Close.Cmp(math.Zero) == 0 {
			candle.Close.Set(&candle.Open)
			candle.High.Set(&candle.Open)
			candle.Low.Set(&candle.Open)
		} else if candle.Open.Cmp(&candle.High) > 0 {
			candle.High.Set(&candle.Open)
		} else if candle.Open.Cmp(&candle.Low) < 0 {
			candle.Low.Set(&candle.Open)
		}
	}
	c.cutoff = c.candles[0].Start
	return nil
}

func (c *Candles) PushTrade(trade *Trade) error {
	if *trade.Pair() != c.Pair {
		return fmt.Errorf("trade pair does not match candle pair")
	}
	if trade.Time.Before(c.candles[0].Start) {
		return fmt.Errorf("trade is too old")
	}
	if trade.Time.After(c.candles[0].End) {
		newStart := trade.Time.Truncate(c.interval)
		c.shift(int(c.candles[0].Start.Sub(newStart) / c.interval))
	}
	if trade.Time.Before(c.cutoff) {
		return fmt.Errorf("trade out of order")
	}
	c.cutoff = trade.Time
	candle := &c.candles[0]
	candle.BaseVolume.Add(&candle.BaseVolume, &trade.Base.Amount)
	candle.QuoteVolume.Add(&candle.QuoteVolume, &trade.Quote.Amount)
	candle.Open.Set(trade.Price())
	if candle.Close.Cmp(math.Zero) == 0 {
		candle.Close.Set(&candle.Open)
		candle.High.Set(&candle.Open)
		candle.Low.Set(&candle.Open)
	} else if candle.Open.Cmp(&candle.High) > 0 {
		candle.High.Set(&candle.Open)
	} else if candle.Open.Cmp(&candle.Low) < 0 {
		candle.Low.Set(&candle.Open)
	}
	return nil
}

func (c *Candles) ListRange(start int, end int) []*Candle {
	if start < 0 || end > len(c.candles) || end < start {
		return []*Candle{}
	}
	candles := make([]*Candle, end-start)
	for i := start; i < end; i++ {
		candles[i-start] = &c.candles[i]
	}
	return candles
}

func (c *Candles) Len() int {
	return len(c.candles)
}

func (c *Candles) Ticker() *Ticker {
	ticker := &Ticker{
		BaseAsset:  c.Pair.Base,
		QuoteAsset: c.Pair.Quote,
		Price:      c.candles[0].Close,
		Time:       c.cutoff,
	}
	start := ticker.Time.Add(-24 * time.Hour)
	for _, candle := range c.candles {
		if candle.Start.Before(start) {
			break
		}
		if ticker.Price.Cmp(math.Zero) == 0 {
			ticker.Price.Set(&candle.Close)
		}
		ticker.BaseVolume.Add(&ticker.BaseVolume, &candle.BaseVolume)
		ticker.QuoteVolume.Add(&ticker.QuoteVolume, &candle.QuoteVolume)
	}
	return ticker
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
	if c.Open.Cmp(math.Zero) != 0 {
		r.Open.Quo(math.One, &c.Open)
	}
	if c.Close.Cmp(math.Zero) != 0 {
		r.Close.Quo(math.One, &c.Close)
	}
	if c.Low.Cmp(math.Zero) != 0 {
		r.High.Quo(math.One, &c.Low)
	}
	if c.High.Cmp(math.Zero) != 0 {
		r.Low.Quo(math.One, &c.High)
	}
	return &r
}
