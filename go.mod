module github.com/videocoin/cloud-autoscaler

go 1.14

require (
	cloud.google.com/go v0.56.0
	github.com/AlekSi/pointer v1.1.0
	github.com/ghodss/yaml v1.0.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/labstack/echo v3.3.10+incompatible
	github.com/prometheus/alertmanager v0.20.0
	github.com/prometheus/client_golang v1.5.1
	github.com/sirupsen/logrus v1.5.0
	github.com/videocoin/cloud-api v1.0.0
	github.com/videocoin/cloud-pkg v1.0.0
	google.golang.org/api v0.22.0
)

replace github.com/videocoin/cloud-pkg => ../cloud-pkg

replace github.com/videocoin/cloud-api => ../cloud-api
