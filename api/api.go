package api

import (
	"html/template"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mintthemoon/chaindex/exchange"
	"github.com/mintthemoon/chaindex/store"
	"github.com/mintthemoon/chaindex/token"
	"github.com/mintthemoon/chaindex/trading"
	"github.com/rs/zerolog"
)

type Api struct {
	engine *gin.Engine
	exchanges map[string]exchange.Exchange
	exchangeManager *exchange.ExchangeManager
	stores store.StoreManager
	logger zerolog.Logger
}

func NewApi(exchanges map[string]exchange.Exchange, exchangeManager *exchange.ExchangeManager, stores store.StoreManager, logger zerolog.Logger) *Api {
	apiLogger := logger.With().Str("api", "gin").Logger()
	engine := gin.New()
	a := &Api{
		engine: engine,
		exchanges: exchanges,
		exchangeManager: exchangeManager,
		stores: stores,
		logger: apiLogger,
	}
	a.AddMiddleware()
	a.AddRoutes()
	return a
}

func (a *Api) AddMiddleware() {
	a.engine.Use(func (ctx *gin.Context) {
		now := time.Now()
		path := ctx.Request.URL.Path
		if ctx.Request.URL.RawQuery != "" {
			path = path + "?" + ctx.Request.URL.RawQuery
		}
		ctx.Next()
		latency := time.Since(now)
		params := gin.LogFormatterParams{
			BodySize: ctx.Writer.Size(),
			ClientIP: ctx.ClientIP(),
			ErrorMessage: ctx.Errors.ByType(gin.ErrorTypePrivate).String(),
			Latency: latency,
			Method: ctx.Request.Method,
			Path: path,
			StatusCode: ctx.Writer.Status(),
			TimeStamp: now.Add(latency),
		}
		var event *zerolog.Event
		if ctx.Writer.Status() >= 500 {
			event = a.logger.Error()
		} else {
			event = a.logger.Info()
		}
		event.
			Int("body_size", params.BodySize).
			Str("client_ip", params.ClientIP).
			Str("latency", params.Latency.String()).
			Str("method", params.Method).
			Str("path", params.Path).
			Int("status_code", params.StatusCode).
			Msg(params.ErrorMessage)
	})
	a.engine.Use(gin.Recovery())
}

