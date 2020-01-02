package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/videocoin/cloud-autoscaler/types"
)

var (
	InstanceStatusCreating = "creating"
	InstanceStatusRemoving = "removing"

	InstanceStatuses = []string{
		InstanceStatusCreating, InstanceStatusRemoving,
	}
)

type Metrics struct {
	Instances *prometheus.GaugeVec

	rules types.Rules
}

func NewMetrics(namespace string, rules types.Rules) *Metrics {
	return &Metrics{
		rules: rules,
		Instances: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "instances",
				Help:      "Count of instances in progress",
			},
			[]string{"status"},
		),
	}
}

func (m *Metrics) RegisterAll() {
	prometheus.MustRegister(m.Instances)

	for _, status := range InstanceStatuses {
		m.Instances.WithLabelValues(status).Set(0)
	}
}
