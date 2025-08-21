package main

import (
	"log"
	"net/http"

	"github.com/AlexisZankowitch/concept-insight/slack"
	"github.com/AlexisZankowitch/concept-insight/utils"
	"github.com/strowk/foxy-contexts/pkg/app"
	"github.com/strowk/foxy-contexts/pkg/fxctx"
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"github.com/strowk/foxy-contexts/pkg/streamable_http"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

type MySessionData struct {
	isItGreat bool
}

func (m *MySessionData) String() string {
	return "MySessionData"
}

func main() {
	slackService := slack.NewSlackService()


	server := app.
	NewBuilder().
	// adding the tool to the app
	WithTool(func() fxctx.Tool { return slack.NewFindTechExpertTool(slackService) }).
	WithServerCapabilities(&mcp.ServerCapabilities{
		Tools: &mcp.ServerCapabilitiesTools{
			ListChanged: utils.Ptr(false),
		},
	}).
	// setting up server
	WithName("great-tool-server").
	WithVersion("0.0.1").
	WithTransport(
		streamable_http.NewTransport(
			streamable_http.Endpoint{
				Hostname: "localhost",
				Port:     8080,
				Path:     "/mcp",
			}),
		).
		// Configuring fx logging to only show errors
		WithFxOptions(
			fx.Provide(func() *zap.Logger {
				cfg := zap.NewDevelopmentConfig()
				cfg.Level.SetLevel(zap.ErrorLevel)
				logger, _ := cfg.Build()
				return logger
			}),
			fx.Option(fx.WithLogger(
				func(logger *zap.Logger) fxevent.Logger {
					return &fxevent.ZapLogger{Logger: logger}
				},
			)),
		)

	err := server.Run()
	if err != nil {
		if err == http.ErrServerClosed {
			log.Println("Server closed")
		} else {
			log.Fatalf("Server error: %v", err)
		}
	}
}

// --8<-- [end:server]
