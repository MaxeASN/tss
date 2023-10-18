package api

import "context"

func NewApiService(config *APIConfig) *APIService {
	return &APIService{
		Config: config,
	}
}

type InitRequest struct{}

func (s *APIService) Init(ctx context.Context, req *InitRequest) error {
	return nil
}

type UpdateRequest struct{}

func (s *APIService) Update(ctx context.Context, req *UpdateRequest) error {
	return nil
}

type SignRequest struct{}

func (s *APIService) Sign(ctx context.Context, req *SignRequest) error {
	return nil
}
