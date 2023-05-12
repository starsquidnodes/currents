package main

import (
	"os"
	"strings"
	"time"

	"github.com/mintthemoon/chaindex/api"
	"github.com/mintthemoon/chaindex/config"
	"github.com/mintthemoon/chaindex/exchange"
	"github.com/mintthemoon/chaindex/store"
	"github.com/rs/zerolog"
)

func main() {
	logLevelEnv := os.Getenv(config.EnvLogLevel)
	if logLevelEnv == "" {
		logLevelEnv = config.DefaultLogLevel
	}
	logLevel, err := zerolog.ParseLevel(logLevelEnv)
	if err != nil {
		logLevel = zerolog.InfoLevel
	}
	logger := zerolog.
		New(zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.StampMilli,
		}).
		Level(logLevel).
		With().
		Timestamp().
		Logger()
	storeBackend := os.Getenv(config.EnvStoreBackend)
	if storeBackend == "" {
		storeBackend = config.DefaultStoreBackend
	}
	storeUrl := os.Getenv(config.EnvStoreUrl)
	if storeUrl == "" {
		storeUrl = "http://localhost:8086"
	}
	storeManager, err := store.NewStoreManager(storeBackend, storeUrl, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize database")
	}
	defer storeManager.Close()
	err = storeManager.Health()
	if err != nil {
		logger.Fatal().Err(err).Msg("database health check failed")
	}
	exchangesEnv := os.Getenv(config.EnvExchanges)
	if exchangesEnv == "" {
		exchangesEnv = config.DefaultExchanges
	}
	exchangeNames := strings.Split(exchangesEnv, ",")
	exchanges := make(map[string]exchange.Exchange, len(exchangeNames))
	for _, exchangeName := range exchangeNames {
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
		err = exchange.Subscribe()
		if err != nil {
			logger.Error().Err(err).Msg("failed to subscribe to exchange")
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
