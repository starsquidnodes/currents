package main

import (
	"os"
	"time"

	"github.com/mintthemoon/chaindex/chain"
	"github.com/mintthemoon/chaindex/config"
	"github.com/mintthemoon/chaindex/store"
	"github.com/rs/zerolog"
)

func main() {
	logLevelEnv := os.Getenv("LOG_LEVEL")
	if logLevelEnv == "" {
		logLevelEnv = "info"
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
	o, err := chain.NewOsmosisRpc("https://osmosis-rpc.polkachu.com:443", osmosisStore, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create osmosis client")
	}
	err = o.Subscribe()
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to subscribe osmosis client")
	}
	for {
		time.Sleep(10 * time.Second)
		trades, err := osmosisStore.Trades("OSMO", "USDC", time.Now().Add(10 * -time.Minute), time.Now())
		if err != nil {
			logger.Fatal().Err(err).Msg("failed to get trades")
		}
		for _, trade := range trades {
			logger.Info().Str("base", trade.Base.Symbol).Str("quote", trade.Quote.Symbol).Str("price", trade.Price().String()).Msg("trade")
		}
	}
}
