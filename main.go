package main

import (
	"os"
	"time"

	"github.com/mintthemoon/currents/api"
	"github.com/mintthemoon/currents/config"
	"github.com/mintthemoon/currents/exchange"
	"github.com/mintthemoon/currents/store"
	"github.com/rs/zerolog"
)

func main() {
	logger := zerolog.
		New(zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.StampMilli,
		}).
		Level(config.Cfg.LogLevel).
		With().
		Timestamp().
		Logger()
	logger.Trace().
		Any("exchanges", config.Cfg.Exchanges).
		Str("log_level", config.Cfg.LogLevel.String()).
		Str("store_backend", config.Cfg.StoreBackend).
		Str("store_url", config.Cfg.StoreUrl).
		Str("influxdb_token", config.Cfg.InfluxdbToken).
		Str("influxdb_organization", config.Cfg.InfluxdbOrganization).
		Str("osmosis_assetlist_json_url", config.Cfg.OsmosisAssetlistJsonUrl).
		Dur("osmosis_assetlist_refresh_interval", config.Cfg.OsmosisAssetlistRefreshInterval).
		Dur("osmosis_assetlist_retry_interval", config.Cfg.OsmosisAssetlistRetryInterval).
		Dur("trades_max_age", config.Cfg.TradesMaxAge).
		Dur("candles_interval", config.Cfg.CandlesInterval).
		Dur("candles_period", config.Cfg.CandlesPeriod).
		Msg("config")
	storeManager, err := store.NewStoreManager(config.Cfg.StoreBackend, config.Cfg.StoreUrl, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize database")
	}
	defer storeManager.Close()
	err = storeManager.Health()
	if err != nil {
		logger.Fatal().Err(err).Msg("database health check failed")
	}
	exchanges := make(map[string]exchange.Exchange, len(config.Cfg.Exchanges))
	for _, exchangeName := range config.Cfg.Exchanges {
		store, err := storeManager.Store(exchangeName)
		if err != nil {
			logger.Error().Err(err).Str("exchange", exchangeName).Msg("failed to initialize exchange store")
			continue
		}
		exchange, err := exchange.NewExchange(exchangeName, store, logger)
		if err != nil {
			logger.Error().Err(err).Str("exchange", exchangeName).Msg("failed to initialize exchange")
			continue
		}
		exchanges[exchangeName] = exchange
	}
	exchangeManager, err := exchange.NewExchangeManager(exchanges, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize exchange manager")
	}
	exchangeManager.Start()
	api := api.NewApi(exchanges, exchangeManager, storeManager, logger)
	api.Start()
}
