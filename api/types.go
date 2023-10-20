package api

import "github.com/bnb-chain/tss/worker"

type APIService struct {
	Config *APIConfig
	Worker *worker.Worker
}
