package api

import (
	"context"
	"strconv"

	"github.com/bnb-chain/tss/worker"
	"github.com/ethereum/go-ethereum/log"
)

func NewApiService(config *APIConfig, worker *worker.Worker) *APIService {
	return &APIService{
		Config: config,
		Worker: worker,
	}
}

type InitRequest struct{}

func (s *APIService) Init(ctx context.Context, chainId int, layer2Address string) error {

	return nil
}

type UpdateRequest struct{}

func (s *APIService) Update(ctx context.Context, req *UpdateRequest) error {
	return nil
}

type SignRequest struct{}

func (s *APIService) Sign(ctx context.Context, chainId int, address string, data string) (any, error) {
	log.Info("Received Sign request", "chainId", chainId, "address", address, "data", data)
	// genrerate the task
	event := &worker.TaskEvent{
		Tid:     strconv.Itoa(chainId),
		ChainId: chainId,
		Address: address,
		Message: data,
		Op:      worker.Sign,
	}

	// push the task to the queue
	resultCh, err := s.Worker.PushEvent(event)
	if err != nil {
		return nil, NewPushEventError("Push event error")
	}
	// try to get the result
	for {
		select {
		case <-ctx.Done():
			s.Worker.ReleaseWorker(event.Tid)
			return nil, NewRequestCanceledError("Request cancelled")
		case res := <-resultCh:
			return res.Result, res.Err
		}
	}
	return nil, nil
}
