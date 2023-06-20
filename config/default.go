package config

func DefaultConfig() *StringConfig {
	return &StringConfig{
		Exchanges:                    "osmosis",
		LogLevel:                     "info",
		StoreBackend:                 "influxdb2",
		StoreUrl:                     "http://localhost:8086",
		InfluxdbOrganization:         "currents",
		OsmosisAssetsJsonUrl:         "https://raw.githubusercontent.com/osmosis-labs/assetlists/main/osmosis-1/osmosis-1.assetlist.json",
		OsmosisAssetsRefreshInterval: "15m",
		OsmosisAssetsRetryInterval:   "30s",
		TradesMaxAge:                 "48h",
		CandlesInterval:              "1m",
		CandlesPeriod:                "48h",
	}
}
