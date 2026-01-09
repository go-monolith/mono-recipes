package math

import (
	"context"
	"errors"
	gomath "math"

	"github.com/go-monolith/mono"
)

// Validation errors.
var (
	errInvalidOperation = errors.New("invalid operation")
	errDivisionByZero   = errors.New("division by zero")
	errNegativeSqrt     = errors.New("cannot calculate square root of negative number")
)

// calculate handles the math.calculate service request.
func (m *MathModule) calculate(ctx context.Context, req CalculateRequest, _ *mono.Msg) (CalculateResponse, error) {
	result, err := performOperation(req.Operation, req.A, req.B)
	if err != nil {
		return CalculateResponse{
			Operation: req.Operation,
			Error:     err.Error(),
		}, nil // Return error in response, not as Go error
	}

	return CalculateResponse{
		Result:    result,
		Operation: req.Operation,
	}, nil
}

// performOperation executes the math operation.
func performOperation(op Operation, a, b float64) (float64, error) {
	switch op {
	case OpAdd:
		return a + b, nil
	case OpSubtract:
		return a - b, nil
	case OpMultiply:
		return a * b, nil
	case OpDivide:
		if b == 0 {
			return 0, errDivisionByZero
		}
		return a / b, nil
	case OpPower:
		return gomath.Pow(a, b), nil
	case OpSqrt:
		if a < 0 {
			return 0, errNegativeSqrt
		}
		return gomath.Sqrt(a), nil
	default:
		return 0, errInvalidOperation
	}
}
