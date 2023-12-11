package task

import (
	flag "github.com/spf13/pflag"
)

const DefaultWorkerQueueSize = 10
const DefaultWorkerTimeout = "10s"

type WorkerConfig struct {
	WorkerLimit int `koanf:"limit"`
}

var DefaultWorkerConfig = WorkerConfig{
	WorkerLimit: DefaultWorkerQueueSize,
}

func WorkerConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Int(prefix+".limit", DefaultWorkerConfig.WorkerLimit, "Number of workers")
}
