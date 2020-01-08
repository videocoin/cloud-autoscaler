package types

type Instance struct {
	Project     string `yaml:"project"`
	Region      string `yaml:"region"`
	Zone        string `yaml:"zone"`
	MachineType string `yaml:"machineType"`
	Subnetwork  string `yaml:"subnetwork"`
	DiskSizeGb  int64  `yaml:"diskSizeGb"`
	SourceImage string `yaml:"sourceImage"`
	DockerImage string `yaml:"dockerImage"`
}
