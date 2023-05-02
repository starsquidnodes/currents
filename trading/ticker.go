package trading

import (
	"time"

	"github.com/ericlagergren/decimal"
)

type Ticker interface {
	BaseAsset() string
	QuoteAsset() string
	BaseVolume() decimal.Big
	QuoteVolume() decimal.Big
	Price() decimal.Big
	Time() time.Time
}