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
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger, _ := config.Build()
	zap.ReplaceGlobals(logger)

	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       3,  // use default DB
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
