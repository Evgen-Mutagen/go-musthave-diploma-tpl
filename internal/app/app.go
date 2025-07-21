package app

import (
	"context"
	"fmt"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/controller"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/core"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/middlewareinternal"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/repository"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
	"net/http"
	"time"
)

type App struct {
	cfg          *Config
	Router       *chi.Mux
	db           *repository.Database
	Logger       *zap.Logger
	Server       *http.Server
	OrderService core.OrderProcessor
}

func New(cfg *Config) *App {
	app := &App{
		cfg:    cfg,
		Router: chi.NewRouter(),
		Logger: zap.L(),
	}

	app.initDB()

	orderRepo := repository.NewOrderRepository(app.db)

	userRepo := repository.NewUserRepository(app.db)

	app.OrderService = service.NewOrderService(orderRepo, cfg.AccrualSystemAddress, userRepo, app.Logger)

	app.initRouter()
	return app
}

func (a *App) Run(ctx context.Context) error {
	a.initDB()
	a.initRouter()

	a.Server = &http.Server{
		Addr:    a.cfg.RunAddress,
		Handler: a.Router,
	}

	go func() {
		if err := a.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.Logger.Fatal("Server failed", zap.Error(err))
		}
	}()

	<-ctx.Done()
	return a.shutdown()
}

func (a *App) initDB() error {

	dbConfig := repository.DatabaseConfig{
		DSN:            a.cfg.DatabaseURI,
		MigrationsPath: a.cfg.MigrationsPath,
	}

	db, err := repository.NewDatabase(dbConfig)
	if err != nil {
		a.Logger.Error("Database initialization failed",
			zap.String("dsn", a.cfg.MaskDBPassword()),
			zap.Error(err))
		return fmt.Errorf("database initialization failed: %w", err)
	}

	a.db = db
	a.Logger.Info("Database initialized successfully",
		zap.String("migrations_path", a.cfg.MigrationsPath))

	return nil
}

func (a *App) initRouter() {
	a.Router.Use(middleware.RequestID)
	a.Router.Use(middleware.RealIP)
	a.Router.Use(middleware.Logger)
	a.Router.Use(middleware.Recoverer)
	a.Router.Use(middleware.Compress(5))

	// Services
	userRepo := repository.NewUserRepository(a.db)
	orderRepo := repository.NewOrderRepository(a.db)
	withdrawalRepo := repository.NewWithdrawalRepository(a.db)

	authService := service.NewAuthService(userRepo, a.cfg.JWTSecretKey)
	orderService := service.NewOrderService(orderRepo, a.cfg.AccrualSystemAddress, userRepo, a.Logger)
	balanceService := service.NewBalanceService(userRepo, orderRepo, withdrawalRepo)
	withdrawalService := service.NewWithdrawalService(withdrawalRepo, userRepo)

	logger := a.Logger
	// Controllers
	authController := controller.NewAuthController(authService, logger)
	orderController := controller.NewOrderController(orderService, logger)
	balanceController := controller.NewBalanceController(balanceService)
	withdrawalController := controller.NewWithdrawalController(withdrawalService)

	// Public routes
	a.Router.Post("/api/user/register", authController.Register)
	a.Router.Post("/api/user/login", authController.Login)

	// Protected routes
	a.Router.Group(func(r chi.Router) {
		r.Use(middlewareinternal.JWTAuthMiddleware(authService))

		r.Post("/api/user/orders", orderController.UploadOrder)
		r.Get("/api/user/orders", orderController.GetOrders)
		r.Get("/api/user/balance", balanceController.GetBalance)
		r.Post("/api/user/balance/withdraw", withdrawalController.Withdraw)
		r.Get("/api/user/withdrawals", withdrawalController.GetWithdrawals)
	})
}

func (a *App) shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	return a.Server.Shutdown(ctx)
}

func StartOrderProcessor(ctx context.Context, processor core.OrderProcessor, logger *zap.Logger) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Order processing stopped")
			return
		case <-ticker.C:
			if err := processor.ProcessOrders(ctx); err != nil {
				logger.Error("Order processing failed", zap.Error(err))
			}
		}
	}
}
