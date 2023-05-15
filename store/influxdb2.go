package store

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	influxdb2api "github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/mintthemoon/currents/config"
	"github.com/mintthemoon/currents/token"
	"github.com/mintthemoon/currents/trading"
	"github.com/rs/zerolog"
)

type (
	Influxdb2Manager struct {
		client influxdb2.Client
		url    string
		stores map[string]*Influxdb2Store
		logger zerolog.Logger
	}

	Influxdb2Store struct {
		name   string
		writer influxdb2api.WriteAPI
		reader influxdb2api.QueryAPI
		logger zerolog.Logger
	}
)

func NewInfluxdb2Manager(url string, logger zerolog.Logger) (*Influxdb2Manager, error) {
	influxLogger := logger.With().Str("backend", "influxdb2").Logger()
	influxdbToken := os.Getenv(config.EnvInfluxdbToken)
	if influxdbToken == "" {
		influxLogger.Error().Str("env", config.EnvInfluxdbToken).Msg("missing required config variable")
		return nil, fmt.Errorf("missing influxdb2 auth token")
	}
	client := influxdb2.NewClientWithOptions(
		url,
		influxdbToken,
		influxdb2.DefaultOptions().
			SetBatchSize(5).
			SetFlushInterval(250).
			SetRetryInterval(500).
			SetMaxRetryInterval(2500),
	)
	i := &Influxdb2Manager{
		client: client,
		url:    url,
		stores: map[string]*Influxdb2Store{},
		logger: influxLogger,
	}
	return i, nil
}

func (i *Influxdb2Manager) Store(name string) (Store, error) {
	store, ok := i.stores[name]
	var err error
	if !ok {
		store, err = NewInfluxdb2Store(name, i.client, i.logger)
		if err != nil {
			return nil, err
		}
		i.stores[name] = store
	}
	return store, nil
}

func (i *Influxdb2Manager) Health() error {
	health, err := i.client.Health(context.Background())
	if err != nil {
		i.logger.Debug().Err(err).Msg("database health check failed (expected if using influxdb cloud)")
		pinged, err := i.client.Ping(context.Background())
		if err != nil {
			i.logger.Error().Err(err).Msg("database ping failed")
			return err
		}
		if !pinged {
			i.logger.Error().Msg("database ping failed")
			return fmt.Errorf("influxdb2 ping check failed")
		}
		i.logger.Info().Str("url", i.url).Msg("database connected")
		return nil
	}
	if health.Status != "pass" {
		i.logger.Error().
			Err(err).
			Str("name", health.Name).
			Str("status", string(health.Status)).
			Str("version", *health.Version).
			Str("commit", *health.Commit).
			Msgf("database %s", *health.Message)
		return fmt.Errorf("influxdb2 health check failed")
	}
	i.logger.Info().
		Str("name", health.Name).
		Str("status", string(health.Status)).
		Str("version", *health.Version).
		Str("commit", *health.Commit).
		Msgf("database %s", *health.Message)
	return nil
}

func (i *Influxdb2Manager) Close() {
	i.client.Close()
}

func NewInfluxdb2Store(name string, client influxdb2.Client, logger zerolog.Logger) (*Influxdb2Store, error) {
	storeLogger := logger.With().Str("store", name).Logger()
	organization := os.Getenv(config.EnvInfluxdbOrganization)
	if organization == "" {
		organization = config.DefaultInfluxdbOrganization
	}
	writer := client.WriteAPI(organization, name)
	reader := client.QueryAPI(organization)
	errorsChannel := writer.Errors()
	go func() {
		for err := range errorsChannel {
			storeLogger.Error().Err(err).Msg("database write error")
		}
	}()
	storeLogger.Debug().Msg("new store client")
	s := &Influxdb2Store{
		name:   name,
		writer: writer,
		reader: reader,
		logger: storeLogger,
	}
	return s, nil
}

func (i *Influxdb2Store) Name() string {
	return i.name
}

