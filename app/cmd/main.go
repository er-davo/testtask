// @title Subscriptions API
// @version 1.0
// @description REST API для управления подписками пользователей
// @host localhost:8080
// @BasePath /
package main

import (
	"context"
	"os"
	"os/signal"
	"subscriptionsservice/internal/application"
	"subscriptionsservice/internal/config"
	"subscriptionsservice/internal/database"
	"subscriptionsservice/internal/logger"
	"syscall"

	"go.uber.org/zap"
)

func main() {

	configFilePath := os.Getenv("CONFIG_PATH")
	if configFilePath == "" {
		panic("env ConfigPath is empty")
	}
	cfg, err := config.Load(configFilePath)
	if err != nil {
		panic("error on loading config: " + err.Error())
	}

	log := logger.NewLogger(cfg.App.LogLevel)
	defer log.Sync()

	err = database.Migrate(cfg.App.MirgationDir, cfg.DatabaseURL)
	if err != nil {
		log.Fatal("error on migrating database", zap.Error(err))
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	app := application.New(cfg, log)
	if err != nil {
		log.Fatal("error on creating app", zap.Error(err))
	}

	if err := app.Run(ctx); err != nil {
		if ctx.Err() != nil {
			log.Info("app stopped by context")
		} else {
			log.Error("app exited with error", zap.Error(err))
		}
	}
}
