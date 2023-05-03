package exchange

import (
	"fmt"

	"github.com/mintthemoon/chaindex/store"
	"github.com/rs/zerolog"
)

type Exchange interface {
	Name() string
	DisplayName() string
	Subscribe() error
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