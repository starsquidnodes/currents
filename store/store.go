package store

import (
	"fmt"
	"time"

	"github.com/mintthemoon/chaindex/trading"
	"github.com/rs/zerolog"
)

type (
	StoreManager interface {
		Store(name string) (Store, error)
		Health() error
		Close()
	}

	Store interface {
		SaveTrade(trading.Trade) error
		SaveCandle(trading.Candle) error
		SaveTicker(trading.Ticker) error
		Trades(base string, quote string, start time.Time, end time.Time) ([]trading.BasicTrade, error)
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
