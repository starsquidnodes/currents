package store

import (
	"fmt"
	"time"

	"github.com/mintthemoon/chaindex/trading"
	"github.com/mintthemoon/chaindex/token"
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
		Trades(pairs *token.Pair, start time.Time, end time.Time) ([]*trading.Trade, error)
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
