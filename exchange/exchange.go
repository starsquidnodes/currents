package exchange

import (
	"fmt"
	"time"

	"indexer/config"
	"indexer/store"
	"indexer/token"
	"indexer/trading"

	"github.com/rs/zerolog"
)

type (
	ExchangeManager struct {
		Exchanges map[string]Exchange
		data      map[string]*ExchangeData
		logger    zerolog.Logger
	}

	Exchange interface {
		Name() string
		DisplayName() string
		Start() error
		Pairs() ([]*token.Pair, error)
		Store() store.Store
		SubscribeTrades() chan *trading.Trade
		SubscribePairs() chan []*token.Pair
	}

	ExchangeData struct {
		pairs   chan []*token.Pair
		trades  chan *trading.Trade
		candles map[string]*trading.Candles
		tickers map[string]*trading.Ticker
		db      store.Store
		logger  zerolog.Logger
	}
)

func NewExchangeManager(exchanges map[string]Exchange, logger zerolog.Logger) (*ExchangeManager, error) {
	e := &ExchangeManager{
		Exchanges: exchanges,
		data:      map[string]*ExchangeData{},
		logger:    logger,
	}
	return e, nil
}

func (e *ExchangeManager) Start() {
	for _, exchange := range e.Exchanges {
		trades := exchange.SubscribeTrades()
		pairs := exchange.SubscribePairs()
		exchangeData := NewExchangeData(pairs, trades, exchange.Store(), e.logger)
		e.data[exchange.Name()] = exchangeData
		exchangeData.Start()
		err := exchange.Start()
		if err != nil {
			e.logger.Error().Err(err).Str("exchange", exchange.Name()).Msg("failed to start exchange")
			continue
		}
	}
}

func (e *ExchangeManager) Candles(exchange string, pair *token.Pair) (*trading.Candles, error) {
	exchangeData, ok := e.data[exchange]
	if !ok {
		return nil, fmt.Errorf("exchange not found")
	}
	return exchangeData.Candles(pair)
}

func (e *ExchangeManager) Tickers(exchange string) ([]*trading.Ticker, error) {
	exchangeData, ok := e.data[exchange]
	if !ok {
		return nil, fmt.Errorf("exchange not found")
	}
	return exchangeData.Tickers()
}

func (e *ExchangeManager) Ticker(exchange string, pair *token.Pair) (*trading.Ticker, error) {
	exchangeData, ok := e.data[exchange]
	if !ok {
		return nil, fmt.Errorf("exchange not found")
	}
	return exchangeData.Ticker(pair)
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

func NewExchangeData(pairs chan []*token.Pair, trades chan *trading.Trade, db store.Store, logger zerolog.Logger) *ExchangeData {
	return &ExchangeData{
		pairs:   pairs,
		trades:  trades,
		candles: map[string]*trading.Candles{},
		tickers: map[string]*trading.Ticker{},
		db:      db,
		logger:  logger,
	}
}

func (e *ExchangeData) Start() {
	go e.SubscribePairs()
	go e.SubscribeTrades()
	go e.FillCandles()
}

func (e *ExchangeData) SubscribePairs() {
	for pairs := range e.pairs {
		e.SetPairs(pairs)
	}
}

func (e *ExchangeData) SubscribeTrades() {
	for trade := range e.trades {
		e.db.SaveTrade(trade)
		pair := trade.Pair()
		candles, ok := e.candles[pair.String()]
		if !ok {
			candles, ok = e.candles[pair.Reversed().String()]
			if !ok {
				e.logger.Error().Str("pair", pair.String()).Msg("pair not found")
				continue
			}
			trade = trade.Reversed()
			pair = pair.Reversed()
		}
		err := candles.PushTrade(trade)
		if err != nil {
			e.logger.Error().
				Err(err).
				Str("pair", pair.String()).
				Time("trade_time", trade.Time).
				Msg("failed to add trade to candles")
			continue
		}
		e.tickers[pair.String()] = candles.Ticker()
	}
}

func (e *ExchangeData) FillCandles() {
	for {
		end := time.Now().UTC().Truncate(config.Cfg.CandlesInterval).Add(config.Cfg.CandlesInterval)
		for symbol, candles := range e.candles {
			candles.Extend(end)
			e.tickers[symbol] = candles.Ticker()
		}
		e.logger.Debug().Time("end", end).Msg("filled candles")
		time.Sleep(time.Until(time.Now().Truncate(config.Cfg.CandlesInterval).Add(config.Cfg.CandlesInterval)))
	}
}

func (e *ExchangeData) SetPairs(pairs []*token.Pair) {
	candlesEnd := time.Now().UTC().Truncate(config.Cfg.CandlesInterval).Add(config.Cfg.CandlesInterval)
	for _, pair := range pairs {
		_, ok := e.candles[pair.String()]
		if !ok {
			candles, err := store.CandlesFromStore(e.db, pair, candlesEnd, config.Cfg.CandlesPeriod, config.Cfg.CandlesInterval)
			if err != nil {
				e.logger.Error().Err(err).Str("pair", pair.String()).Msg("failed to load candles from store")
				continue
			}
			e.candles[pair.String()] = candles
			e.tickers[pair.String()] = candles.Ticker()
			e.logger.Trace().Str("pair", pair.String()).Msg("new pair")
		}
	}
	e.logger.Debug().Int("num_pairs", len(pairs)).Msg("updated pairs")
}

func (e *ExchangeData) Candles(pair *token.Pair) (*trading.Candles, error) {
	candles, ok := e.candles[pair.String()]
	if !ok {
		return nil, fmt.Errorf("candles not found for pair")
	}
	return candles, nil
}

func (e *ExchangeData) Tickers() ([]*trading.Ticker, error) {
	tickers := []*trading.Ticker{}
	for _, ticker := range e.tickers {
		tickers = append(tickers, ticker)
	}
	return tickers, nil
}

func (e *ExchangeData) Ticker(pair *token.Pair) (*trading.Ticker, error) {
	ticker, ok := e.tickers[pair.String()]
	if !ok {
		ticker, ok = e.tickers[pair.Reversed().String()]
		if !ok {
			return nil, fmt.Errorf("ticker not found for pair")
		}
		ticker = ticker.Reversed()
	}
	return ticker, nil
}
