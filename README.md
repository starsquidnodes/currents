## Config options
| `ENV_VAR` | Description | Default | Options |
| ------- | ---- | --- | --- |
| `LOG_LEVEL` | Log message filter | info | trace, debug, info, warn, error |
| `STORE_BACKEND` | Backend database type | influxdb2 | influxdb2 |
| `STORE_URL` | Backend database URL | http://localhost:8086 | URL |
| `INFLUXDB_TOKEN` | InfluxDB2 auth token | _(required)_ | String (secret) |
| `INFLUXDB_ORGANIZATION` | InfluxDB2 organization | kujira | String |
| `OSMOSIS_ASSETLIST_JSON_URL` | URL for `assetlist.json` file | https://raw.githubusercontent.com/osmosis-labs/assetlists/main/osmosis-1/osmosis-1.assetlist.json | URL |
| `OSMOSIS_ASSETLIST_REFRESH_INTERVAL` | Time to wait between Osmosis asset list updates | 15m | `time.Duration` string |
| `OSMOSIS_ASSETLIST_RETRY_INTERVAL` | Time to wait before retrying a failed Osmosis asset list update | 30s | `time.Duration` string |
