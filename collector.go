package autoscaler

import (
	"github.com/prometheus/client_golang/prometheus"
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
	rules     Rules
}

func NewMetrics(namespace string, rules Rules) *Metrics {
	return &Metrics{
		rules: rules,
		Instances: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "instances",
				Help:      "Count of instances in progress",
			},
			[]string{"status", "machine_type"},
		),
	}
}

func (m *Metrics) RegisterAll() error {
	prometheus.MustRegister(m.Instances)

	machineTypes := []string{"n1-standard-2", "n1-standard-4", "n1-standard-8"}
	for _, status := range InstanceStatuses {
		for _, mt := range machineTypes {
			m.Instances.WithLabelValues(status, mt).Set(0)
		}
		m.Instances.WithLabelValues(status, "").Set(0)
	}

	return nil
}
