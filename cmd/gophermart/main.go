package main

import (
	"context"
	"fmt"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/app"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/repository"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/util/logger"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg := app.NewConfigFromFlags()

	if err := logger.Init(cfg.LogLevel); err != nil {
		panic(fmt.Sprintf("Failed to init logger: %v", err))
	}
	defer logger.Sync()

	logger.Log.Info("Testing database connection and migrations...")
	testDB, err := repository.NewDatabase(cfg.DatabaseURI)
	if err != nil {
		logger.Log.Fatal("Database initialization failed", zap.Error(err))
	}
	defer testDB.Close()
	logger.Log.Info("Database connection and migrations OK")

	application := app.New(cfg)
	runServer(application, cfg)
}

func runServer(application *app.App, cfg *app.Config) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	serverCtx, serverCancel := context.WithCancel(context.Background())
	defer serverCancel()

	go func() {
		logger.Log.Info("Starting HTTP server", zap.String("address", cfg.RunAddress))
		if err := application.Run(serverCtx); err != nil {
			logger.Log.Fatal("Server failed", zap.Error(err))
		}
	}()

	sig := <-sigChan
	logger.Log.Info("Received shutdown signal", zap.String("signal", sig.String()))
	serverCancel()
}
