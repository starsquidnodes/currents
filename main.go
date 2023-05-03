package main

import (
	"os"
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
	osmosisStore, err := storeManager.Store("osmosis")
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize osmosis store")
	}
	exchanges := map[string]exchange.Exchange{}
	exchanges["osmosis"], err = exchange.NewExchange("osmosis", osmosisStore, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize osmosis exchange")
	}
	for _, exchange := range exchanges {
		err = exchange.Subscribe()
		if err != nil {
			logger.Fatal().Err(err).Msg("failed to subscribe to exchange")
		}
	}
	api := api.NewApi(exchanges, logger)
	api.Start()
}
