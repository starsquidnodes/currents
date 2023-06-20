package config

import "os"

const (
	EnvExchanges                    = "EXCHANGES"
	EnvLogLevel                     = "LOG_LEVEL"
	EnvStoreBackend                 = "STORE_BACKEND"
	EnvStoreUrl                     = "STORE_URL"
	EnvInfluxdbToken                = "INFLUXDB_TOKEN"
	EnvInfluxdbOrganization         = "INFLUXDB_ORGANIZATION"
	EnvOsmosisAssetsJsonUrl         = "OSMOSIS_ASSETS_JSON_URL"
	EnvOsmosisAssetsRefreshInterval = "OSMOSIS_ASSETS_REFRESH_INTERVAL"
	EnvOsmosisAssetsRetryInterval   = "OSMOSIS_ASSETS_RETRY_INTERVAL"
	EnvTradesMaxAge                 = "TRADES_MAX_AGE"
	EnvCandlesInterval              = "CANDLES_INTERVAL"
	EnvCandlesPeriod                = "CANDLES_PERIOD"
)

func EnvConfig() *StringConfig {
	return &StringConfig{
		Exchanges:                    os.Getenv(EnvExchanges),
		LogLevel:                     os.Getenv(EnvLogLevel),
		StoreBackend:                 os.Getenv(EnvStoreBackend),
		StoreUrl:                     os.Getenv(EnvStoreUrl),
		InfluxdbToken:                os.Getenv(EnvInfluxdbToken),
		InfluxdbOrganization:         os.Getenv(EnvInfluxdbOrganization),
		OsmosisAssetsJsonUrl:         os.Getenv(EnvOsmosisAssetsJsonUrl),
		OsmosisAssetsRefreshInterval: os.Getenv(EnvOsmosisAssetsRefreshInterval),
		OsmosisAssetsRetryInterval:   os.Getenv(EnvOsmosisAssetsRetryInterval),
		TradesMaxAge:                 os.Getenv(EnvTradesMaxAge),
		CandlesInterval:              os.Getenv(EnvCandlesInterval),
		CandlesPeriod:                os.Getenv(EnvCandlesPeriod),
	}
}
