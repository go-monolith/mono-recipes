package math

// Operation represents a math operation type.
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

// CalculateResponse is the response from a math calculation.
type CalculateResponse struct {
	Result    float64   `json:"result"`
	Operation Operation `json:"operation"`
	Error     string    `json:"error,omitempty"`
}
