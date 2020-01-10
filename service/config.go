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

	Addr       string `default:"0.0.0.0:5030" envconfig:"ADDR"`
	RulesPath  string `default:"rules.yml"`
	ClusterEnv string `default:"dev" envconfig:"CLUSTER_ENV"`
}
