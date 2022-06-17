package main

import (
	"github.com/go-redis/redis/v8"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"limq/api"
	"limq/authenticator"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	{
		var config zap.Config

		if len(os.Getenv("DEBUG")) > 0 {
			config = zap.NewDevelopmentConfig()
		} else {
			config = zap.NewProductionConfig()
		}

		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		logger, _ := config.Build()
		zap.ReplaceGlobals(logger)
		defer logger.Sync()
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     envOrDefault("REDIS", "localhost:6379"),
		Password: envOrDefault("REDIS_PASSWORD", ""),
		DB:       envIntOrDefault("REDIS_DB", 3),
	})

	authManager := authenticator.NewA(rdb)
	stubManager := api.NewStub(authManager)

	server := &fasthttp.Server{}
	server.Handler = stubManager.Handler()

	signalNotifier := make(chan os.Signal, 1)
	signal.Notify(signalNotifier, os.Interrupt, syscall.SIGTERM, syscall.SIGSTOP)

	go func() {
		<-signalNotifier
		server.Shutdown()
		zap.L()
	}()

	zap.L().Info("starting the server on :8080")
	if err := server.ListenAndServe(":8080"); err != nil {
		zap.L().Error("error on startup: " + err.Error())
	}

	zap.L().Info("server is terminated")
}
