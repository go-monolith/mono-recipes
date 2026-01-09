package api

import (
	"context"
	"encoding/json"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/helper"
)

// ServiceAdapter provides type-safe access to API module services.
// Use this adapter when consuming API services from other modules.
type ServiceAdapter struct {
	container mono.ServiceContainer
}

// NewServiceAdapter creates a new API service adapter.
func NewServiceAdapter(container mono.ServiceContainer) *ServiceAdapter {
	return &ServiceAdapter{container: container}
}

// GetData calls the get-data service and returns typed response.
func (a *ServiceAdapter) GetData(ctx context.Context) (*DataResponse, error) {
	var resp DataResponse
	err := helper.CallRequestReplyService(
		ctx,
		a.container,
		ServiceGetData,
		json.Marshal,
		json.Unmarshal,
		struct{}{},
		&resp,
	)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateOrder calls the create-order service with the given request.
func (a *ServiceAdapter) CreateOrder(ctx context.Context, req *OrderRequest) (*OrderResponse, error) {
	var resp OrderResponse
	err := helper.CallRequestReplyService(
		ctx,
		a.container,
		ServiceCreateOrder,
		json.Marshal,
		json.Unmarshal,
		req,
		&resp,
	)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetStatus calls the get-status service and returns the service status.
func (a *ServiceAdapter) GetStatus(ctx context.Context) (*StatusResponse, error) {
	var resp StatusResponse
	err := helper.CallRequestReplyService(
		ctx,
		a.container,
		ServiceGetStatus,
		json.Marshal,
		json.Unmarshal,
		struct{}{},
		&resp,
	)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
