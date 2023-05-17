package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

type (
	StringConfig struct {
		Exchanges string
		LogLevel string
		StoreBackend string
		StoreUrl string
		InfluxdbToken string
		InfluxdbOrganization string
		OsmosisAssetlistJsonUrl string
		OsmosisAssetlistRefreshInterval string
		OsmosisAssetlistRetryInterval string
		TradesMaxAge string
		CandlesInterval string
		CandlesPeriod string
	}

	Config struct {
		Exchanges []string
		LogLevel zerolog.Level
		StoreBackend string
		StoreUrl string
		InfluxdbToken string
		InfluxdbOrganization string
		OsmosisAssetlistJsonUrl string
		OsmosisAssetlistRefreshInterval time.Duration
		OsmosisAssetlistRetryInterval time.Duration
		TradesMaxAge time.Duration
		CandlesInterval time.Duration
		CandlesPeriod time.Duration
	}
)

var Cfg = InitConfig()

func (s *StringConfig) Validate() (*Config, error) {
	exchanges := strings.Split(s.Exchanges, ",")
	logLevel, err := zerolog.ParseLevel(s.LogLevel)
	if err != nil {
		logLevel = zerolog.InfoLevel
	}
	if s.StoreBackend != "influxdb2" {
		return nil, fmt.Errorf("invalid store backend")
	}
	osmosisAssetlistRefreshInterval, err := time.ParseDuration(s.OsmosisAssetlistRefreshInterval)
	if err != nil {
		return nil, fmt.Errorf("invalid osmosis assetlist refresh interval")
	}
	osmosisAssetlistRetryInterval, err := time.ParseDuration(s.OsmosisAssetlistRetryInterval)
	if err != nil {
		return nil, fmt.Errorf("invalid osmosis assetlist retry interval")
	}
	tradesMaxAge, err := time.ParseDuration(s.TradesMaxAge)
	if err != nil {
		return nil, fmt.Errorf("invalid trades max age")
	}
	candlesInterval, err := time.ParseDuration(s.CandlesInterval)
	if err != nil {
		return nil, fmt.Errorf("invalid candles interval")
	}
	candlesPeriod, err := time.ParseDuration(s.CandlesPeriod)
	if err != nil {
		return nil, fmt.Errorf("invalid candles period")
	}
	return &Config{
		Exchanges: exchanges,
		LogLevel: logLevel,
		StoreBackend: s.StoreBackend,
		StoreUrl: s.StoreUrl,
		InfluxdbToken: s.InfluxdbToken,
		InfluxdbOrganization: s.InfluxdbOrganization,
		OsmosisAssetlistJsonUrl: s.OsmosisAssetlistJsonUrl,
		OsmosisAssetlistRefreshInterval: osmosisAssetlistRefreshInterval,
		OsmosisAssetlistRetryInterval: osmosisAssetlistRetryInterval,
		TradesMaxAge: tradesMaxAge,
		CandlesInterval: candlesInterval,
		CandlesPeriod: candlesPeriod,
	}, nil
}

func (s *StringConfig) MustValidate() *Config {
	cfg, err := s.Validate()
	if err != nil {
		panic(fmt.Errorf("config validation error: %s", err))
	}
	return cfg
}

func InitConfig() *Config {
	return MergeConfig(DefaultConfig(), EnvConfig()).MustValidate()
}

func MergeConfig(base *StringConfig, overlay *StringConfig) *StringConfig {
	if overlay.Exchanges != "" {
		base.Exchanges = overlay.Exchanges
	}
	if overlay.LogLevel != "" {
		base.LogLevel = overlay.LogLevel
	}
	if overlay.StoreBackend != "" {
		base.StoreBackend = overlay.StoreBackend
	}
	if overlay.StoreUrl != "" {
		base.StoreUrl = overlay.StoreUrl
	}
	if overlay.InfluxdbToken != "" {
		base.InfluxdbToken = overlay.InfluxdbToken
	}
	if overlay.InfluxdbOrganization != "" {
		base.InfluxdbOrganization = overlay.InfluxdbOrganization
	}
	if overlay.OsmosisAssetlistJsonUrl != "" {
		base.OsmosisAssetlistJsonUrl = overlay.OsmosisAssetlistJsonUrl
	}
	if overlay.OsmosisAssetlistRefreshInterval != "" {
		base.OsmosisAssetlistRefreshInterval = overlay.OsmosisAssetlistRefreshInterval
	}
	if overlay.OsmosisAssetlistRetryInterval != "" {
		base.OsmosisAssetlistRetryInterval = overlay.OsmosisAssetlistRetryInterval
	}
	if overlay.TradesMaxAge != "" {
		base.TradesMaxAge = overlay.TradesMaxAge
	}
	if overlay.CandlesInterval != "" {
		base.CandlesInterval = overlay.CandlesInterval
	}
	if overlay.CandlesPeriod != "" {
		base.CandlesPeriod = overlay.CandlesPeriod
	}
	return base
}