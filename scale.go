package autoscaler

import (
	"context"
	"fmt"
	"google.golang.org/api/option"
	"strings"
	"sync"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/sirupsen/logrus"
	dispatcherv1 "github.com/videocoin/cloud-api/dispatcher/v1"
	computev1 "google.golang.org/api/compute/v1"
)

var containerDeclTpl = `
spec:
  containers:
  - name: %s
    image: '%s'
    env:
      - name: INTERNAL
        value: '1'
      - name: LOGLEVEL
        value: 'debug'
      - name: DISPATCHER_ADDR
        value: '%s'
      - name: WORKER_SENTRY_DSN
        value: '%s'
      - name: LOKI_URL
        value: '%s'
      - name: TASK_TYPE
        value: '%s'
    stdin: false
    tty: false
  restartPolicy: Always
`

type AutoScaler struct {
	logger  *logrus.Entry
	compute *computev1.Service
	Metrics *Metrics
	Rules   Rules
	GCECfg  *GCEConfig
}

func NewAutoScaler(
	logger *logrus.Entry,
	metrics *Metrics,
	rules Rules,
	gceCfg *GCEConfig,
) (*AutoScaler, error) {
	ctx := context.Background()
	computeSvc, err := computev1.NewService(ctx, option.WithCredentialsJSON([]byte(gceCfg.SA)))
	if err != nil {
		return nil, err
	}

	return &AutoScaler{
		logger:  logger,
		compute: computeSvc,
		Metrics: metrics,
		Rules:   rules,
		GCECfg:  gceCfg,
	}, nil
}

func (s *AutoScaler) ScaleUp(rule Rule, count uint) error {
	s.logger.WithField("machine_type", rule.Instance.MachineType).Info("scaling up")

	instances, err := s.compute.Instances.
		List(rule.Instance.Project, rule.Instance.Zone).
		Filter(fmt.Sprintf("(name=transcoder-%s-*) AND (status=RUNNING)", s.GCECfg.Env)).
		Do()
	if err != nil {
		s.logger.WithError(err).Error("failed to get transcoder instances")
	} else {
		if len(instances.Items) >= s.GCECfg.MaxCount {
			s.logger.Warning("max count of transcoders is already run")
			return nil
		}
	}

	floatCount := float64(count)

	m := s.Metrics.Instances
	m.WithLabelValues(InstanceStatusCreating, rule.Instance.MachineType).Add(floatCount)
	defer m.WithLabelValues(InstanceStatusCreating, rule.Instance.MachineType).Sub(floatCount)

	var wg sync.WaitGroup
	c := count

	for {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := s.createInstance(rule)
			if err != nil {
				s.logger.Error(err)
			}

		}()

		c--
		if c == 0 {
			break
		}
	}

	wg.Wait()

	s.logger.WithField("count", count).Info("all instances have been created")

	return nil
}

func (s *AutoScaler) ScaleDown(rule Rule, instanceName string) error {
	s.logger.Info("scaling down")

	go func() {
		err := s.removeInstance(rule, instanceName)
		if err != nil {
			s.logger.Error(err)
		}
	}()
	return nil
}

