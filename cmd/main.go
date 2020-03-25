package main

import (
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"

	"github.com/ghodss/yaml"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
	"github.com/videocoin/cloud-autoscaler/service"
	"github.com/videocoin/cloud-autoscaler/types"
	"github.com/videocoin/cloud-pkg/logger"
	"github.com/videocoin/cloud-pkg/tracer"
)

var (
	ServiceName string = "autoscaler"
	Version     string = "dev"
)

func main() {
	logger.Init(ServiceName, Version)

	log := logrus.NewEntry(logrus.New())
	log = logrus.WithFields(logrus.Fields{
		"service": ServiceName,
		"version": Version,
	})

	closer, err := tracer.NewTracer(ServiceName)
	if err != nil {
		log.Info(err.Error())
	} else {
		defer closer.Close()
	}

	cfg := &service.Config{
		Name:    ServiceName,
		Version: Version,
	}

	err = envconfig.Process(ServiceName, cfg)
	if err != nil {
		log.Fatal(err.Error())
	}

	rulesContent, err := ioutil.ReadFile(cfg.RulesPath)
	if err != nil {
		log.Fatal(err)
	}

	asRules := new(types.AutoScaleRules)
	err = yaml.Unmarshal(rulesContent, &asRules)
	if err != nil {
		log.Fatal(err)
	}

	cfg.Rules = asRules.Rules
	cfg.Logger = log

	svc, err := service.NewService(cfg)
	if err != nil {
		log.Fatal(err.Error())
	}

	signals := make(chan os.Signal, 1)
	exit := make(chan bool, 1)
	errCh := make(chan error, 1)

	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-signals

		log.Infof("recieved signal %s", sig)
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
