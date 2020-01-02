package types

import "errors"

var (
	ErrRuleNotFound = errors.New("rule not found")
)

const (
	ScaleUp   = "up"
	ScaleDown = "down"
)

type Rule struct {
	AlertName string `yaml:"alertname"`
	Scale     string `yaml:"scale"`
	Project   string `yaml:"project"`
	Zone      string `yaml:"zone"`
	Template  string `template:"template"`
}

func (r *Rule) IsScaleUp() bool {
	return r.Scale == ScaleUp
}

func (r *Rule) IsScaleDown() bool {
	return r.Scale == ScaleDown
}

type Rules []*Rule

type AutoScaleRules struct {
	Rules Rules `yaml:"rules"`
}

func GetRule(rules Rules, alertname string) (*Rule, error) {
	for _, rule := range rules {
		if rule.AlertName == alertname {
			return rule, nil
		}
	}

	return nil, ErrRuleNotFound
}
