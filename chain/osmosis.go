package chain

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/osmosis-labs/assetlist"
	"github.com/rs/zerolog"
)

type (
	OsmosisRpc struct {
		rpc *CometRpc
		assets map[string]assetlist.Asset
		logger zerolog.Logger
	}

	OsmosisTokenSwap struct {
		In Token
		Out Token
		Pool string
	}

	OsmosisTrade struct {
		Base Token
		Quote Token
	}
)

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
	err = o.PollAssetList()
	return o, err
}

func (o *OsmosisRpc) Subscribe() error {
	channel, err := o.rpc.Subscribe("tm.event='Tx' AND token_swapped.module='gamm'")
	if err != nil {
		return err
	}
	o.logger.Info().Msg("subscribed to swap events")
	for {
		event := <-channel
		trades := o.GetTrades(&event)
		for _, trade := range trades {
			o.logger.Info().Str("base", trade.Base.String()).Str("quote", trade.Quote.String()).Msg("trade")
		}
	}
}

func (o *OsmosisRpc) GetTrades(event *coretypes.ResultEvent) []OsmosisTrade {
	trades := []OsmosisTrade{}
	swaps, err := ParseOsmosisTokenSwaps(event)
	if err != nil {
		o.logger.Error().Err(err).Msg("failed to parse swap event")
		return trades
	}
	if len(o.assets) == 0 {
		o.logger.Warn().Msg("cannot process trades when asset list is empty")
		return trades
	}
	for _, swap := range swaps {
		inAsset, ok := o.assets[swap.In.Symbol]
		if !ok {
			o.logger.Debug().Str("symbol", swap.In.Symbol).Msg("skipping unlisted asset swap")
			continue
		}
		outAsset, ok := o.assets[swap.Out.Symbol]
		if !ok {
			o.logger.Debug().Str("symbol", swap.Out.Symbol).Msg("skipping unlisted asset swap")
			continue
		}
		base, err := swap.In.RebaseOsmosisAsset(inAsset)
		if err != nil {
			o.logger.Warn().Err(err).Str("symbol", swap.In.Symbol).Msg("failed to rebase in token")
			continue
		}
		quote, err := swap.Out.RebaseOsmosisAsset(outAsset)
		if err != nil {
			o.logger.Warn().Err(err).Str("symbol", swap.Out.Symbol).Msg("failed to rebase out token")
			continue
		}
		trades = append(trades, OsmosisTrade{Base: base, Quote: quote})
	}
	return trades
}

func (o *OsmosisRpc) PollAssetList() error {
	url := os.Getenv("OSMOSIS_ASSETLIST_JSON_URL")
	if url == "" {
		url = "https://raw.githubusercontent.com/osmosis-labs/assetlists/main/osmosis-1/osmosis-1.assetlist.json"
	}
	refreshIntervalStr := os.Getenv("OSMOSIS_ASSETLIST_REFRESH_INTERVAL")
	if refreshIntervalStr == "" {
		refreshIntervalStr = "15m"
	}
	refreshInterval, err := time.ParseDuration(refreshIntervalStr)
	if err != nil {
		o.logger.Error().Err(err).Msg("failed to parse asset list refresh interval")
		return err
	}
	retryIntervalStr := os.Getenv("OSMOSIS_ASSETLIST_RETRY_INTERVAL")
	if retryIntervalStr == "" {
		retryIntervalStr = "30s"
	}
	retryInterval, err := time.ParseDuration(retryIntervalStr)
	if err != nil {
		o.logger.Error().Err(err).Msg("failed to parse asset list retry interval")
		return err
	}
	go func() {
		for {
			assetList, err := LoadOsmosisAssetList(url)
			if err != nil {
				o.logger.Error().Err(err).Msg("failed to load asset list")
				time.Sleep(retryInterval)
				continue
			}
			assets := make(map[string]assetlist.Asset, len(assetList.Assets))
			for _, asset := range assetList.Assets {
				assets[asset.Base] = asset
			}
			o.assets = assets
			o.logger.Debug().Int("num_assets", len(o.assets)).Msg("refreshed asset list")
			time.Sleep(refreshInterval)
		}
	}()
	return nil
}

func (t *Token) RebaseOsmosisAsset(asset assetlist.Asset) (Token, error) {
	exponents := make(map[string]int64, len(asset.DenomUnits))
	for _, denomUnit := range asset.DenomUnits {
		exponents[denomUnit.Denom] = denomUnit.Exponent
	}
	displayExponent, ok := exponents[asset.Display]
	if !ok {
		return Token{}, fmt.Errorf("could not determine display units for %s", t.Symbol)
	}
	exponent := t.Amount.Scale() + int(displayExponent)
	return t.Rebase(exponent, asset.Symbol), nil
}

func ParseOsmosisTokenSwaps(event *coretypes.ResultEvent) ([]OsmosisTokenSwap, error) {
	tokenSwapModule, ok := event.Events["token_swapped.module"]
	if !ok {
		return []OsmosisTokenSwap{}, nil
	}
	tokenSwapPool, ok := event.Events["token_swapped.pool_id"]
	if !ok {
		return nil, fmt.Errorf("swap event missing pool_id")
	}
	tokensIn, ok := event.Events["token_swapped.tokens_in"]
	if !ok {
		return nil, fmt.Errorf("swap event missing tokens_in")
	}
	tokensOut, ok := event.Events["token_swapped.tokens_out"]
	if !ok {
		return nil, fmt.Errorf("swap event missing tokens_out")
	}
	numSwaps := len(tokenSwapModule)
	if len(tokenSwapPool) != numSwaps || len(tokensIn) != numSwaps || len(tokensOut) != numSwaps {
		return nil, fmt.Errorf("swap event attributes length mismatch")
	}
	swaps := make([]OsmosisTokenSwap, len(tokensIn))
	for i, module := range tokenSwapModule {
		if module != "gamm" {
			continue
		}
		in, err := ParseToken(tokensIn[i])
		if err != nil {
			return nil, fmt.Errorf("failed to parse input token '%s': %v", tokensIn[i], err)
		}
		out, err := ParseToken(tokensOut[i])
		if err != nil {
			return nil, fmt.Errorf("failed to parse output token '%s': %v", tokensOut[i], err)
		}
		swaps[i] = OsmosisTokenSwap{
			In: in,
			Out: out,
			Pool: tokenSwapPool[i],
		}
	}
	return swaps, nil
}

func LoadOsmosisAssetList(url string) (*assetlist.Root, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	root := &assetlist.Root{}
	err = json.NewDecoder(res.Body).Decode(root)
	return root, err
}
