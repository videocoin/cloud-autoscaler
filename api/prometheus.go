package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/labstack/echo"
	"github.com/prometheus/alertmanager/notify/webhook"
	"github.com/videocoin/cloud-autoscaler/types"
)

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
		rule, err := types.GetRule(s.AutoScaler.Rules, alert.Labels["alertname"])
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
				go s.AutoScaler.ScaleUp(*rule, uint(count))
			}

			if rule.IsScaleDown() {
				go s.AutoScaler.ScaleDown(*rule, alert.Labels["hostname"])
			}
		}
	}

	return c.JSON(http.StatusOK, nil)
}

func validatePromWebhookMessage(m *webhook.Message) error {
	return nil
}
