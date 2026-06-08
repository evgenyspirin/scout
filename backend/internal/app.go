// Package internal wires the application together (composition root) and
// manages its lifecycle with graceful shutdown.
package internal

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"scout/config"
	"scout/internal/application/authapp"
	"scout/internal/application/photoapp"
	"scout/internal/application/thumbapp"
	"scout/internal/infrastructure/cache"
	sqlitedb "scout/internal/infrastructure/db/sqlite"
	sqlitephoto "scout/internal/infrastructure/db/sqlite/photo"
	"scout/internal/infrastructure/jwt"
	"scout/internal/infrastructure/metrics"
	miniostore "scout/internal/infrastructure/storage/minio"
	"scout/internal/infrastructure/thumbnail"
	"scout/internal/infrastructure/users"
	"scout/internal/interface/api/rest"
)

// App is the composed application.
type App struct {
	logger *zap.Logger
	cfg    config.Config
	db     *sql.DB
	redis  *cache.Redis
	server *rest.Server
}

// NewApp builds the application from configuration.
func NewApp(ctx context.Context) (*App, error) {
	_ = godotenv.Load() // .env is optional (env may be provided by the platform)

	logger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}

	cfg := config.Load()

	db, err := sqlitedb.Open(cfg.SQLite.Path)
	if err != nil {
		return nil, err
	}

	store, err := miniostore.New(ctx, miniostore.Config{
		Endpoint:  cfg.MinIO.Endpoint,
		AccessKey: cfg.MinIO.AccessKey,
		SecretKey: cfg.MinIO.SecretKey,
		Bucket:    cfg.MinIO.Bucket,
		UseSSL:    cfg.MinIO.UseSSL,
	})
	if err != nil {
		return nil, err
	}

	redisCache := cache.NewRedis(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB, cfg.Redis.TTL)
	if pingErr := redisCache.Ping(ctx); pingErr != nil {
		logger.Warn("redis ping failed; thumbnails will use the in-memory cache only", zap.Error(pingErr))
	}
	lru := cache.NewLRU(cfg.Thumbnail.LRUBytes)
	twoLevel := cache.NewTwoLevel(lru, redisCache)

	mtr := metrics.New()
	generator := thumbnail.NewGenerator(cfg.Thumbnail.MaxConcurrency)
	jwtSvc := jwt.New(cfg.App.JWTSecret)

	userRepo, err := users.NewRepository(users.DefaultCredentials())
	if err != nil {
		return nil, err
	}

	photoRepo := sqlitephoto.NewRepository(db)
	photoSvc := photoapp.NewService(photoRepo, store, cfg.MinIO.PresignTTL, cfg.App.PublicBaseURL)
	thumbSvc := thumbapp.NewService(twoLevel, store, generator, mtr)
	authSvc := authapp.NewService(userRepo, jwtSvc, cfg.App.JWTTTL)

	server := rest.NewServer(rest.ServerDeps{
		Logger:     logger,
		Metrics:    mtr,
		JWT:        jwtSvc,
		Auth:       rest.NewAuthController(logger, authSvc),
		Photos:     rest.NewPhotoController(logger, photoSvc, store),
		Thumbnails: rest.NewThumbnailController(logger, thumbSvc, cfg.Thumbnail.CacheMaxAge),
		Ops:        rest.NewOpsController(mtr),
		BodyLimit:  cfg.App.BodyLimit,
	})

	return &App{logger: logger, cfg: cfg, db: db, redis: redisCache, server: server}, nil
}

// Run starts the HTTP server and blocks until a shutdown signal is received.
func (a *App) Run(ctx context.Context) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGUSR1)
	defer stop()

	addr := a.cfg.App.Host + ":" + a.cfg.App.Port

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		a.logger.Info("starting scout", zap.String("addr", addr), zap.String("env", a.cfg.App.Env))
		if err := a.server.App().Listen(addr); err != nil && !errors.Is(err, context.Canceled) {
			return err
		}
		return nil
	})

	<-ctx.Done()
	a.logger.Info("shutting down scout gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := a.server.App().ShutdownWithContext(shutdownCtx); err != nil {
		a.logger.Error("graceful shutdown error", zap.Error(err))
	}

	if err := g.Wait(); err != nil {
		return err
	}
	a.logger.Info("scout stopped")
	return nil
}

// Server exposes the HTTP server (used by integration tests).
func (a *App) Server() *rest.Server { return a.server }

// Close releases resources.
func (a *App) Close() {
	if a.db != nil {
		_ = a.db.Close()
	}
	if a.redis != nil {
		_ = a.redis.Close()
	}
	if a.logger != nil {
		_ = a.logger.Sync()
	}
}
