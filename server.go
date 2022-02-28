package server

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/betas-in/logger"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/ziflex/lecho/v3"
)

// Server ...
type Server struct {
	log                *logger.Logger
	app                *echo.Echo
	corsAllowedOrigins string
	port               int
	shutdownTimeout    time.Duration
}

// NewServer ...
func NewServer(logger *logger.Logger, port int, corsAllowedOrigins string, shutdownTimeout time.Duration) *Server {
	server := Server{}
	server.port = port
	server.corsAllowedOrigins = corsAllowedOrigins
	server.shutdownTimeout = shutdownTimeout
	server.log = logger

	server.app = echo.New()
	server.app.Logger = lecho.From(logger.Get())
	server.app.Use(middleware.Recover())
	server.app.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{server.corsAllowedOrigins},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowCredentials: true,
	}))
	// server.app.Use(middleware.CSRF())
	server.app.Use(middleware.RequestID())
	server.app.Use(middleware.Secure())
	server.app.Use(middleware.Logger())
	server.app.Use(lecho.Middleware(lecho.Config{
		Logger:  lecho.From(logger.Get()),
		NestKey: "request",
	}))
	server.app.HideBanner = true
	server.app.HidePort = true
	server.app.GET("/health", server.Health)

	return &server
}

// AddRoute ...
func (s *Server) AddRoute(verb, route string, f func(echo.Context) error) {
	switch verb {
	case "GET":
		s.app.GET(route, f)
	case "POST":
		s.app.POST(route, f)
	case "PUT":
		s.app.PUT(route, f)
	case "DELETE":
		s.app.DELETE(route, f)
	default:
		s.log.Error("server.addRoute").Msgf("The verb %s is not supported", verb)
	}
}

// Start ...
func (s *Server) Start() {
	go func() {
		s.app.Logger.Infof("Starting server at port %d", s.port)
		address := fmt.Sprintf(":%d", s.port)
		err := s.app.Start(address)
		if err != nil {
			s.app.Logger.Infof("Shutting down the server: %+v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()
	err := s.app.Shutdown(ctx)
	if err != nil {
		s.app.Logger.Fatal(err)
	}
}

// Health ...
func (s *Server) Health(c echo.Context) error {
	return c.String(200, "OK")
}
