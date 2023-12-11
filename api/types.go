package api

import "github.com/bnb-chain/tss/task"

type APIService struct {
	Config *APIConfig
	Worker *task.Worker
}
