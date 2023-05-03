package exchange

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/mintthemoon/chaindex/chain"
	"github.com/mintthemoon/chaindex/config"
	"github.com/mintthemoon/chaindex/store"
	"github.com/mintthemoon/chaindex/token"
	"github.com/mintthemoon/chaindex/trading"
	"github.com/osmosis-labs/assetlist"
	"github.com/rs/zerolog"
)

type (
	OsmosisExchange struct {
		rpc *chain.CometRpc
		assets map[string]assetlist.Asset
		store store.Store
		logger zerolog.Logger
	}

	OsmosisTokenSwap struct {
		In token.Token
		Out token.Token
		Pool string
	}
)

func NewOsmosisExchange(url string, store store.Store, logger zerolog.Logger) (*OsmosisExchange, error) {
	rpc, err := chain.NewCometRpc(url, logger)
	if err != nil {
		return nil, err
	}
	o := &OsmosisExchange{
		rpc: rpc,
		store: store,
		logger: logger,
	}
	o.logger.Info().Msg("chain connected")
	err = o.PollAssetList()
	return o, err
}

func (o *OsmosisExchange) Name() string {
	return "osmosis"
}

func (o *OsmosisExchange) DisplayName() string {
	return "Osmosis"
}

func (o *OsmosisExchange) Subscribe() error {
	channel, err := o.rpc.Subscribe("tm.event='Tx' AND token_swapped.module='gamm'")
	if err != nil {
		return err
	}
	o.logger.Info().Msg("subscribed to swap events")
	go func() {
		for {
			event := <-channel
			trades := o.GetTrades(&event)
			for _, trade := range trades {
				o.logger.Debug().Str("base", trade.Base.String()).Str("quote", trade.Quote.String()).Msg("trade")
				o.store.SaveTrade(&trade)
			}
		}
	}()
	return nil
}

func (o *OsmosisExchange) GetTrades(event *coretypes.ResultEvent) []trading.BasicTrade {
	trades := []trading.BasicTrade{}
	swaps, err := ParseOsmosisTokenSwaps(event)
	if err != nil {
		o.logger.Error().Err(err).Msg("failed to parse swap event")
		return trades
	}
	if len(o.assets) == 0 {
		o.logger.Warn().Msg("cannot process trades when asset list is empty")
		return trades
	}
	now := time.Now()
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
		base, err := RebaseOsmosisAsset(&swap.In, inAsset)
		if err != nil {
			o.logger.Debug().Err(err).Str("symbol", swap.In.Symbol).Msg("failed to rebase in token")
			continue
		}
		quote, err := RebaseOsmosisAsset(&swap.Out, outAsset)
		if err != nil {
			o.logger.Debug().Err(err).Str("symbol", swap.Out.Symbol).Msg("failed to rebase out token")
			continue
		}
		trades = append(trades, trading.BasicTrade{Base: base, Quote: quote, Time: now})
	}
	return trades
}

func (o *OsmosisExchange) PollAssetList() error {
	url := os.Getenv("OSMOSIS_ASSETLIST_JSON_URL")
	if url == "" {
		url = config.DefaultOsmosisAssetlistJsonUrl
	}
	refreshIntervalStr := os.Getenv(config.EnvOsmosisAssetlistRefreshInterval)
	if refreshIntervalStr == "" {
		refreshIntervalStr = config.DefaultOsmosisAssetlistRefreshInterval
	}
	refreshInterval, err := time.ParseDuration(refreshIntervalStr)
	if err != nil {
		o.logger.Error().Err(err).Msg("failed to parse asset list refresh interval")
		return err
	}
	retryIntervalStr := os.Getenv(config.EnvOsmosisAssetlistRetryInterval)
	if retryIntervalStr == "" {
		retryIntervalStr = config.DefaultOsmosisAssetlistRetryInterval
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
				o.logger.Error().Err(err).Str("url", url).Msg("failed to load asset list")
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

func RebaseOsmosisAsset(t *token.Token, asset assetlist.Asset) (token.Token, error) {
	exponents := make(map[string]int64, len(asset.DenomUnits))
	for _, denomUnit := range asset.DenomUnits {
		exponents[denomUnit.Denom] = denomUnit.Exponent
	}
	displayExponent, ok := exponents[asset.Display]
	if !ok {
		return token.Token{}, fmt.Errorf("could not determine display units for %s", t.Symbol)
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
		in, err := token.ParseToken(tokensIn[i])
		if err != nil {
			return nil, fmt.Errorf("failed to parse input token '%s': %v", tokensIn[i], err)
		}
		out, err := token.ParseToken(tokensOut[i])
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
	if err == nil && len(root.Assets) == 0 {
		err = fmt.Errorf("asset list is empty")
	}
	return root, err
}
