package api

import "time"

// DataResponse is the response for api.getData service.
type DataResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Value     int       `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

// OrderRequest is the request for api.createOrder service.
type OrderRequest struct {
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

// OrderResponse is the response for api.createOrder service.
type OrderResponse struct {
	OrderID   string    `json:"order_id"`
	ProductID string    `json:"product_id"`
	Quantity  int       `json:"quantity"`
	Total     float64   `json:"total"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// StatusResponse is the response for api.getStatus service.
type StatusResponse struct {
	Service   string    `json:"service"`
	Status    string    `json:"status"`
	Uptime    string    `json:"uptime"`
	Timestamp time.Time `json:"timestamp"`
}
