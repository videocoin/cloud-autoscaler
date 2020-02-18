package types

type Instance struct {
	MachineType string `yaml:"machineType"`
	DiskSizeGb  int64  `yaml:"diskSizeGb"`
	SourceImage string `yaml:"sourceImage"`
}
