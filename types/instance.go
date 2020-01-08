package types

type Instance struct {
	MachineType string `yaml:"machineType"`
	Subnetwork  string `yaml:"subnetwork"`
	DiskSizeGb  int64  `yaml:"diskSizeGb"`
	SourceImage string `yaml:"sourceImage"`
	DockerImage string `yaml:"dockerImage"`
}
