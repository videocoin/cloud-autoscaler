package service

import (
	consulapi "github.com/hashicorp/consul/api"
	"github.com/videocoin/cloud-autoscaler/api"
	"github.com/videocoin/cloud-autoscaler/core"
	"github.com/videocoin/cloud-autoscaler/metrics"
	"github.com/videocoin/cloud-autoscaler/pkg/sd"
)

type Service struct {
	cfg       *Config
	sd        *sd.Client
	apiServer *api.Server
	info      *consulapi.AgentService
}

func NewService(cfg *Config) (*Service, error) {
	sdcli, err := sd.NewClient(cfg.ConsulAddr)
	if err != nil {
		return nil, err
	}

	metrics := metrics.NewMetrics(cfg.Name, cfg.Rules)
	err = metrics.RegisterAll()
	if err != nil {
		return nil, err
	}

	autoscaler, err := core.NewAutoScaler(cfg.Logger, sdcli, metrics, cfg.Rules)
	if err != nil {
		return nil, err
	}

	apiServerCfg := &api.ServerConfig{
		Name:    cfg.Name,
		Version: cfg.Version,
		Addr:    cfg.Addr,
	}
	apiServer := api.NewServer(apiServerCfg, cfg.Logger, autoscaler)

	s := &Service{
		cfg:       cfg,
		sd:        sdcli,
		apiServer: apiServer,
	}

	return s, nil
}

func (s *Service) Init() error {
	return nil
}

func (s *Service) Start() error {
	s.cfg.Logger.Info("starting api server")
	go s.apiServer.Start()

	return nil
}

func (s *Service) Stop() error {
	return nil
}
