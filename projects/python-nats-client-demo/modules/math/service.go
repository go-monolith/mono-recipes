package math

import (
	"context"
	"errors"
	gomath "math"

	"github.com/go-monolith/mono"

	"github.com/example/python-nats-client-demo/modules/math/contract"
)

// Validation errors.
var (
	errInvalidOperation = errors.New("invalid operation")
	errDivisionByZero   = errors.New("division by zero")
	errNegativeSqrt     = errors.New("cannot calculate square root of negative number")
)

// calculate handles the math.calculate service request.
func (m *MathModule) calculate(_ context.Context, req contract.CalculateRequest, _ *mono.Msg) (contract.CalculateResponse, error) {
	result, err := performOperation(req.Operation, req.A, req.B)
	if err != nil {
		return contract.CalculateResponse{
			Operation: req.Operation,
			Error:     err.Error(),
		}, nil // Return error in response, not as Go error
	}

	return contract.CalculateResponse{
		Result:    result,
		Operation: req.Operation,
	}, nil
}

// performOperation executes the math operation.
func performOperation(op contract.Operation, a, b float64) (float64, error) {
	switch op {
	case contract.OpAdd:
		return a + b, nil
	case contract.OpSubtract:
		return a - b, nil
	case contract.OpMultiply:
		return a * b, nil
	case contract.OpDivide:
		if b == 0 {
			return 0, errDivisionByZero
		}
		return a / b, nil
	case contract.OpPower:
		return gomath.Pow(a, b), nil
	case contract.OpSqrt:
		if a < 0 {
			return 0, errNegativeSqrt
		}
		return gomath.Sqrt(a), nil
	default:
		return 0, errInvalidOperation
	}
}
