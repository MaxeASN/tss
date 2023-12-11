package api

import (
	"context"
	"errors"
	"github.com/bnb-chain/tss/common"
	"github.com/bnb-chain/tss/task"
	"github.com/ethereum/go-ethereum/log"
)

func NewApiService(config *APIConfig, worker *task.Worker) *APIService {
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
type SignResponse struct {
	Result any
}

func (s *APIService) Sign(ctx context.Context, chainId int, address string, data string) (any, error) {
	log.Info("Received Sign request", "chainId", chainId, "address", address, "data", data, "\n")
	// genrerate the task
	event := &task.TaskEvent{
		Tid:     "",
		ChainId: chainId,
		Address: address,
		Message: data,
		Op:      common.SignType,
	}

	// generate task key
	err := s.Worker.GenEventKey(event)
	if err != nil {
		return nil, errors.New("message invalid, need hash with prefix \"0x\"")
	}

	// push the task to the queue
	resultCh, err := s.Worker.PushEvent(event)
	if err != nil {
		return nil, NewPushEventError("Push event error, " + err.Error())
	}
	// try to get the result
	//for {
	select {
	case <-ctx.Done():
		s.Worker.CancelTask(event.Tid)
		return nil, NewRequestCanceledError("Request cancelled")
	case res := <-resultCh:
		return &SignResponse{Result: res.Result}, res.Err
	}
	//}
	return nil, nil
}
