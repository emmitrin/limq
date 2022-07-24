package main

import (
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"limq/api"
	"limq/authenticator"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	{
		var config zap.Config

		if len(os.Getenv("DEBUG")) > 0 {
			config = zap.NewDevelopmentConfig()
			config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		} else {
			config = zap.NewProductionConfig()
		}

		logger, _ := config.Build()
		zap.ReplaceGlobals(logger)
		defer logger.Sync()
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     envOrDefault("REDIS", "localhost:6379"),
		Password: envOrDefault("REDIS_PASSWORD", ""),
		DB:       envIntOrDefault("REDIS_DB", 3),
	})

	pool, err := acquirePg()
	if err != nil {
		zap.L().Fatal("unable to set up postgresql", zap.Error(err))
	}

	authManager := authenticator.NewA(rdb)
	stubManager := api.NewStub(pool, authManager)

	server := &fasthttp.Server{}
	server.Handler = stubManager.Handler()

	signalNotifier := make(chan os.Signal, 1)
	signal.Notify(signalNotifier, os.Interrupt, syscall.SIGTERM, syscall.SIGSTOP)

	go func() {
		<-signalNotifier
		server.Shutdown()
		zap.L()
	}()

	address := envOrDefault("ADDRESS", ":8081")

	zap.L().Info("starting the server", zap.String("address", address))

	if err := server.ListenAndServe(address); err != nil {
		zap.L().Error("error on startup: " + err.Error())
	}

	zap.L().Info("server is terminated")
}

func acquirePg() (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return pgxpool.Connect(ctx, os.Getenv("DATABASE_URL"))
}
