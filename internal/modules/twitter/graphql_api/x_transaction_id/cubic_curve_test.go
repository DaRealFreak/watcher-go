package x_transaction_id

import (
	"math"
	"testing"
)

// almostEqual checks floating-point equality within a small epsilon.
func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-5
}

// TestCubicLinear verifies that a Cubic with control points [0,0,1,1]
// behaves like a linear function f(t) = t for all t.
func TestCubicLinear(t *testing.T) {
	c := NewCubic([]float64{0, 0, 1, 1})
	tests := []float64{-0.5, 0.0, 0.25, 0.5, 0.75, 1.0, 1.5}
	for _, tt := range tests {
		want := tt
		got := c.GetValue(tt)
		if !almostEqual(got, want) {
			t.Errorf("Cubic linear at t=%.2f: got %.6f, want %.6f", tt, got, want)
		}
	}
}

// TestCubicExtrapolation verifies the behavior for t < 0 and t > 1
// with non-linear control points.
func TestCubicExtrapolation(t *testing.T) {
	c := NewCubic([]float64{0.2, 0.4, 0.6, 0.8})
	// Before t=0: start_gradient = 0.4/0.2 = 2 => value = 2 * t
	if got := c.GetValue(-0.5); !almostEqual(got, -1.0) {
		t.Errorf("Cubic extrapolate before zero: got %.6f, want %.6f", got, -1.0)
	}
	// After t=1: end_gradient = (0.8-1)/(0.6-1) = 0.5 => value = 1 + 0.5*(t-1)
	if got := c.GetValue(2.0); !almostEqual(got, 1.5) {
		t.Errorf("Cubic extrapolate after one: got %.6f, want %.6f", got, 1.5)
	}
}

// TestCubicBezierMidpoint checks that a symmetric bezier curve
// (both control points on the diagonal) yields f(0.5) == 0.5.
func TestCubicBezierMidpoint(t *testing.T) {
	c := NewCubic([]float64{0.25, 0.25, 0.75, 0.75})
	if got := c.GetValue(0.5); !almostEqual(got, 0.5) {
		t.Errorf("Cubic symmetric at t=0.5: got %.6f, want %.6f", got, 0.5)
	}
}
