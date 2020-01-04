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
	ConsulAddr string `default:"127.0.0.1:8500" envconfig:"CONSUL_ADDR"`
	RulesPath  string `default:"rules.yml"`
}
