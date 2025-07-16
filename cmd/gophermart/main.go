package main

import (
	"context"
	"fmt"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/app"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/repository"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/util/logger"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				application.Logger.Info("Order processing stopped")
				return
			case <-ticker.C:
				if err := application.OrderService.ProcessOrders(ctx); err != nil {
					application.Logger.Error("Order processing failed", zap.Error(err))
				}
			}
		}
	}()

	application.Server = &http.Server{
		Addr:    cfg.RunAddress,
		Handler: application.Router,
	}

	go func() {
		application.Logger.Info("Starting HTTP server",
			zap.String("address", cfg.RunAddress))
		if err := application.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			application.Logger.Fatal("Server failed", zap.Error(err))
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	application.Logger.Info("Shutting down server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := application.Server.Shutdown(shutdownCtx); err != nil {
		application.Logger.Error("Server shutdown error", zap.Error(err))
	}
	cancel()
}
