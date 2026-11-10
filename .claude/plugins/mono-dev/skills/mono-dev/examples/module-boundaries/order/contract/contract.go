// Package contract is the published port of the order module: its service
// contracts, their DTOs, and the events it emits. Consumers of the event import
// it from here, so no shared "events" package is needed and no import cycle
// can arise.
package contract

import "github.com/go-monolith/mono/pkg/helper"

// PlaceOrderRequest asks to place an order.
type PlaceOrderRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

// PlaceOrderResponse reports the outcome.
type PlaceOrderResponse struct {
	OrderID string `json:"order_id"`
	Status  string `json:"status"` // "placed" or "out_of_stock"
}

// PlaceOrder is the request-reply service that places an order.
var PlaceOrder = helper.RequestReplyService[PlaceOrderRequest, PlaceOrderResponse]("order", "place-order")

// OrderPlacedEvent is published when an order was placed.
type OrderPlacedEvent struct {
	OrderID   string `json:"order_id"`
	ProductID string `json:"product_id"`
}

// OrderPlacedV1 is emitted by the order module.
var OrderPlacedV1 = helper.EventDefinition[OrderPlacedEvent]("order", "OrderPlaced", "v1")
