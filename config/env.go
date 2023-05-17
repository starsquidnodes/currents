package config

import "os"

const (
	EnvExchanges = "EXCHANGES"
	EnvLogLevel = "LOG_LEVEL"
	EnvStoreBackend = "STORE_BACKEND"
	EnvStoreUrl = "STORE_URL"
	EnvInfluxdbToken = "INFLUXDB_TOKEN"
	EnvInfluxdbOrganization = "INFLUXDB_ORGANIZATION"
	EnvOsmosisAssetlistJsonUrl = "OSMOSIS_ASSETLIST_JSON_URL"
	EnvOsmosisAssetlistRefreshInterval = "OSMOSIS_ASSETLIST_REFRESH_INTERVAL"
	EnvOsmosisAssetlistRetryInterval = "OSMOSIS_ASSETLIST_RETRY_INTERVAL"
	EnvTradesMaxAge = "TRADES_MAX_AGE"
	EnvCandlesInterval = "CANDLES_INTERVAL"
	EnvCandlesPeriod = "CANDLES_PERIOD"
)

func EnvConfig() *StringConfig {
	return &StringConfig{
		Exchanges: os.Getenv(EnvExchanges),
		LogLevel: os.Getenv(EnvLogLevel),
		StoreBackend: os.Getenv(EnvStoreBackend),
		StoreUrl: os.Getenv(EnvStoreUrl),
		InfluxdbToken: os.Getenv(EnvInfluxdbToken),
		InfluxdbOrganization: os.Getenv(EnvInfluxdbOrganization),
		OsmosisAssetlistJsonUrl: os.Getenv(EnvOsmosisAssetlistJsonUrl),
		OsmosisAssetlistRefreshInterval: os.Getenv(EnvOsmosisAssetlistRefreshInterval),
		OsmosisAssetlistRetryInterval: os.Getenv(EnvOsmosisAssetlistRetryInterval),
		TradesMaxAge: os.Getenv(EnvTradesMaxAge),
		CandlesInterval: os.Getenv(EnvCandlesInterval),
		CandlesPeriod: os.Getenv(EnvCandlesPeriod),
	}
}