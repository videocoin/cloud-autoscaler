package sd

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strings"

	consulapi "github.com/hashicorp/consul/api"
)

var (
	ErrAlreadyLocked = errors.New("already locked")
)

type Client struct {
	cli *consulapi.Client
}

func NewClient(addr string) (*Client, error) {
	cfg := consulapi.DefaultConfig()
	cfg.Address = addr

	consul, err := consulapi.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("consul: failed to create client: %s", err)
	}

	_, err = consul.Agent().Self()
	if err != nil {
		return nil, fmt.Errorf("consul: failed to connect agent: %s", err)
	}

	return &Client{cli: consul}, nil
}

func (c *Client) Register(name string, host string, port int) (*consulapi.AgentService, error) {
	hn, _ := os.Hostname()
	hnTag := fmt.Sprintf("host:%s", hn)

	reg := &consulapi.AgentServiceRegistration{
		ID:      fmt.Sprintf("%s:%d", name, port),
		Name:    name,
		Address: host,
		Port:    port,
		Tags:    []string{hnTag},
	}

	agent := c.cli.Agent()

	err := agent.ServiceRegister(reg)
	if err != nil {
		return nil, err
	}

	checkReg := &consulapi.AgentCheckRegistration{
		ID:        fmt.Sprintf("%s:health", reg.ID),
		Name:      fmt.Sprintf("%s Health Check", strings.Title(name)),
		Notes:     "Health check",
		ServiceID: reg.ID,
	}
	checkReg.AgentServiceCheck = consulapi.AgentServiceCheck{
		HTTP:     fmt.Sprintf("http://%s:%d/healthz", host, port),
		Interval: "5s",
	}

	err = agent.CheckRegister(checkReg)
	if err != nil {
		return nil, err
	}

	services, err := agent.Services()
	if err != nil {
		return nil, err
	}

	service, ok := services[reg.ID]
	if !ok {
		return nil, errors.New("service is not registered")
	}

	return service, nil
}

func (c *Client) Deregister(id string) error {
	err := c.cli.Agent().ServiceDeregister(id)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) GetServices(
	name, tag string,
	passingOnly bool,
	q *consulapi.QueryOptions,
) ([]*consulapi.ServiceEntry, *consulapi.QueryMeta, error) {
	return c.cli.Health().Service(name, tag, passingOnly, q)
}

func (c *Client) GetService(
	name string,
	tag string,
	passingOnly bool,
	q *consulapi.QueryOptions,
	random bool,
) (*consulapi.ServiceEntry, error) {
	services, _, err := c.cli.Health().Service(name, tag, passingOnly, q)
	if err != nil {
		return nil, err
	}

	if len(services) == 0 {
		return nil, errors.New("service not found")
	}

	if random {
		return services[rand.Intn(len(services))], nil
	}

	return services[0], nil
}

func (c *Client) GetServiceLockKeys(serviceName string) []string {
	lockKeys := []string{}

	keys, _, _ := c.cli.KV().Keys("/service", "", nil)
	for _, key := range keys {
		prefix := fmt.Sprintf("service/%s:", serviceName)
		suffix := ".lock"
		if strings.HasPrefix(key, prefix) && strings.HasSuffix(key, suffix) {
			lockKeys = append(lockKeys, key)
		}
	}

	return lockKeys
}

func (c *Client) KeyIsLocked(key string) (bool, error) {
	isLocked := false

	lockPair, _, err := c.cli.KV().Get(key, nil)
	if err != nil {
		return false, err
	}

	if lockPair != nil {
		if lockPair.Session != "" {
			isLocked = true
		}
	}

	return isLocked, nil
}
