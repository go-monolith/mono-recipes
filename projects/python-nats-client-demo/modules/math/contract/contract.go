// Package contract is the published port of the math module: the DTOs of its
// services and the typed contracts they are registered and called through.
// It is the only package of the math module that other code may import.
//
// It imports only the standard library and mono/pkg/helper. The subject of
// every contract (services.<module>.<service>) is what the Python clients in
// client/ and demo.py use, so module and service names must not change.
package contract

import "github.com/go-monolith/mono/pkg/helper"

// Operation is a math operation type.
type Operation string

// Supported math operations.
const (
	OpAdd      Operation = "add"
	OpSubtract Operation = "subtract"
	OpMultiply Operation = "multiply"
	OpDivide   Operation = "divide"
	OpPower    Operation = "power"
	OpSqrt     Operation = "sqrt"
)

// CalculateRequest is the request for a math calculation.
type CalculateRequest struct {
	Operation Operation `json:"operation"`
	A         float64   `json:"a"`
	B         float64   `json:"b,omitempty"` // Optional for sqrt
}

// CalculateResponse is the response from a math calculation. A failed
// operation is reported in Error, not as a service error.
type CalculateResponse struct {
	Result    float64   `json:"result"`
	Operation Operation `json:"operation"`
	Error     string    `json:"error,omitempty"`
}

// Calculate is the request-reply service on services.math.calculate.
var Calculate = helper.RequestReplyService[CalculateRequest, CalculateResponse]("math", "calculate")
