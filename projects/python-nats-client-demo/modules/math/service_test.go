package math

import (
	"context"
	"testing"
)

func TestPerformOperation(t *testing.T) {
	tests := []struct {
		name      string
		op        Operation
		a         float64
		b         float64
		want      float64
		wantError error
	}{
		{
			name: "add two positive numbers",
			op:   OpAdd,
			a:    10,
			b:    5,
			want: 15,
		},
		{
			name: "add negative numbers",
			op:   OpAdd,
			a:    -10,
			b:    -5,
			want: -15,
		},
		{
			name: "subtract",
			op:   OpSubtract,
			a:    100,
			b:    42,
			want: 58,
		},
		{
			name: "subtract resulting in negative",
			op:   OpSubtract,
			a:    5,
			b:    10,
			want: -5,
		},
		{
			name: "multiply",
			op:   OpMultiply,
			a:    7,
			b:    8,
			want: 56,
		},
		{
			name: "multiply by zero",
			op:   OpMultiply,
			a:    100,
			b:    0,
			want: 0,
		},
		{
			name: "divide",
			op:   OpDivide,
			a:    100,
			b:    4,
			want: 25,
		},
		{
			name:      "divide by zero",
			op:        OpDivide,
			a:         10,
			b:         0,
			want:      0,
			wantError: errDivisionByZero,
		},
		{
			name: "power",
			op:   OpPower,
			a:    2,
			b:    10,
			want: 1024,
		},
		{
			name: "power of zero",
			op:   OpPower,
			a:    0,
			b:    5,
			want: 0,
		},
		{
			name: "sqrt positive",
			op:   OpSqrt,
			a:    144,
			b:    0,
			want: 12,
		},
		{
			name: "sqrt of zero",
			op:   OpSqrt,
			a:    0,
			b:    0,
			want: 0,
		},
		{
			name:      "sqrt negative",
			op:        OpSqrt,
			a:         -16,
			b:         0,
			want:      0,
			wantError: errNegativeSqrt,
		},
		{
			name:      "invalid operation",
			op:        "invalid",
			a:         1,
			b:         2,
			want:      0,
			wantError: errInvalidOperation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := performOperation(tt.op, tt.a, tt.b)

			if tt.wantError != nil {
				if err == nil {
					t.Errorf("performOperation() expected error %v, got nil", tt.wantError)
					return
				}
				if err != tt.wantError {
					t.Errorf("performOperation() error = %v, wantError = %v", err, tt.wantError)
				}
				return
			}

			if err != nil {
				t.Errorf("performOperation() unexpected error: %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("performOperation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateResponse(t *testing.T) {
	m := &MathModule{}

	tests := []struct {
		name        string
		req         CalculateRequest
		wantResult  float64
		wantError   string
		wantOp      Operation
	}{
		{
			name:       "successful add",
			req:        CalculateRequest{Operation: OpAdd, A: 5, B: 3},
			wantResult: 8,
			wantOp:     OpAdd,
		},
		{
			name:      "division by zero returns error in response",
			req:       CalculateRequest{Operation: OpDivide, A: 10, B: 0},
			wantError: "division by zero",
			wantOp:    OpDivide,
		},
		{
			name:      "invalid operation returns error in response",
			req:       CalculateRequest{Operation: "unknown", A: 1, B: 2},
			wantError: "invalid operation",
			wantOp:    "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := m.calculate(context.Background(), tt.req, nil)

			if err != nil {
				t.Errorf("calculate() returned Go error: %v", err)
				return
			}

			if resp.Operation != tt.wantOp {
				t.Errorf("calculate() operation = %v, want %v", resp.Operation, tt.wantOp)
			}

			if tt.wantError != "" {
				if resp.Error != tt.wantError {
					t.Errorf("calculate() error = %q, want %q", resp.Error, tt.wantError)
				}
			} else {
				if resp.Result != tt.wantResult {
					t.Errorf("calculate() result = %v, want %v", resp.Result, tt.wantResult)
				}
			}
		})
	}
}
