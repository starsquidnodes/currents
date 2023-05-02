package trading

import (
	"time"

	"github.com/ericlagergren/decimal"
)

type Ticker interface {
	BaseAsset() string
	QuoteAsset() string
	Price() decimal.Big
	Time() time.Time
}