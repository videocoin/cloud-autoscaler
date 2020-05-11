package service

import (
	"cloud.google.com/go/compute/metadata"
	"github.com/videocoin/cloud-autoscaler/api"
	"github.com/videocoin/cloud-autoscaler/core"
	"github.com/videocoin/cloud-autoscaler/metrics"
	"github.com/videocoin/cloud-autoscaler/types"
)

type Service struct {
	cfg       *Config
	apiServer *api.Server
}

func NewService(cfg *Config) (*Service, error) {
	gceConfig := &types.GCEConfig{
		Env:             cfg.ClusterEnv,
		WorkerSentryDSN: cfg.WorkerSentryDSN,
		UsePreemtible:   cfg.UsePreemtible,
		MaxCount:        cfg.MaxTranscodersCount,
	}

	if metadata.OnGCE() {
		project, err := metadata.ProjectID()
		if err != nil {
			return nil, err
		}

		zone, err := metadata.Zone()
		if err != nil {
			return nil, err
		}

		region := zone[0 : len(zone)-2]

		gceConfig.Project = project
		gceConfig.Region = region
		gceConfig.Zone = zone
	}

	metrics := metrics.NewMetrics(cfg.Name, cfg.Rules, gceConfig)
	err := metrics.RegisterAll()
	if err != nil {
		return nil, err
	}

	autoscaler, err := core.NewAutoScaler(cfg.Logger, metrics, cfg.Rules, gceConfig)
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
		apiServer: apiServer,
	}

	return s, nil
}

func (s *Service) Start(errCh chan error) {
	go func() {
		s.cfg.Logger.Info("starting api server")
		errCh <- s.apiServer.Start()
	}()
}

func (s *Service) Stop() error {
	return nil
}
