package metrics

import (
	"context"

	"cloud.google.com/go/compute/metadata"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/videocoin/cloud-autoscaler/types"
	"golang.org/x/oauth2/google"
	computev1 "google.golang.org/api/compute/v1"
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
			[]string{"status", "machine_type"},
		),
	}
}

func (m *Metrics) RegisterAll() error {
	prometheus.MustRegister(m.Instances)

	if metadata.OnGCE() {
		project, err := metadata.ProjectID()
		if err != nil {
			return err
		}

		zone, err := metadata.Zone()
		if err != nil {
			return err
		}

		ctx := context.Background()
		gccli, err := google.DefaultClient(ctx, computev1.CloudPlatformScope)
		if err != nil {
			return err
		}
		computeService, err := computev1.New(gccli)
		if err != nil {
			return err
		}

		machineTypes := []string{}
		req := computeService.MachineTypes.List(project, zone)
		if err := req.Pages(ctx, func(page *computev1.MachineTypeList) error {
			for _, machineType := range page.Items {
				machineTypes = append(machineTypes, machineType.Name)
			}
			return nil
		}); err != nil {
			return err
		}

		for _, status := range InstanceStatuses {
			for _, mt := range machineTypes {
				m.Instances.WithLabelValues(status, mt).Set(0)
			}
			m.Instances.WithLabelValues(status, "").Set(0)
		}
	} else {
		for _, status := range InstanceStatuses {
			m.Instances.WithLabelValues(status, "").Set(0)
		}
	}

	return nil
}
