package exchange

import (
	"fmt"
	"os"
	"time"

	"github.com/mintthemoon/currents/config"
	"github.com/mintthemoon/currents/store"
	"github.com/mintthemoon/currents/token"
	"github.com/mintthemoon/currents/trading"
	"github.com/rs/zerolog"
)

type (
	ExchangeManager struct {
		Exchanges       map[string]Exchange
		Candles         map[string]map[string][]*trading.Candle
		CandlesInterval time.Duration
		CandlesPeriod   time.Duration
		Tickers         map[string]map[string]*trading.Ticker
		logger          zerolog.Logger
	}

	Exchange interface {
		Name() string
		DisplayName() string
		Subscribe() error
		Pairs() ([]*token.Pair, error)
		Store() store.Store
	}
)

func NewExchangeManager(exchanges map[string]Exchange, logger zerolog.Logger) (*ExchangeManager, error) {
	candlesIntervalEnv := os.Getenv(config.EnvCandlesInterval)
	if candlesIntervalEnv == "" {
		candlesIntervalEnv = config.DefaultCandlesInterval
	}
	candlesInterval, err := time.ParseDuration(candlesIntervalEnv)
	if err != nil {
		return nil, err
	}
	candlesPeriodEnv := os.Getenv(config.EnvCandlesPeriod)
	if candlesPeriodEnv == "" {
		candlesPeriodEnv = config.DefaultCandlesPeriod
	}
	candlesPeriod, err := time.ParseDuration(candlesPeriodEnv)
	if err != nil {
		return nil, err
	}
	candles := make(map[string]map[string][]*trading.Candle, len(exchanges))
	tickers := make(map[string]map[string]*trading.Ticker, len(exchanges))
	for name := range exchanges {
		candles[name] = map[string][]*trading.Candle{}
		tickers[name] = map[string]*trading.Ticker{}
	}
	e := &ExchangeManager{
		Exchanges:       exchanges,
		Candles:         candles,
		CandlesInterval: candlesInterval,
		CandlesPeriod:   candlesPeriod,
		Tickers:         tickers,
		logger:          logger,
	}
	return e, nil
}

func (e *ExchangeManager) Start() {
	go func() {
		for {
			now := time.Now()
			e.FillCandles()
			e.FillTickers()
			time.Sleep(time.Until(now.Add(e.CandlesInterval).Truncate(e.CandlesInterval)))
		}
	}()
}

func (e *ExchangeManager) FillCandles() {
	numCandles := int(e.CandlesPeriod / e.CandlesInterval)
	end := time.Now().UTC().Truncate(e.CandlesInterval)
	start := end.Add(-e.CandlesPeriod)
	for name, exchange := range e.Exchanges {
		pairs, err := exchange.Pairs()
		if err != nil {
			e.logger.Error().
				Err(err).
				Str("exchange", name).
				Msg("failed to get pairs for candle generation")
			continue
		}
		for _, pair := range pairs {
			trades, err := exchange.Store().Trades(pair, start, end)
			if err != nil {
				e.logger.Error().
					Err(err).
					Str("exchange", name).
					Str("pair", pair.String()).
					Msg("failed to get trades for candle generation")
				continue
			}
			candles := trading.CandlesFromTrades(pair, trades, start, end, e.CandlesInterval)
			if len(candles) != numCandles {
				e.logger.Error().
					Str("exchange", name).
					Str("pair", pair.String()).
					Int("expected", numCandles).
					Int("actual", len(candles)).
					Msg("generated an unexpected number of candles")
				continue
			}
			e.Candles[name][pair.String()] = candles
			e.logger.Trace().
				Str("exchange", name).
				Str("pair", pair.String()).
				Int("num_candles", len(candles)).
				Msg("generated candles")
		}
		e.logger.Debug().
			Str("exchange", name).
			Msg("generated candles")
	}
}

func (e *ExchangeManager) FillTickers() {
	end := time.Now().UTC().Truncate(e.CandlesInterval)
	start := end.Add(-24 * time.Hour)
	for name, exchange := range e.Exchanges {
		exchangeCandles, ok := e.Candles[name]
		if !ok {
			e.logger.Error().
				Str("exchange", name).
				Msg("failed to get candles for ticker generation")
			continue
		}
		pairs, err := exchange.Pairs()
		if err != nil {
			e.logger.Error().
				Err(err).
				Str("exchange", name).
				Msg("failed to get pairs for candle generation")
			continue
		}
		for _, pair := range pairs {
			candles, ok := exchangeCandles[pair.String()]
			if !ok {
				e.logger.Error().
					Str("exchange", name).
					Str("pair", pair.String()).
					Msg("failed to get candles for ticker generation")
				continue
			}
			i := 0
			for candles[i].Start.Before(start) {
				i++
			}
			ticker := trading.TickerFromCandles(candles[i:])
			e.Tickers[name][pair.String()] = ticker
			e.logger.Trace().
				Str("exchange", name).
				Str("pair", pair.String()).
				Msg("generated ticker")
		}
		e.logger.Debug().
			Str("exchange", name).
			Msg("generated tickers")
	}
}

func NewExchange(name string, store store.Store, logger zerolog.Logger) (Exchange, error) {
	exchangeLogger := logger.With().Str("exchange", name).Logger()
	switch name {
	case "osmosis":
		return NewOsmosisExchange("https://osmosis-rpc.polkachu.com:443", store, exchangeLogger)
	default:
		return nil, fmt.Errorf("unsupported exchange: %s", name)
	}
}