func (s *Influxdb2Store) SaveTrade(trade *trading.Trade) error {
	id, err := uuid.NewRandom()
	if err != nil {
		return err
	}
	p := influxdb2.NewPoint(
		"trade",
		map[string]string{
			"base_asset":  trade.Base.Symbol,
			"quote_asset": trade.Quote.Symbol,
			"id":          id.String(), // ensures trades have unique tags
		},
		map[string]interface{}{
			"base_volume":  trade.Base.Amount.String(),
			"quote_volume": trade.Quote.Amount.String(),
		},
		trade.Time,
	)
	s.writer.WritePoint(p)
	s.logger.Trace().Str("base", trade.Base.Symbol).Str("quote", trade.Quote.Symbol).Msg("saving trade")
	return nil
}

func (s *Influxdb2Store) Trades(pair *token.Pair, start time.Time, end time.Time) ([]*trading.Trade, error) {
	fluxQuery := fmt.Sprintf(
		`from(bucket: "%s")
			|> range(start: %s, stop: %s)
			|> filter(fn: (r) => r._measurement == "trade" and ((r.base_asset == "%s" and r.quote_asset == "%s") or (r.base_asset == "%s" and r.quote_asset == "%s")))
			|> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")
			|> group()
			|> sort(columns: ["_time"], desc: true)
			|> yield(name: "trade")
		`,
		s.name,
		start.Format(time.RFC3339),
		end.Format(time.RFC3339),
		pair.Base,
		pair.Quote,
		pair.Quote,
		pair.Base,
	)
	res, err := s.reader.Query(context.Background(), fluxQuery)
	if err != nil {
		s.logger.Error().Err(err).Msg("database query error")
		return nil, err
	}
	trades := []*trading.Trade{}
	for res.Next() {
		tradeBaseSymbol := fmt.Sprintf("%v", res.Record().ValueByKey("base_asset"))
		tradeBaseVolume := fmt.Sprintf("%v", res.Record().ValueByKey("base_volume"))
		tradeQuoteSymbol := fmt.Sprintf("%v", res.Record().ValueByKey("quote_asset"))
		tradeQuoteVolume := fmt.Sprintf("%v", res.Record().ValueByKey("quote_volume"))
		if tradeBaseSymbol != pair.Base {
			if tradeQuoteSymbol != pair.Base {
				s.logger.Error().
					Str("trade_base", tradeBaseSymbol).
					Str("trade_quote", tradeQuoteSymbol).
					Str("query_base", pair.Base).
					Str("query_quote", pair.Quote).
					Msg("unexpected symbol in query result")
				continue
			}
			tmpSymbol := tradeBaseSymbol
			tmpVolume := tradeBaseVolume
			tradeBaseSymbol = tradeQuoteSymbol
			tradeBaseVolume = tradeQuoteVolume
			tradeQuoteSymbol = tmpSymbol
			tradeQuoteVolume = tmpVolume
		}
		base, err := token.ParseToken(fmt.Sprintf("%s%s", tradeBaseVolume, tradeBaseSymbol))
		if err != nil {
			s.logger.Error().
				Err(err).
				Str("symbol", tradeBaseSymbol).
				Str("volume", tradeBaseVolume).
				Msg("failed to parse trade base token")
			continue
		}
		quote, err := token.ParseToken(fmt.Sprintf("%s%s", tradeQuoteVolume, tradeQuoteSymbol))
		if err != nil {
			s.logger.Error().
				Err(err).
				Str("symbol", tradeQuoteSymbol).
				Str("volume", tradeQuoteVolume).
				Msg("failed to parse trade quote token")
			continue
		}
		if res.Err() != nil {
			s.logger.Error().Err(res.Err()).Msg("database query error")
			continue
		}
		trade := &trading.Trade{
			Base:  *base,
			Quote: *quote,
			Time:  res.Record().Time().UTC(),
		}
		trades = append(trades, trade)
	}
	return trades, nil
}
