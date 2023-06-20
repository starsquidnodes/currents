package exchange

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"indexer/chain"
	"indexer/config"
	"indexer/store"
	"indexer/token"
	"indexer/trading"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/osmosis-labs/assetlist"
	"github.com/rs/zerolog"
)

type (
	OsmosisExchange struct {
		rpc                *chain.CometRpc
		assets             map[string]*assetlist.Asset
		assetsSymbol       map[string]*assetlist.Asset
		pairs              []*token.Pair
		store              store.Store
		tradeSubscriptions []chan *trading.Trade
		pairSubscriptions  []chan []*token.Pair
		logger             zerolog.Logger
	}

	OsmosisTokenSwap struct {
		In   token.Token
		Out  token.Token
		Pool string
	}
)

func NewOsmosisExchange(url string, store store.Store, logger zerolog.Logger) (*OsmosisExchange, error) {
	rpc, err := chain.NewCometRpc(url, logger)
	if err != nil {
		return nil, err
	}
	o := &OsmosisExchange{
		rpc:    rpc,
		store:  store,
		logger: logger,
	}
	o.logger.Info().Msg("exchange connected")
	err = o.PollAssetList()
	return o, err
}

func (o *OsmosisExchange) Name() string {
	return "osmosis"
}

func (o *OsmosisExchange) DisplayName() string {
	return "Osmosis"
}

func (o *OsmosisExchange) Start() error {
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
				for _, subscription := range o.tradeSubscriptions {
					subscription <- &trade
				}
			}
		}
	}()
	return nil
}

func (o *OsmosisExchange) SubscribeTrades() chan *trading.Trade {
	channel := make(chan *trading.Trade)
	o.tradeSubscriptions = append(o.tradeSubscriptions, channel)
	return channel
}

func (o *OsmosisExchange) SubscribePairs() chan []*token.Pair {
	channel := make(chan []*token.Pair)
	o.pairSubscriptions = append(o.pairSubscriptions, channel)
	return channel
}

func (o *OsmosisExchange) Pairs() ([]*token.Pair, error) {
	return o.pairs, nil
}

func (o *OsmosisExchange) Store() store.Store {
	return o.store
}

func (o *OsmosisExchange) GetTrades(event *coretypes.ResultEvent) []trading.Trade {
	trades := []trading.Trade{}
	swaps, err := ParseOsmosisTokenSwaps(event)
	if err != nil {
		o.logger.Error().Err(err).Msg("failed to parse swap event")
		return trades
	}
	if len(o.assets) == 0 {
		o.logger.Warn().Msg("cannot process trades when asset list is empty")
		return trades
	}
	now := time.Now().UTC()
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
		supportedPools := o.GetSupportedPools(inAsset, outAsset)
		_, ok = supportedPools[swap.Pool]
		if !ok {
			continue
		}
		trades = append(trades, trading.Trade{Base: *base, Quote: *quote, Time: now})
	}
	return trades
}

func (o *OsmosisExchange) PollAssetList() error {
	go func() {
		for {
			cfg := config.Cfg.ExchangeConfig["osmosis"]
			assetList, err := LoadOsmosisAssetList(cfg.AssetsUrl)
			if err != nil {
				o.logger.Error().Err(err).Str("url", cfg.AssetsUrl).Msg("failed to load asset list")
				time.Sleep(cfg.AssetsRetryInterval)
				continue
			}
			assets := make(map[string]*assetlist.Asset, len(assetList.Assets))
			assetsSymbol := make(map[string]*assetlist.Asset, len(assetList.Assets))
			pairs := []*token.Pair{}
			pools := map[string]struct{}{}
			for i, asset := range assetList.Assets {
				assets[asset.Base] = &assetList.Assets[i]
				if asset.Base == "ibc/D189335C6E4A68B513C10AB227BF1C1D38C746766278BA3EEB4FB14124F1D858" {
					assetList.Assets[i].Symbol = "USDC.axl"
					assetsSymbol["USDC.axl"] = &assetList.Assets[i]
				} else if asset.Base == "ibc/8242AD24008032E457D2E12D46588FD39FB54FB29680C6C7663D296B383C37C4" {
					assetList.Assets[i].Symbol = "USDT.axl"
					assetsSymbol["USDT.axl"] = &assetList.Assets[i]
				} else {
					assetsSymbol[asset.Symbol] = &assetList.Assets[i]
				}
			}
			o.assets = assets
			o.assetsSymbol = assetsSymbol
			for _, asset := range o.assets {
				if asset.Symbol == "OSMO" {
					continue
				}
				supportedPools := o.GetSupportedPools(asset)
				for id, quoteSymbol := range supportedPools {
					_, ok := pools[id]
					if ok {
						o.logger.Debug().Str("base", asset.Symbol).Str("quote", quoteSymbol).Str("id", id).Msg("skipping already present pool")
						continue
					}
					quoteAsset, ok := o.assetsSymbol[quoteSymbol]
					if !ok {
						o.logger.Debug().Str("symbol", quoteSymbol).Msg("skipping unlisted asset pair")
						continue
					}
					pair := &token.Pair{
						Base:  asset.Symbol,
						Quote: quoteAsset.Symbol,
					}
					pairs = append(pairs, pair)
					pools[id] = struct{}{}
				}
			}
			for _, subscription := range o.pairSubscriptions {
				subscription <- pairs
			}
			o.pairs = pairs
			o.logger.Debug().Int("num_assets", len(o.assets)).Msg("refreshed asset list")
			time.Sleep(cfg.AssetsRefreshInterval)
		}
	}()
	return nil
}

func (o *OsmosisExchange) GetSupportedPools(assets ...*assetlist.Asset) map[string]string {
	supportedPools := map[string]string{}
	for _, asset := range assets {
		for _, keyword := range asset.Keywords {
			fields := strings.Split(keyword, ":")
			if len(fields) != 2 {
				continue
			}
			_, err := strconv.Atoi(fields[1])
			if err != nil {
				continue
			}
			supportedPools[fields[1]] = fields[0]
		}
	}
	return supportedPools
}

func RebaseOsmosisAsset(t *token.Token, asset *assetlist.Asset) (*token.Token, error) {
	exponents := make(map[string]int64, len(asset.DenomUnits))
	for _, denomUnit := range asset.DenomUnits {
		exponents[denomUnit.Denom] = denomUnit.Exponent
	}
	displayExponent, ok := exponents[asset.Display]
	if !ok {
		return &token.Token{}, fmt.Errorf("could not determine display units for %s", t.Symbol)
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
			In:   *in,
			Out:  *out,
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
