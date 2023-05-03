package config

const (
	DefaultLogLevel = "info"
	DefaultStoreBackend = "influxdb2"
	DefaultStoreUrl = "http://localhost:8086"
	DefaultInfluxdbOrganization = "chaindex"
	DefaultOsmosisAssetlistJsonUrl = "https://raw.githubusercontent.com/osmosis-labs/assetlists/main/osmosis-1/osmosis-1.assetlist.json"
	DefaultOsmosisAssetlistRefreshInterval = "15m"
	DefaultOsmosisAssetlistRetryInterval = "30s"
)