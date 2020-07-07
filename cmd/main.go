package main

import (
	"github.com/ghodss/yaml"
	"github.com/kelseyhightower/envconfig"
	autoscaler "github.com/videocoin/cloud-autoscaler"
	"github.com/videocoin/cloud-pkg/logger"
	"github.com/videocoin/cloud-pkg/tracer"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
)

var (
	ServiceName string = "autoscaler"
	Version     string = "dev"
)

func main() {
	log := logger.NewLogrusLogger(ServiceName, Version, nil)

	closer, err := tracer.NewTracer(ServiceName)
	if err != nil {
		log.Info(err.Error())
	} else {
		defer closer.Close()
	}

	cfg := &autoscaler.Config{
		Name:    ServiceName,
		Version: Version,
		Logger:  log,
	}

	err = envconfig.Process(ServiceName, cfg)
	if err != nil {
		log.Fatal(err.Error())
	}

	rulesContent, err := ioutil.ReadFile(cfg.RulesPath)
	if err != nil {
		log.Fatal(err)
	}

	asRules := new(autoscaler.AutoScaleRules)
	err = yaml.Unmarshal(rulesContent, &asRules)
	if err != nil {
		log.Fatal(err)
	}

	cfg.Rules = asRules.Rules
	cfg.Logger = log

	svc, err := autoscaler.NewApp(cfg)
	if err != nil {
		log.Fatal(err.Error())
	}

	signals := make(chan os.Signal, 1)
	exit := make(chan bool, 1)
	errCh := make(chan error, 1)

	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-signals

		log.Infof("received signal %s", sig)
		exit <- true
	}()

	log.Info("starting")
	go svc.Start(errCh)

	select {
	case <-exit:
		break
	case err := <-errCh:
		if err != nil {
			log.Error(err)
		}
		break
	}

	log.Info("stopping")
	err = svc.Stop()
	if err != nil {
		log.Error(err)
		return
	}

	log.Info("stopped")
}