func (s *AutoScaler) createInstance(rule Rule) error {
	disks := []*computev1.AttachedDisk{
		{
			AutoDelete: true,
			Boot:       true,
			InitializeParams: &computev1.AttachedDiskInitializeParams{
				SourceImage: rule.Instance.SourceImage,
				DiskSizeGb:  rule.Instance.DiskSizeGb,
			},
		},
	}

	networkInterfaces := []*computev1.NetworkInterface{
		{
			Subnetwork: fmt.Sprintf(
				"projects/%s/regions/%s/subnetworks/%s",
				rule.Instance.Project,
				rule.Instance.Region,
				s.GCECfg.Env,
			),
			AccessConfigs: []*computev1.AccessConfig{
				{
					NetworkTier: "STANDARD",
				},
			},
		},
	}

	serviceAccounts := []*computev1.ServiceAccount{
		{
			Scopes: []string{
				"https://www.googleapis.com/auth/monitoring",
				"https://www.googleapis.com/auth/devstorage.full_control",
				"https://www.googleapis.com/auth/compute",
			},
		},
	}

	instanceName := fmt.Sprintf("transcoder-%s-%s", s.GCECfg.Env, randString(12))
	dockerImage := fmt.Sprintf("gcr.io/%s/worker:latest", rule.Instance.Project)
	taskType := dispatcherv1.TaskTypeVOD.String()
	if rule.Instance != nil && !rule.Instance.Preemtible {
		taskType = dispatcherv1.TaskTypeLive.String()
	}
	containerDecl := fmt.Sprintf(
		containerDeclTpl,
		instanceName,
		dockerImage,
		s.GCECfg.DispatcherAddr,
		s.GCECfg.WorkerSentryDSN,
		s.GCECfg.LokiURL,
		taskType,
	)
	instance := &computev1.Instance{
		Name:              instanceName,
		MachineType:       fmt.Sprintf("zones/%s/machineTypes/%s", rule.Instance.Zone, rule.Instance.MachineType),
		Disks:             disks,
		NetworkInterfaces: networkInterfaces,
		ServiceAccounts:   serviceAccounts,
		Zone:              rule.Instance.Zone,
		Metadata: &computev1.Metadata{
			Items: []*computev1.MetadataItems{
				{
					Key:   "gce-container-declaration",
					Value: pointer.ToString(containerDecl),
				},
				{
					Key:   "shutdown-script",
					Value: pointer.ToString("#! /bin/bash\n\ndocker container kill -s 2 $(docker ps -q)"),
				},
			},
		},
		Scheduling: &computev1.Scheduling{
			Preemptible: rule.Instance.Preemtible,
		},
	}

	logger := s.logger.WithField("instance", instance.Name)

	_, err := s.compute.Instances.Insert(rule.Instance.Project, rule.Instance.Zone, instance).Do()
	if err != nil {
		return err
	}

	logger.Info("creating instance")

	for {
		newInstance, err := s.compute.Instances.Get(rule.Instance.Project, rule.Instance.Zone, instance.Name).Do()
		if err != nil {
			return err
		}

		logger.WithFields(logrus.Fields{
			"name":   newInstance.Name,
			"status": newInstance.Status,
		}).Info("current status")

		if newInstance.Status == "STOPPING" ||
			newInstance.Status == "TERMINATED" {
			logger.WithField("name", newInstance.Name).Info("transcoder has been terminated")
			break
		}

		if newInstance.Status != "RUNNING" {
			time.Sleep(time.Second * 10)
			continue
		}

		time.Sleep(time.Second * 60 * 2)
		break
	}

	return nil
}

func (s *AutoScaler) removeInstance(rule Rule, name string) error {
	logger := s.logger.WithField("instance", name)

	instance, err := s.compute.Instances.Get(rule.Instance.Project, rule.Instance.Zone, name).Do()
	if err != nil {
		return err
	}

	m := s.Metrics.Instances
	m.WithLabelValues(InstanceStatusRemoving, instance.MachineType).Inc()
	defer m.WithLabelValues(InstanceStatusRemoving, instance.MachineType).Dec()

	_, err = s.compute.Instances.Delete(rule.Instance.Project, rule.Instance.Zone, instance.Name).Do()
	if err != nil {
		return err
	}

	logger.Info("removing instance")

	c := 0
	for {
		instance, err := s.compute.Instances.Get(rule.Instance.Project, rule.Instance.Zone, name).Do()
		if err != nil {
			if strings.HasPrefix(err.Error(), "googleapi: Error 404:") {
				logger.Info("instance has been removed")
				break
			} else {
				return err
			}
		}

		logger.WithField("status", instance.Status).Info("current status")

		if instance.Status == "STOPPING" {
			logger.Info("instance has been stopping")
			break
		}

		if instance.Status == "TERMINATED" {
			logger.Info("instance has been terminated")
			break
		}

		time.Sleep(time.Second * 5)

		c++

		if c > 10 {
			break
		}
	}

	return nil
}
