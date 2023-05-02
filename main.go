package main

import (
	"context"
	"os"
	"time"

    "github.com/influxdata/influxdb-client-go/v2"
	"github.com/rs/zerolog"
	"github.com/mintthemoon/chaindex/chain"
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
	client := influxdb2.NewClientWithOptions(
		"http://localhost:8086",
		"hlPH0W73wF_ATePRZ2ifWwnPuHRJMkOJiOLH5Y-6r0C7uZjzY2nP0tG8MyNplje1PK-9E5CKyFHhFztEYoE31A==",
		influxdb2.DefaultOptions().
			SetBatchSize(5).
			SetFlushInterval(250).
			SetRetryInterval(500).
			SetMaxRetryInterval(2500),
	)
	defer client.Close()
	ctx := context.Background()
	health, err := client.Health(ctx)
	if err != nil {
		panic(err)
	}
	logger.Info().
		Str("name", health.Name).
		Str("status", string(health.Status)).
		Str("version", *health.Version).
		Str("commit", *health.Commit).
		Msgf("database %s", *health.Message)
	writeApi := client.WriteAPI("kujira", "test")
	defer writeApi.Flush()
	errorsChannel := writeApi.Errors()
	go func() {
		for err := range errorsChannel {
			logger.Error().Err(err).Msg("database write error")
		}
	}()
	o, err := chain.NewOsmosisRpc("https://osmosis-rpc.polkachu.com:443", logger)
	if err != nil {
		panic(err)
	}
	err = o.Subscribe()
	if err != nil {
		panic(err)
	}
}