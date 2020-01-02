package api

import (
	"net/http"

	"github.com/labstack/echo"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/videocoin/cloud-autoscaler/core"
)

type ServerConfig struct {
	Name    string
	Version string
	Addr    string
}

type Server struct {
	cfg        *ServerConfig
	logger     *logrus.Entry
	AutoScaler *core.AutoScaler
}

func NewServer(cfg *ServerConfig, logger *logrus.Entry, as *core.AutoScaler) *Server {
	return &Server{
		cfg:        cfg,
		logger:     logger,
		AutoScaler: as,
	}
}

func (s *Server) Start() error {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	// e.Logger = logrusext.MWLogger{s.logger}
	// e.Use(logrusext.Hook())
	// e.Use(middleware.Recover())

	e.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"service": s.cfg.Name,
			"version": s.cfg.Version,
		})
	})

	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	e.POST("/prometheus/webhook", s.prometheusWebhook)

	return e.Start(s.cfg.Addr)
}
