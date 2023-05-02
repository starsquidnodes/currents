package trading

import (
	"time"
	
	"github.com/ericlagergren/decimal"
)

type Trade interface {
	BaseAmount() decimal.Big
	QuoteAmount() decimal.Big
	BaseAsset() string
	QuoteAsset() string
	Time() time.Time
}