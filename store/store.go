package store

import (
	"fmt"
	"time"

	"github.com/mintthemoon/currents/token"
	"github.com/mintthemoon/currents/trading"
	"github.com/rs/zerolog"
)

type (
	StoreManager interface {
		Store(name string) (Store, error)
		Health() error
		Close()
	}

	Store interface {
		Name() string
		SaveTrade(*trading.Trade) error
		Trades(pair *token.Pair, start time.Time, end time.Time) ([]*trading.Trade, error)
	}
)

func NewStoreManager(backend string, url string, logger zerolog.Logger) (StoreManager, error) {
	switch backend {
	case "influxdb2":
		return NewInfluxdb2Manager(url, logger)
	default:
		return nil, fmt.Errorf("unsupported store backend: %s", backend)
	}
}

func CandlesFromStore(s Store, pair *token.Pair, end time.Time, period time.Duration, interval time.Duration) (*trading.Candles, error) {
	start := end.Add(-period)
	trades, err := s.Trades(pair, start, end)
	if err != nil {
		return nil, err
	}
	return trading.NewCandles(pair, trades, interval, period, end)
}