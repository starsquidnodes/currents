package main

import (
	"flag"
	"fmt"
	"indexer/config"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/rs/zerolog"
)

func main() {
	var (
		logLevel   string
		logFormat  string
		configFile string
	)

	logger := zerolog.
		New(zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.StampMilli,
		}).
		Level(zerolog.DebugLevel).
		With().
		Timestamp().
		Logger()

	flag.StringVar(&logLevel, "log-level", "INFO", "logging level")
	flag.StringVar(&logFormat, "log-format", "text", "logging format; must be either json or text")
	flag.StringVar(&configFile, "config-file", "", "config file")

	flag.Parse()

	var cfg = config.Cfg

	if configFile != "" {
		_, err := toml.DecodeFile(configFile, cfg)
		if err != nil {
			logger.Fatal().Err(err)
		}
	}

	fmt.Println(cfg)

	logger.Info().Msg("Yay")
}
