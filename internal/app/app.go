package app

import (
	"context"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/controller"
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
	cfg    *Config
	router *chi.Mux
	db     *repository.Database
	logger *zap.Logger
	server *http.Server
}

func New(cfg *Config) *App {
	return &App{
		cfg:    cfg,
		router: chi.NewRouter(),
		logger: zap.L(),
	}
}

func (a *App) Run(ctx context.Context) error {
	a.initDB()
	a.initRouter()

	a.server = &http.Server{
		Addr:    a.cfg.RunAddress,
		Handler: a.router,
	}

	go func() {
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.logger.Fatal("Server failed", zap.Error(err))
		}
	}()

	<-ctx.Done()
	return a.shutdown()
}

func (a *App) initDB() error {
	db, err := repository.NewDatabase(a.cfg.DatabaseURI)
	if err != nil {
		a.logger.Error("Database initialization failed",
			zap.Error(err),
			zap.String("dsn", a.cfg.MaskDBPassword()))
		return err
	}
	a.db = db

	a.logger.Info("Database initialized successfully")
	return nil
}

func (a *App) initRouter() {
	a.router.Use(middleware.RequestID)
	a.router.Use(middleware.RealIP)
	a.router.Use(middleware.Logger)
	a.router.Use(middleware.Recoverer)
	a.router.Use(middleware.Compress(5))

	// Services
	userRepo := repository.NewUserRepository(a.db)
	orderRepo := repository.NewOrderRepository(a.db)
	withdrawalRepo := repository.NewWithdrawalRepository(a.db)

	authService := service.NewAuthService(userRepo)
	orderService := service.NewOrderService(orderRepo, a.cfg.AccrualSystemAddress)
	balanceService := service.NewBalanceService(userRepo, orderRepo, withdrawalRepo)
	withdrawalService := service.NewWithdrawalService(withdrawalRepo, userRepo)

	logger := a.logger
	// Controllers
	authController := controller.NewAuthController(authService, logger)
	orderController := controller.NewOrderController(orderService, logger)
	balanceController := controller.NewBalanceController(balanceService)
	withdrawalController := controller.NewWithdrawalController(withdrawalService)

	// Public routes
	a.router.Post("/api/user/register", authController.Register)
	a.router.Post("/api/user/login", authController.Login)

	// Protected routes
	a.router.Group(func(r chi.Router) {
		r.Use(middlewareinternal.JWTAuthMiddleware(authService))

		r.Post("/api/user/orders", orderController.UploadOrder)
		r.Get("/api/user/orders", orderController.GetOrders)
		r.Get("/api/user/balance", balanceController.GetBalance)
		r.Post("/api/user/balance/withdraw", withdrawalController.Withdraw)
		r.Get("/api/user/withdrawals", withdrawalController.GetWithdrawals)
	})
}

func (a *App) shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return a.server.Shutdown(ctx)
}
