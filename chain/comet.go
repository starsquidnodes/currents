package chain

import (
	"context"

	rpcclient "github.com/cometbft/cometbft/rpc/client"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	jsonrpcclient "github.com/cometbft/cometbft/rpc/jsonrpc/client"
	"github.com/rs/zerolog"
)

type CometRpc struct {
	ctx context.Context
	client rpcclient.Client
	url string
	logger zerolog.Logger
}

func NewCometRpc(url string, logger zerolog.Logger) (*CometRpc, error) {
	cometLogger := logger.With().Str("client", "comet").Logger()
	httpClient, err := jsonrpcclient.DefaultHTTPClient(url)
	if err != nil {
		cometLogger.Error().Err(err).Msg("failed to create client HTTP connection")
		return nil, err
	}
	rpcClient, err := rpchttp.NewWithClient(url, "/websocket", httpClient)
	if err != nil {
		cometLogger.Error().Err(err).Msg("failed to create client")
		return nil, err
	}
	c := &CometRpc{
		ctx: context.Background(),
		client: rpcClient,
		url: url,
		logger: cometLogger,
	}
	c.logger.Debug().Str("url", url).Msg("client connected")
	return c, nil
}

func (c *CometRpc) Height() (int64, error) {
	status, err := c.client.Status(c.ctx)
	if err != nil {
		c.logger.Error().Err(err).Str("method", "status").Msg("failed to get chain height")
		return 0, err
	}
	c.logger.Debug().Int64("height", status.SyncInfo.LatestBlockHeight).Msg("got chain height")
	return status.SyncInfo.LatestBlockHeight, nil
}

func (c *CometRpc) Block(height int64) (*coretypes.ResultBlock, error) {
	block, err := c.client.Block(c.ctx, &height)
	if err != nil {
		c.logger.Error().Err(err).Str("method", "block").Int64("height", height).Msg("failed to get block")
		return nil, err
	}
	c.logger.Debug().Int64("height", height).Str("hash", block.Block.Hash().String()).Msg("got block")
	return block, nil
}

func (c *CometRpc) Subscribe(query string) (<-chan coretypes.ResultEvent, error) {
	err := c.client.Start()
	if err != nil {
		c.logger.Error().Err(err).Str("method", "start").Msg("failed to start client")
		return nil, err
	}
	channel, err := c.client.Subscribe(c.ctx, "", query)
	if err != nil {
		c.logger.Error().Err(err).Str("method", "subscribe").Msg("failed to subscribe")
	}
	c.logger.Debug().Str("query", query).Msg("subscribed")
	return channel, err
}