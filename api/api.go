package api

import (
	"html/template"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mintthemoon/chaindex/exchange"
	"github.com/rs/zerolog"
)

type Api struct {
	engine *gin.Engine
	exchanges map[string]exchange.Exchange
	logger zerolog.Logger
}

func NewApi(exchanges map[string]exchange.Exchange, logger zerolog.Logger) *Api {
	apiLogger := logger.With().Str("api", "gin").Logger()
	engine := gin.New()
	a := &Api{
		engine: engine,
		exchanges: exchanges,
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

func (a *Api) AddRoutes() {
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
		panic(err)
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
	a.engine.GET("/exchange", func (ctx *gin.Context) {
		exchanges := make([]gin.H, len(a.exchanges))
		i := 0
		for _, exchange := range a.exchanges {
			exchanges[i] = gin.H{
				"name": exchange.Name(),
			}
			i++
		}
		ctx.JSON(200, gin.H{"exchanges": exchanges})
	})
	a.engine.GET("/exchange/:exchange", func (ctx *gin.Context) {
		exchangeName := ctx.Param("exchange")
		c, ok := a.exchanges[exchangeName]
		if ok {
			ctx.JSON(200, gin.H{
				"name": c.Name(),
			})
		} else {
			ctx.JSON(404, gin.H{"error": "exchange not found"})
		}
	})
	a.engine.GET("/exchange/:exchange/trade", func (ctx *gin.Context) {
		exchangeName := ctx.Param("exchange")
		_, ok := a.exchanges[exchangeName]
		if ok {
			ctx.JSON(400, gin.H{"error": "not implemented"})
		} else {
			ctx.JSON(404, gin.H{"error": "exchange not found"})
		}
	})
}

func (a *Api) Start() {
	a.engine.Run()
}