package autoscaler

type App struct {
	cfg *Config
	srv *Server
}

func NewApp(cfg *Config) (*App, error) {
	gceConfig := &GCEConfig{
		Env:             cfg.ClusterEnv,
		WorkerSentryDSN: cfg.WorkerSentryDSN,
		MaxCount:        cfg.MaxTranscodersCount,
		LokiURL:         cfg.LokiURL,
		DispatcherAddr:  cfg.DispatcherAddr,
		APIKey:          cfg.GCEApiKey,
	}

	metrics := NewMetrics(cfg.Name, cfg.Rules)
	err := metrics.RegisterAll()
	if err != nil {
		return nil, err
	}

	scaler, err := NewAutoScaler(cfg.Logger, metrics, cfg.Rules, gceConfig)
	if err != nil {
		return nil, err
	}

	srvCfg := &ServerConfig{
		Name:    cfg.Name,
		Version: cfg.Version,
		Addr:    cfg.Addr,
	}
	srv := NewServer(srvCfg, cfg.Logger, scaler)

	s := &App{
		cfg: cfg,
		srv: srv,
	}

	return s, nil
}

func (app *App) Start(errCh chan error) {
	go func() {
		app.cfg.Logger.Info("starting api server")
		errCh <- app.srv.Start()
	}()
}

func (app *App) Stop() error {
	return nil
}
