package core

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/sirupsen/logrus"
	"github.com/videocoin/cloud-autoscaler/metrics"
	"github.com/videocoin/cloud-autoscaler/types"
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
    stdin: false
    tty: false
  restartPolicy: Always
`

func (s *AutoScaler) ScaleUp(rule types.Rule, count uint) error {
	s.logger.WithField("machine_type", rule.Instance.MachineType).Info("scaling up")

	floatCount := float64(count)

	m := s.Metrics.Instances
	m.WithLabelValues(metrics.InstanceStatusCreating, rule.Instance.MachineType).Sub(-1 * floatCount)
	defer m.WithLabelValues(metrics.InstanceStatusCreating, rule.Instance.MachineType).Sub(floatCount)

	var wg sync.WaitGroup
	c := count

	for {
		wg.Add(1)
		go func() error {
			defer wg.Done()
			return s.createInstance(rule)
		}()

		c--
		if c <= 0 {
			break
		}
	}

	wg.Wait()

	s.logger.WithField("count", count).Info("all instances have been created")

	return nil
}

func (s *AutoScaler) ScaleDown(rule types.Rule, instanceName string) error {
	s.logger.Info("scaling down")

	go s.removeInstance(rule, instanceName)

	return nil
}

func (s *AutoScaler) createInstance(rule types.Rule) error {
	disks := []*computev1.AttachedDisk{
		&computev1.AttachedDisk{
			AutoDelete: true,
			Boot:       true,
			InitializeParams: &computev1.AttachedDiskInitializeParams{
				SourceImage: rule.Instance.SourceImage,
				DiskSizeGb:  rule.Instance.DiskSizeGb,
			},
		},
	}

	networkInterfaces := []*computev1.NetworkInterface{
		&computev1.NetworkInterface{
			Subnetwork: fmt.Sprintf(
				"projects/%s/regions/%s/subnetworks/%s",
				s.GCECfg.Project,
				s.GCECfg.Region,
				rule.Instance.Subnetwork,
			),
			AccessConfigs: []*computev1.AccessConfig{
				&computev1.AccessConfig{
					NetworkTier: "STANDARD",
				},
			},
		},
	}

	serviceAccounts := []*computev1.ServiceAccount{
		&computev1.ServiceAccount{
			Scopes: []string{
				"https://www.googleapis.com/auth/monitoring",
				"https://www.googleapis.com/auth/devstorage.full_control",
				"https://www.googleapis.com/auth/compute",
			},
		},
	}

	instanceName := fmt.Sprintf("transcoder-%s", randString(12))
	containerDecl := fmt.Sprintf(containerDeclTpl, instanceName, rule.Instance.DockerImage)
	instance := &computev1.Instance{
		Name:              instanceName,
		MachineType:       fmt.Sprintf("zones/%s/machineTypes/%s", s.GCECfg.Zone, rule.Instance.MachineType),
		Disks:             disks,
		NetworkInterfaces: networkInterfaces,
		ServiceAccounts:   serviceAccounts,
		Zone:              s.GCECfg.Zone,
		Metadata: &computev1.Metadata{
			Items: []*computev1.MetadataItems{
				&computev1.MetadataItems{
					Key:   "gce-container-declaration",
					Value: pointer.ToString(containerDecl),
				},
			},
		},
	}

	logger := s.logger.WithField("instance", instance.Name)

	_, err := s.compute.Instances.Insert(s.GCECfg.Project, s.GCECfg.Zone, instance).Do()
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Info("creating instance")

	for {
		newInstance, err := s.compute.Instances.Get(s.GCECfg.Project, s.GCECfg.Zone, instance.Name).Do()
		if err != nil {
			logger.WithFields(logrus.Fields{
				"project": s.GCECfg.Project,
				"zone":    s.GCECfg.Zone,
			}).Errorf("failed to get instance: %s", err.Error())
			return err
		}

		logger.WithFields(logrus.Fields{
			"name":   newInstance.Name,
			"status": newInstance.Status,
		}).Info("current status")

		if newInstance.Status == "RUNNING" {
			time.Sleep(time.Second * 60)
			break
		}

		time.Sleep(time.Second * 5)
	}

	return nil
}

func (s *AutoScaler) removeInstance(rule types.Rule, name string) error {
	logger := s.logger.WithField("instance", name)

	instance, err := s.compute.Instances.Get(s.GCECfg.Project, s.GCECfg.Zone, name).Do()
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	m := s.Metrics.Instances
	m.WithLabelValues(metrics.InstanceStatusRemoving, instance.MachineType).Inc()
	defer m.WithLabelValues(metrics.InstanceStatusRemoving, instance.MachineType).Dec()

	_, err = s.compute.Instances.Delete(s.GCECfg.Project, s.GCECfg.Zone, instance.Name).Do()
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	logger.Info("removing instance")

	c := 0
	for {
		instance, err := s.compute.Instances.Get(s.GCECfg.Project, s.GCECfg.Zone, name).Do()
		if err != nil {
			if strings.HasPrefix(err.Error(), "googleapi: Error 404:") {
				logger.Info("instance has been removed")
				break
			} else {
				logger.Error(err.Error())
				return err
			}
		}

		logger.WithField("status", instance.Status).Info("current status")

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
