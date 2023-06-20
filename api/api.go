package api

import (
	"html/template"
	"math"
	"sort"
	"strconv"
	"time"

	"indexer/exchange"
	"indexer/store"
	"indexer/token"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

const (
	CandlesPerPage = 500
)

type Api struct {
	engine          *gin.Engine
	exchanges       map[string]exchange.Exchange
	exchangeManager *exchange.ExchangeManager
	stores          store.StoreManager
	logger          zerolog.Logger
}

func NewApi(exchanges map[string]exchange.Exchange, exchangeManager *exchange.ExchangeManager, stores store.StoreManager, logger zerolog.Logger) *Api {
	apiLogger := logger.With().Str("api", "gin").Logger()
	engine := gin.New()
	a := &Api{
		engine:          engine,
		exchanges:       exchanges,
		exchangeManager: exchangeManager,
		stores:          stores,
		logger:          apiLogger,
	}
	a.AddMiddleware()
	a.AddRoutes()
	return a
}

func (a *Api) AddMiddleware() {
	a.engine.Use(func(ctx *gin.Context) {
		now := time.Now()
		path := ctx.Request.URL.Path
		if ctx.Request.URL.RawQuery != "" {
			path = path + "?" + ctx.Request.URL.RawQuery
		}
		ctx.Next()
		latency := time.Since(now)
		params := gin.LogFormatterParams{
			BodySize:     ctx.Writer.Size(),
			ClientIP:     ctx.ClientIP(),
			ErrorMessage: ctx.Errors.ByType(gin.ErrorTypePrivate).String(),
			Latency:      latency,
			Method:       ctx.Request.Method,
			Path:         path,
			StatusCode:   ctx.Writer.Status(),
			TimeStamp:    now.Add(latency),
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
	indexTmpl, err := template.New("index.html").Parse(`<html color-mode="user">
		<head>
			<title>currents | Price API</title>
			<link rel="stylesheet" href="https://unpkg.com/mvp.css@1.12/mvp.css" crossorigin integrity="sha384-7msajHe9jAIT4i9ezoDS64luSrU1be0dyZK9tXwgeFLoOsCwkDK0SsbA2qvdQ/v7"> 
		</head>
		<body>
			<header>
				<nav>
					<h1>currents</h1>
					<ul>
						<li><a href="https://docs.mintthemoon.xyz/currents" target="_blank">Docs</a></li>
						<li><a href="https://indexer" target="_blank">Source</a></li>
					</ul>
				</nav>
				<h1>Price Indexer API</h1>
				<p>Exchange price tracking simplified.</p>
				<p><small>currents is built for <a href="https://kujira.app" target="_blank">Kujira</a> by <a href="https://mintthemoon.xyz" target="_blank">mintthemoon</a></small></p>
			</header>
			<main>
				<hr/>
				<h2><a href="/exchanges">Exchanges</a></h2>
				<ul>
					{{ range .exchanges }}
						<li>
							<a href="/exchanges/{{ .name }}">{{ .display }}</a>
							<ul>
								<li><a href="/exchanges/{{ .name }}/pairs">Pairs</a></li>
								<li><a href="/exchanges/{{ .name }}/tickers">Tickers</a></li>
								<li><a href="/exchanges/{{ .name }}/candles">Candles</a></li>
								<li><a href="/exchanges/{{ .name }}/trades">Trades</a></li>
							</ul>
						</li>
					{{ end }}
				</ul>
			</main>
		</body>
	</html>`)
	if err != nil {
		return err
	}
	a.engine.SetHTMLTemplate(indexTmpl)
	a.engine.GET("/", func(ctx *gin.Context) {
		exchanges := make([]gin.H, len(a.exchanges))
		i := 0
		for _, exchange := range a.exchanges {
			exchanges[i] = gin.H{
				"name":    exchange.Name(),
				"display": exchange.DisplayName(),
			}
			i++
		}
		ctx.HTML(200, "index.html", gin.H{"exchanges": exchanges})
	})
	a.engine.GET("/exchanges", func(ctx *gin.Context) {
		exchanges := make([]string, len(a.exchanges))
		i := 0
		for _, exchange := range a.exchanges {
			exchanges[i] = exchange.Name()
			i++
		}
		ctx.JSON(200, gin.H{"exchanges": exchanges})
	})
	a.engine.GET("/exchanges/:exchange", func(ctx *gin.Context) {
		exchangeName := ctx.Param("exchange")
		e, ok := a.exchanges[exchangeName]
		if ok {
			ctx.JSON(200, gin.H{
				"exchange": gin.H{
					"display": e.DisplayName(),
					"name":    e.Name(),
				},
			})
		} else {
			ctx.JSON(404, gin.H{"error": "exchange not found"})
		}
	})
	a.engine.GET("/exchanges/:exchange/pairs", func(ctx *gin.Context) {
		exchangeName := ctx.Param("exchange")
		e, ok := a.exchanges[exchangeName]
		if !ok {
			ctx.JSON(404, gin.H{"error": "exchange not found"})
			return
		}
		pairs, err := e.Pairs()
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}
		pairStrings := make([]string, len(pairs))
		for i, pair := range pairs {
			pairStrings[i] = pair.String()
		}
		sort.Slice(pairStrings, func(i, j int) bool {
			return pairStrings[i] < pairStrings[j]
		})
		ctx.JSON(200, gin.H{"pairs": pairStrings})
	})
	a.engine.GET("/exchanges/:exchange/tickers", func(ctx *gin.Context) {
		exchangeName := ctx.Param("exchange")
		_, ok := a.exchanges[exchangeName]
		if !ok {
			ctx.JSON(404, gin.H{"error": "exchange not found"})
			return
		}
		tickers, err := a.exchangeManager.Tickers(exchangeName)
		if err != nil {
			ctx.JSON(404, gin.H{"error": "exchange tickers not found"})
			return
		}
		sort.Slice(tickers, func(i, j int) bool {
			return tickers[i].BaseAsset < tickers[j].BaseAsset
		})
		ctx.JSON(200, gin.H{"tickers": tickers})
	})
	a.engine.GET("/exchanges/:exchange/candles", func(ctx *gin.Context) {
		exchangeName := ctx.Param("exchange")
		_, ok := a.exchanges[exchangeName]
		if !ok {
			ctx.JSON(404, gin.H{"error": "exchange not found"})
			return
		}
		ctx.JSON(400, gin.H{"error": "must provide base/quote pair in request, e.g. /exchanges/" + exchangeName + "/candles/BASE/QUOTE"})
	})
	a.engine.GET("/exchanges/:exchange/trades", func(ctx *gin.Context) {
		exchangeName := ctx.Param("exchange")
		_, ok := a.exchanges[exchangeName]
		if !ok {
			ctx.JSON(404, gin.H{"error": "exchange not found"})
			return
		}
		ctx.JSON(400, gin.H{"error": "must provide base/quote pair in request, e.g. /exchanges/" + exchangeName + "/trades/BASE/QUOTE"})
	})
	a.engine.GET("/exchanges/:exchange/tickers/:base/:quote", func(ctx *gin.Context) {
		exchangeName := ctx.Param("exchange")
		_, ok := a.exchanges[exchangeName]
		if !ok {
			ctx.JSON(404, gin.H{"error": "exchange not found"})
			return
		}
		pair := &token.Pair{
			Base:  ctx.Param("base"),
			Quote: ctx.Param("quote"),
		}
		ticker, err := a.exchangeManager.Ticker(exchangeName, pair)
		if err != nil {
			ctx.JSON(404, gin.H{"error": "tickers not found"})
			return
		}
		ctx.JSON(200, gin.H{"ticker": ticker})
	})
	a.engine.GET("/exchanges/:exchange/candles/:base/:quote", func(ctx *gin.Context) {
		exchangeName := ctx.Param("exchange")
		_, ok := a.exchanges[exchangeName]
		if !ok {
			ctx.JSON(404, gin.H{"error": "exchange not found"})
			return
		}
		pair := &token.Pair{
			Base:  ctx.Param("base"),
			Quote: ctx.Param("quote"),
		}
		candles, err := a.exchangeManager.Candles(exchangeName, pair)
		isReversed := false
		if err != nil {
			candles, err = a.exchangeManager.Candles(exchangeName, pair.Reversed())
			if err != nil {
				ctx.JSON(404, gin.H{"error": "candles not found"})
				return
			}
			isReversed = true
		}
		numCandles := candles.Len() - 1
		numPages := int(math.Ceil(float64(numCandles) / CandlesPerPage))
		page, err := strconv.Atoi(ctx.DefaultQuery("page", "1"))
		if err != nil || page > numPages || page < 1 {
			ctx.JSON(400, gin.H{"error": "invalid page"})
			return
		}
		pageStart := (page-1)*CandlesPerPage + 1
		pageEnd := pageStart + CandlesPerPage + 1
		if pageEnd > numCandles {
			pageEnd = numCandles
		}
		candlesList := candles.ListRange(pageStart, pageEnd)
		if isReversed {
			for i, candle := range candlesList {
				candlesList[i] = candle.Reversed()
			}
		}
		pagedCandles := gin.H{"page": gin.H{"current": page, "total": numPages}, "candles": candlesList}
		ctx.JSON(200, pagedCandles)
	})
	a.engine.GET("/exchanges/:exchange/trades/:base/:quote", func(ctx *gin.Context) {
		exchangeName := ctx.Param("exchange")
		_, ok := a.exchanges[exchangeName]
		if !ok {
			ctx.JSON(404, gin.H{"error": "exchange not found"})
			return
		}
		store, err := a.stores.Store(exchangeName)
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}
		pair := &token.Pair{
			Base:  ctx.Param("base"),
			Quote: ctx.Param("quote"),
		}
		period, err := time.ParseDuration(ctx.DefaultQuery("period", "1h"))
		if err != nil {
			ctx.JSON(400, gin.H{"error": "invalid period"})
			return
		}
		endStr := ctx.DefaultQuery("end", "now")
		var end time.Time
		if endStr == "now" {
			end = time.Now()
		} else {
			end, err = time.Parse(time.RFC3339, endStr)
			if err != nil {
				ctx.JSON(400, gin.H{"error": "invalid end"})
				return
			}
		}
		trades, err := store.Trades(pair, end.Add(-period), end)
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(200, gin.H{"trades": trades})
	})
	return nil
}

func (a *Api) Start() {
	a.engine.Run()
}
