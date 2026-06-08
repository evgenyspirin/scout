package rest

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"go.uber.org/zap"

	"scout/internal/infrastructure/jwt"
	"scout/internal/infrastructure/metrics"
	"scout/internal/interface/api/rest/middleware"
)

// ServerDeps are the dependencies required to wire the HTTP server.
type ServerDeps struct {
	Logger     *zap.Logger
	Metrics    *metrics.Metrics
	JWT        *jwt.Service
	Auth       *AuthController
	Photos     *PhotoController
	Thumbnails *ThumbnailController
	Ops        *OpsController
	BodyLimit  int
}

// Server wraps the configured Fiber application.
type Server struct {
	app *fiber.App
}

// NewServer builds the Fiber app, installs middleware and registers routes.
func NewServer(d ServerDeps) *Server {
	app := fiber.New(fiber.Config{
		AppName:               "scout",
		ErrorHandler:          ErrorHandler(d.Logger),
		BodyLimit:             d.BodyLimit,
		DisableStartupMessage: true,
	})

	// Global panic recovery so the process never crashes on a handler panic.
	app.Use(recover.New(recover.Config{EnableStackTrace: true}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,HEAD,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}))
	app.Use(middleware.Correlation())
	app.Use(middleware.RequestLogger(d.Logger, d.Metrics))

	// Public routes.
	app.Get(routeHealth, d.Ops.Health)
	app.Post(routeLogin, d.Auth.Login)

	// Authenticated routes (any valid user).
	authed := app.Group("", middleware.Auth(d.JWT))
	authed.Get(routePhotos, d.Photos.List)
	authed.Get(routePhoto, d.Photos.Get)
	authed.Get(routePhotoThumb, d.Thumbnails.Get)
	authed.Get(routePhotoOrig, d.Photos.Original)

	// Admin-only routes.
	admin := app.Group("", middleware.Auth(d.JWT), middleware.RequireAdmin())
	admin.Post(routeUploadLink, d.Photos.CreateUploadLink)
	admin.Head(routePhotoObject, d.Photos.HeadObject)
	admin.Get(routeMetrics, d.Ops.Metrics())
	admin.Get(routeDebugVars, d.Ops.DebugVars())

	return &Server{app: app}
}

// App returns the underlying Fiber app.
func (s *Server) App() *fiber.App { return s.app }
