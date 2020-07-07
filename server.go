package autoscaler

import (
	"encoding/json"
	"github.com/prometheus/alertmanager/notify/webhook"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

type ServerConfig struct {
	Name    string
	Version string
	Addr    string
}

type Server struct {
	cfg        *ServerConfig
	logger     *logrus.Entry
	AutoScaler *AutoScaler
}

func NewServer(cfg *ServerConfig, logger *logrus.Entry, as *AutoScaler) *Server {
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

func (s *Server) prometheusWebhook(c echo.Context) error {
	body, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return err
	}
	defer c.Request().Body.Close()

	msg := new(webhook.Message)
	err = json.Unmarshal(body, &msg)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	s.logger.Debugf("webhook message alerts: %+v", msg.Alerts)

	err = validatePromWebhookMessage(msg)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	for _, alert := range msg.Alerts.Firing() {
		rule, err := GetRule(s.AutoScaler.Rules, alert.Labels["alertname"])
		if err != nil {
			s.logger.Warningf("rule %s not found", alert.Labels["alertname"])
			break
		}

		if alert.Labels["machine_type"] != "" {
			rule.Instance.MachineType = alert.Labels["machine_type"]
		}

		count, _ := strconv.ParseUint(alert.Annotations["count"], 0, 32)
		if count > 0 {
			if rule.IsScaleUp() {
				go func() {
					err := s.AutoScaler.ScaleUp(*rule, uint(count))
					if err != nil {
						s.logger.Error(err)
					}
				}()
			}

			if rule.IsScaleDown() {
				if strings.HasPrefix(alert.Labels["hostname"], "transcoder-") {
					go func() {
						err := s.AutoScaler.ScaleDown(*rule, alert.Labels["hostname"])
						if err != nil {
							s.logger.Error(err)
						}
					}()
				}
			}
		}
	}

	return c.JSON(http.StatusOK, nil)
}

func validatePromWebhookMessage(m *webhook.Message) error {
	return nil
}
