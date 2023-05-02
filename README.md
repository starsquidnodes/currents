## Config options
| `ENV_VAR` | Description | Default | Options |
| ------- | ---- | --- | --- |
| `LOG_LEVEL` | Log message filter | info | debug, info, warning, error |
| `OSMOSIS_ASSETLIST_JSON_URL` | URL for `assetlist.json` file | https://raw.githubusercontent.com/osmosis-labs/assetlists/main/osmosis-1/osmosis-1.assetlist.json | |
| `OSMOSIS_ASSETLIST_REFRESH_INTERVAL` | Time to wait between Osmosis asset list updates | 15m | Valid `time.Duration` string |
| `OSMOSIS_ASSETLIST_RETRY_INTERVAL` | Time to wait before retrying a failed Osmosis asset list update | 30s | Valid `time.Duration` string |
