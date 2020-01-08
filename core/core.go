package core

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/videocoin/cloud-autoscaler/metrics"
	"github.com/videocoin/cloud-autoscaler/types"
	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
)

type AutoScaler struct {
	logger  *logrus.Entry
	compute *compute.Service
	Metrics *metrics.Metrics
	Rules   types.Rules
}

func NewAutoScaler(
	logger *logrus.Entry,
	metrics *metrics.Metrics,
	rules types.Rules,
) (*AutoScaler, error) {
	computeCli, err := google.DefaultClient(context.TODO(), compute.ComputeScope)
	if err != nil {
		return nil, err
	}

	computeSvc, err := compute.New(computeCli)
	if err != nil {
		return nil, err
	}

	return &AutoScaler{
		logger:  logger,
		compute: computeSvc,
		Metrics: metrics,
		Rules:   rules,
	}, nil
}