func (a *Api) AddRoutes() error {
	indexTmpl, err := template.New("index.html").Parse(`<html>
		<body>
			<h1>Chaindex</h1>
			<h2><a href="/exchange">Exchanges</a></h2>
			<ul>
				{{ range .exchanges }}
					<li>
						<a href="/exchange/{{ .name }}">{{ .display }}</a>
						<ul>
							<li><a href="/exchange/{{ .name }}/trade">Trades</a></li>
						</ul>
					</li>
				{{ end }}
			</ul>
		</body>
	</html>`)
	if err != nil {
		return err
	}
	a.engine.SetHTMLTemplate(indexTmpl)
	a.engine.GET("/", func (ctx *gin.Context) {
		exchanges := make([]gin.H, len(a.exchanges))
		i := 0
		for _, exchange := range a.exchanges {
			exchanges[i] = gin.H{
				"name": exchange.Name(),
				"display": exchange.DisplayName(),
			}
			i++
		}
		ctx.HTML(200, "index.html", gin.H{"exchanges": exchanges})
	})
	a.engine.GET("/exchanges", func (ctx *gin.Context) {
		exchanges := make([]string, len(a.exchanges))
		i := 0
		for _, exchange := range a.exchanges {
			exchanges[i] = exchange.Name()
			i++
		}
		ctx.JSON(200, gin.H{"exchanges": exchanges})
	})
	a.engine.GET("/exchanges/:exchange", func (ctx *gin.Context) {
		exchangeName := ctx.Param("exchange")
		e, ok := a.exchanges[exchangeName]
		if ok {
			ctx.JSON(200, gin.H{
				"exchange": gin.H{
					"display": e.DisplayName(),
					"name": e.Name(),
				},
			})
		} else {
			ctx.JSON(404, gin.H{"error": "exchange not found"})
		}
	})
	a.engine.GET("/exchanges/:exchange/pairs", func (ctx *gin.Context) {
		exchangeName := ctx.Param("exchange")
		e, ok := a.exchanges[exchangeName]
		if ok {
			pairs, err := e.Pairs()
			if err != nil {
				ctx.JSON(500, gin.H{"error": err.Error()})
				return
			}
			pairStrings := make([]string, len(pairs))
			for i, pair := range pairs {
				pairStrings[i] = pair.String()
			}
			ctx.JSON(200, gin.H{"pairs": pairStrings})
		} else {
			ctx.JSON(404, gin.H{"error": "exchange not found"})
		}
	})
	a.engine.GET("/exchanges/:exchange/trades/:base/:quote", func (ctx *gin.Context) {
		exchangeName := ctx.Param("exchange")
		_, ok := a.exchanges[exchangeName]
		if ok {
			store, err := a.stores.Store(exchangeName)
			if err != nil {
				ctx.JSON(500, gin.H{"error": err.Error()})
				return
			}
			pair := &token.Pair{
				Base: ctx.Param("base"),
				Quote: ctx.Param("quote"),
			}
			now := time.Now()
			trades, err := store.Trades(pair, now.Add(-time.Hour), now)
			if err != nil {
				ctx.JSON(500, gin.H{"error": err.Error()})
				return
			}
			ctx.JSON(200, gin.H{"trades": trades})
		} else {
			ctx.JSON(404, gin.H{"error": "exchange not found"})
		}
	})
	a.engine.GET("/exchanges/:exchange/candles/:base/:quote", func (ctx *gin.Context) {
		exchangeName := ctx.Param("exchange")
		_, ok := a.exchanges[exchangeName]
		if ok {
			pair := &token.Pair{
				Base: ctx.Param("base"),
				Quote: ctx.Param("quote"),
			}
			exchangeCandles, ok := a.exchangeManager.Candles[exchangeName]
			if !ok {
				ctx.JSON(404, gin.H{"error": "exchange candles not found"})
				return
			}
			candles, ok := exchangeCandles[pair.String()]
			if !ok {
				reversedCandles, ok := exchangeCandles[pair.Reversed().String()]
				if !ok {
					ctx.JSON(404, gin.H{"error": "pair candles not found"})
					return
				}
				candles = make([]*trading.Candle, len(reversedCandles))
				for i, candle := range reversedCandles {
					candles[i] = candle.Reversed()
				}
			}
			ctx.JSON(200, gin.H{"candles": candles})
		} else {
			ctx.JSON(404, gin.H{"error": "exchange not found"})
		}
	})
	a.engine.GET("/exchanges/:exchange/tickers/:base/:quote", func (ctx *gin.Context) {
		exchangeName := ctx.Param("exchange")
		_, ok := a.exchanges[exchangeName]
		if ok {
			pair := &token.Pair{
				Base: ctx.Param("base"),
				Quote: ctx.Param("quote"),
			}
			exchangeTickers, ok := a.exchangeManager.Tickers[exchangeName]
			if !ok {
				ctx.JSON(404, gin.H{"error": "exchange tickers not found"})
				return
			}
			ticker, ok := exchangeTickers[pair.String()]
			if !ok {
				reversedTicker, ok := exchangeTickers[pair.Reversed().String()]
				if !ok {
					ctx.JSON(404, gin.H{"error": "pair ticker not found"})
					return
				}
				ticker = reversedTicker.Reversed()
			}
			ctx.JSON(200, gin.H{"ticker": ticker})
		} else {
			ctx.JSON(404, gin.H{"error": "exchange not found"})
		}
	})
	return nil
}

func (a *Api) Start() {
	a.engine.Run()
}