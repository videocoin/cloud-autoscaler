package autoscaler

import (
	"github.com/sirupsen/logrus"
)

type Config struct {
	Logger  *logrus.Entry `envconfig:"-"`
	Name    string        `envconfig:"-"`
	Version string        `envconfig:"-"`
	Rules   Rules         `envconfig:"-"`

	Addr                string `envconfig:"ADDR" default:"0.0.0.0:5030"`
	RulesPath           string `envconfig:"RULES_PATH" default:"rules.yml"`
	ClusterEnv          string `envconfig:"CLUSTER_ENV" default:"dev"`
	DispatcherAddr      string `envconfig:"DISPATCHER_ADDR" required:"true"`
	GCESA               string `envconfig:"GCE_SA" required:"true"`
	GCEProject          string `envconfig:"GCE_PROJECT" required:"true"`
	GCERegion           string `envconfig:"GCE_REGION" required:"true"`
	GCEZone             string `envconfig:"GCE_ZONE" required:"true"`
	MaxTranscodersCount int    `envconfig:"MAX_WORKERS" default:"20"`
	LokiURL             string `envconfig:"LOKI_URL"`
	WorkerSentryDSN     string `envconfig:"WORKER_SENTRY_DSN"`
}

type GCEConfig struct {
	Env             string
	WorkerSentryDSN string
	MaxCount        int
	LokiURL         string
	DispatcherAddr  string
	SA              string
	Project         string
	Region          string
	Zone            string
}
