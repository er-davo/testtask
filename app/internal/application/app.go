package application

import (
	"context"

	"subscriptionsservice/internal/config"
	"subscriptionsservice/internal/database"
	"subscriptionsservice/internal/handler"
	"subscriptionsservice/internal/repository"
	"subscriptionsservice/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	_ "subscriptionsservice/internal/docs"
)

// App represents the application with its dependencies.
type App struct {
	cfg *config.Config

	db     *pgxpool.Pool
	engine *gin.Engine

	log *zap.Logger
}

// New creates a new App instance, initializes database, services, handlers and routes.
func New(cfg *config.Config, log *zap.Logger) *App {
	db, err := database.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}

	e := gin.New()

	subsRepo := repository.NewSubscriptionsRepo(
		db, newRepoRetrier(cfg.Retry, isRetryableFunc),
	)
	subsSvc := service.NewSubscriptionService(subsRepo, log)
	subsHandler := handler.NewSubscriptionHandler(subsSvc, log)

	subsHandler.RegisterRoutes(e)

	e.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return &App{
		cfg:    cfg,
		db:     db,
		engine: e,
		log:    log,
	}
}

// Run starts the HTTP server and waits for context cancellation.
func (a *App) Run(ctx context.Context) error {
	go func() {
		if err := a.engine.Run(":" + a.cfg.App.Port); err != nil {
			a.log.Error("failed to run server", zap.Error(err))
		}
	}()

	<-ctx.Done()
	return a.Shutdown()
}

// Shutdown closes database connections and other resources.
func (a *App) Shutdown() error {
	a.db.Close()
	return nil
}
