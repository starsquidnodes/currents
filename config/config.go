package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

type (
	StringConfig struct {
		Exchanges                    string
		LogLevel                     string
		StoreBackend                 string
		StoreUrl                     string
		InfluxdbToken                string
		InfluxdbOrganization         string
		OsmosisAssetsJsonUrl         string
		OsmosisAssetsRefreshInterval string
		OsmosisAssetsRetryInterval   string
		TradesMaxAge                 string
		CandlesInterval              string
		CandlesPeriod                string
	}

	StoreConfig struct {
		Url          string `toml:"url"`
		Token        string `toml:"token"`
		Organization string `toml:"org"`
		Path         string `toml:"path"`
	}

	ExchangeConfig struct {
		AssetsUrl             string        `toml:"assets_url"`
		AssetsRefreshInterval time.Duration `toml:"assets_refresh_interval"`
		AssetsRetryInterval   time.Duration `toml:"assets_retry_interval"`
	}

	Config struct {
		Exchanges       []string                  `toml:"exchanges"`
		LogLevel        zerolog.Level             `toml:"log_level"`
		StoreBackend    string                    `toml:"store_backend"`
		StoreConfig     map[string]StoreConfig    `toml:"store"`
		ExchangeConfig  map[string]ExchangeConfig `toml:"exchange"`
		TradesMaxAge    time.Duration             `toml:"trades_max_age"`
		CandlesInterval time.Duration             `toml:"candles_interval"`
		CandlesPeriod   time.Duration             `toml:"candle_period"`
	}
)

var Cfg = InitConfig()

func (sc *StringConfig) Validate() (*Config, error) {
	exchanges := strings.Split(sc.Exchanges, ",")
	logLevel, err := zerolog.ParseLevel(sc.LogLevel)
	if err != nil {
		logLevel = zerolog.InfoLevel
	}

	supportedBackends := map[string]struct{}{
		"influxdb2": {},
		"sqlite":    {},
	}
	_, found := supportedBackends[sc.StoreBackend]
	if !found {
		return nil, fmt.Errorf("invalid store backend")
	}

	exchangeConfig := map[string]ExchangeConfig{}

	for _, exchange := range exchanges {
		switch exchange {
		case "osmosis":
			assetsRefreshInterval, err := time.ParseDuration(sc.OsmosisAssetsRefreshInterval)
			if err != nil {
				return nil, fmt.Errorf("invalid osmosis assetlist refresh interval")
			}
			assetsRetryInterval, err := time.ParseDuration(sc.OsmosisAssetsRetryInterval)
			if err != nil {
				return nil, fmt.Errorf("invalid osmosis assetlist retry interval")
			}

			exchangeConfig[exchange] = ExchangeConfig{
				AssetsRefreshInterval: assetsRefreshInterval,
				AssetsRetryInterval:   assetsRetryInterval,
			}
		}
	}

	tradesMaxAge, err := time.ParseDuration(sc.TradesMaxAge)
	if err != nil {
		return nil, fmt.Errorf("invalid trades max age")
	}
	candlesInterval, err := time.ParseDuration(sc.CandlesInterval)
	if err != nil {
		return nil, fmt.Errorf("invalid candles interval")
	}
	candlesPeriod, err := time.ParseDuration(sc.CandlesPeriod)
	if err != nil {
		return nil, fmt.Errorf("invalid candles period")
	}

	storeConfig := map[string]StoreConfig{
		sc.StoreBackend: {
			Url:          sc.StoreUrl,
			Token:        sc.InfluxdbToken,
			Organization: sc.InfluxdbOrganization,
			Path:         "TODO path",
		},
	}

	return &Config{
		Exchanges:       exchanges,
		LogLevel:        logLevel,
		StoreBackend:    sc.StoreBackend,
		StoreConfig:     storeConfig,
		ExchangeConfig:  exchangeConfig,
		TradesMaxAge:    tradesMaxAge,
		CandlesInterval: candlesInterval,
		CandlesPeriod:   candlesPeriod,
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
	if overlay.OsmosisAssetsJsonUrl != "" {
		base.OsmosisAssetsJsonUrl = overlay.OsmosisAssetsJsonUrl
	}
	if overlay.OsmosisAssetsRefreshInterval != "" {
		base.OsmosisAssetsRefreshInterval = overlay.OsmosisAssetsRefreshInterval
	}
	if overlay.OsmosisAssetsRetryInterval != "" {
		base.OsmosisAssetsRetryInterval = overlay.OsmosisAssetsRetryInterval
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
