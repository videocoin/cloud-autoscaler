package types

type GCEConfig struct {
	Project         string
	Region          string
	Zone            string
	Env             string
	WorkerSentryDSN string
	UsePreemtible   bool
	MaxCount        int
	LokiURL         string
}
