// Package contract is the published port of the inventory module: the DTOs of
// its services and the typed contracts other modules call them through. It is
// the ONLY package of the inventory module another module may import.
//
// Rule: a contract package imports only the standard library,
// github.com/go-monolith/mono/pkg/types and .../pkg/helper. Never its own
// module package, never another module's non-contract package.
package contract

import "github.com/go-monolith/mono/pkg/helper"

// CheckStockRequest asks whether Quantity units of ProductID are available.
type CheckStockRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

// CheckStockResponse reports availability and the current stock level.
type CheckStockResponse struct {
	Available bool `json:"available"`
	Stock     int  `json:"stock"`
}

// CheckStock is declared once: the provider registers through it, consumers
// call through it. Name, subject, kind and JSON encoding live here only.
var CheckStock = helper.RequestReplyService[CheckStockRequest, CheckStockResponse]("inventory", "check-stock")
