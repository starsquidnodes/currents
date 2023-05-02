package trading

import (
	"time"

	"github.com/ericlagergren/decimal"
)

type Candle interface {
	BaseAsset() string
	QuoteAsset() string
	BaseVolume() decimal.Big
	QuoteVolume() decimal.Big
	High() decimal.Big
	Low() decimal.Big
	Open() decimal.Big
	Close() decimal.Big
	Time() time.Time
}