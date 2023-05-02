package chain

import (
	"github.com/rs/zerolog"
)

type OsmosisRpc struct {
	rpc *CometRpc
	height int64
	logger zerolog.Logger
}

func NewOsmosisRpc(url string, logger zerolog.Logger) (*OsmosisRpc, error) {
	chainLogger := logger.With().Str("chain", "osmosis").Logger()
	rpc, err := NewCometRpc(url, chainLogger)
	if err != nil {
		return nil, err
	}
	o := &OsmosisRpc{
		rpc: rpc,
		logger: chainLogger,
	}
	o.logger.Info().Msg("chain connected")
	return o, nil
}

func (o *OsmosisRpc) Subscribe() error {
	channel, err := o.rpc.Subscribe("tm.event='Tx' AND token_swapped.module='gamm'")
	if err != nil {
		return err
	}
	o.logger.Info().Msg("subscribed to swap events")
	for {
		event := <-channel
		tokensIn, ok := event.Events["token_swapped.tokens_in"]
		if !ok {
			o.logger.Warn().Msg("swap event missing tokens_in")
			continue
		}
		tokensOut, ok := event.Events["token_swapped.tokens_out"]
		if !ok {
			o.logger.Warn().Msg("swap event missing tokens_out")
			continue
		}
		if len(tokensIn) != len(tokensOut) {
			o.logger.Warn().Msg("swap event tokens_in and tokens_out length mismatch")
			continue
		}
		for i, tokenIn := range tokensIn {
			o.logger.Info().Str("in", tokenIn).Str("out", tokensOut[i]).Msg("trade")
		}
	}
}
