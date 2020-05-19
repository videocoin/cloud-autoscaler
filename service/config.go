package service

import (
	"github.com/sirupsen/logrus"
	"github.com/videocoin/cloud-autoscaler/types"
)

type Config struct {
	Logger  *logrus.Entry `envconfig:"-"`
	Name    string        `envconfig:"-"`
	Version string        `envconfig:"-"`
	Rules   types.Rules   `envconfig:"-"`

	Addr                string `envconfig:"ADDR" default:"0.0.0.0:5030"`
	RulesPath           string `default:"rules.yml"`
	ClusterEnv          string `envconfig:"CLUSTER_ENV" default:"dev"`
	WorkerSentryDSN     string `envconfig:"WORKER_SENTRY_DSN"`
	UsePreemtible       bool   `default:"true"`
	MaxTranscodersCount int    `default:"20"`
	LokiURL             string `envconfig:"LOKI_URL"`
}
