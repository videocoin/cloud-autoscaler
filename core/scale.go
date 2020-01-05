package core

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/videocoin/cloud-autoscaler/metrics"
	"github.com/videocoin/cloud-autoscaler/types"
	compute "google.golang.org/api/compute/v1"
)

func (s *AutoScaler) ScaleUp(rule types.Rule, count uint) error {
	s.logger.WithField("machine_type", rule.MachineType).Info("scaling up")

	floatCount := float64(count)

	m := s.Metrics.Instances
	m.WithLabelValues(metrics.InstanceStatusCreating, rule.MachineType).Sub(-1 * floatCount)
	defer m.WithLabelValues(metrics.InstanceStatusCreating, rule.MachineType).Sub(floatCount)

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

func (s *AutoScaler) ScaleDown(rule types.Rule, count uint) error {
	s.logger.Info("scaling down")

	services, _, err := s.sd.GetServices("transcoder", "", true, nil)
	if err != nil {
		s.logger.Error(err.Error())
		return err
	}

	c := count
	nodesForRemove := []string{}

	for _, service := range services {
		nodesForRemove = append(nodesForRemove, service.Node.Node)

		c--
		if c == 0 {
			break
		}
	}

	for _, nodeName := range nodesForRemove {
		go s.removeInstance(rule, nodeName)
	}

	return nil
}

func (s *AutoScaler) createInstance(rule types.Rule) error {
	s.logger.Info("getting instance template")

	template, err := s.compute.InstanceTemplates.Get(rule.Project, rule.Template).Do()
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"project":  rule.Project,
			"template": rule.Template,
		}).Errorf("failed to get instance template: %s", err.Error())
		return err
	}

	machineTypeURL := fmt.Sprintf(
		"zones/%s/machineTypes/%s",
		rule.Zone,
		template.Properties.MachineType,
	)

	var disks []*compute.AttachedDisk

	for _, disk := range template.Properties.Disks {
		disk.InitializeParams.DiskType = fmt.Sprintf(
			"projects/%s/zones/%s/diskTypes/%s",
			rule.Project,
			rule.Zone,
			disk.InitializeParams.DiskType,
		)

		disks = append(disks, disk)
	}

	instance := &compute.Instance{
		Name:              fmt.Sprintf("%s-%s", template.Name, randString(12)),
		Description:       template.Description,
		MachineType:       machineTypeURL,
		Disks:             disks,
		NetworkInterfaces: template.Properties.NetworkInterfaces,
		ServiceAccounts:   template.Properties.ServiceAccounts,
		Zone:              rule.Zone,
		Metadata:          template.Properties.Metadata,
	}

	ilogger := s.logger.WithField("instance", instance.Name)

	_, err = s.compute.Instances.Insert(rule.Project, rule.Zone, instance).Do()
	if err != nil {
		ilogger.Error(err)
		return err
	}

	ilogger.Info("instance creating")

	for {
		newInstance, err := s.compute.Instances.Get(rule.Project, rule.Zone, instance.Name).Do()
		if err != nil {
			ilogger.WithFields(logrus.Fields{
				"project": rule.Project,
				"zone":    rule.Zone,
			}).Errorf("failed to get instance: %s", err.Error())
			return err
		}

		ilogger.WithFields(logrus.Fields{
			"name":   newInstance.Name,
			"status": newInstance.Status,
		}).Info("current status")

		if newInstance.NetworkInterfaces[0].NetworkIP != "" {
			healthAddr := fmt.Sprintf(
				"http://%s:8011/healthz",
				newInstance.NetworkInterfaces[0].NetworkIP,
			)

			ilogger.WithFields(logrus.Fields{
				"name":        newInstance.Name,
				"health_addr": healthAddr,
			}).Info("getting health")

			resp, err := http.Get(healthAddr)
			if err != nil {
				ilogger.WithFields(logrus.Fields{
					"name":        newInstance.Name,
					"health_addr": healthAddr,
				}).Warnf("failed to get health: %s", err)

				time.Sleep(time.Second * 5)
				continue
			}

			if resp != nil && resp.StatusCode == 200 {
				ilogger.WithFields(logrus.Fields{
					"name":        newInstance.Name,
					"health_addr": healthAddr,
				}).Info("transcoder has been started")

				break
			}
		}

		time.Sleep(time.Second * 5)
	}

	return nil
}

func (s *AutoScaler) removeInstance(rule types.Rule, name string) error {
	ilogger := s.logger.WithField("instance", name)

	m := s.Metrics.Instances
	m.WithLabelValues(metrics.InstanceStatusRemoving).Inc()
	defer m.WithLabelValues(metrics.InstanceStatusRemoving).Dec()

	_, err := s.compute.Instances.Delete(rule.Project, rule.Zone, name).Do()
	if err != nil {
		ilogger.Error(err.Error())
		return err
	}

	c := 0
	for {
		instance, err := s.compute.Instances.Get(rule.Project, rule.Zone, name).Do()
		if err != nil {
			if strings.HasPrefix(err.Error(), "googleapi: Error 404:") {
				ilogger.Info("instance has been removed")
				break
			} else {
				ilogger.Error(err.Error())
				return err
			}
		}

		ilogger.WithField("status", instance.Status).Info("current status")

		if instance.Status == "TERMINATED" {
			ilogger.Info("instance has been terminated")
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
