// Package contract is the published port of the api module: the DTOs of its
// services and the typed contracts they are registered and called through. It
// is the only package of the api module that other code may import.
//
// It imports only the standard library and mono/pkg/helper. The subjects
// services.api.get-data, services.api.create-order and services.api.get-status
// are what demo.sh and the nats CLI examples use, and the service names are
// the keys of the per-service limits in main.go, so they must not change.
package contract

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/go-monolith/mono/pkg/helper"
)

// GetDataRequest is the (empty) request of the get-data service.
type GetDataRequest struct{}

// DataResponse is the response of the get-data service.
type DataResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Value     int       `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

// OrderRequest is the request of the create-order service.
type OrderRequest struct {
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

// OrderResponse is the response of the create-order service.
type OrderResponse struct {
	OrderID   string    `json:"order_id"`
	ProductID string    `json:"product_id"`
	Quantity  int       `json:"quantity"`
	Total     float64   `json:"total"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// GetStatusRequest is the (empty) request of the get-status service.
type GetStatusRequest struct{}

// StatusResponse is the response of the get-status service.
type StatusResponse struct {
	Service   string    `json:"service"`
	Status    string    `json:"status"`
	Uptime    string    `json:"uptime"`
	Timestamp time.Time `json:"timestamp"`
}

// GetData is the request-reply service on services.api.get-data
// (default limit: 100 requests per minute per client).
var GetData = helper.RequestReplyService[GetDataRequest, DataResponse]("api", "get-data").
	WithUnmarshaler(unmarshalOptional)

// CreateOrder is the request-reply service on services.api.create-order
// (50 requests per minute per client).
var CreateOrder = helper.RequestReplyService[OrderRequest, OrderResponse]("api", "create-order")

// GetStatus is the request-reply service on services.api.get-status
// (200 requests per minute per client).
var GetStatus = helper.RequestReplyService[GetStatusRequest, StatusResponse]("api", "get-status").
	WithUnmarshaler(unmarshalOptional)

// unmarshalOptional is JSON that also accepts an empty body, so the
// parameterless services answer `nats request <subject> ""` as well as '{}'.
func unmarshalOptional(data []byte, v any) error {
	if len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	return json.Unmarshal(data, v)
}
