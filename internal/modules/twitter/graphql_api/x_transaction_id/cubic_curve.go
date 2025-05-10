package x_transaction_id

import "math"

// Cubic implements a parametric cubic-bezier evaluator.
type Cubic struct {
	curves []float64
}

// NewCubic constructs a Cubic from four control-point values.
func NewCubic(curves []float64) *Cubic {
	return &Cubic{curves: curves}
}

// GetValue returns the bezier output at parameter t in [0,1].
func (c *Cubic) GetValue(t float64) float64 {
	start, end, mid := 0.0, 1.0, 0.0
	var startGrad, endGrad float64

	// before 0
	if t <= 0.0 {
		if c.curves[0] > 0 {
			startGrad = c.curves[1] / c.curves[0]
		} else if c.curves[1] == 0 && c.curves[2] > 0 {
			startGrad = c.curves[3] / c.curves[2]
		}
		return startGrad * t
	}
	// after 1
	if t >= 1.0 {
		if c.curves[2] < 1 {
			endGrad = (c.curves[3] - 1.0) / (c.curves[2] - 1.0)
		} else if c.curves[2] == 1 && c.curves[0] < 1 {
			endGrad = (c.curves[1] - 1.0) / (c.curves[0] - 1.0)
		}
		return 1.0 + endGrad*(t-1.0)
	}

	// binary search for x that matches t
	for start < end {
		mid = (start + end) / 2
		xEst := calculate(c.curves[0], c.curves[2], mid)
		if math.Abs(t-xEst) < 0.00001 {
			return calculate(c.curves[1], c.curves[3], mid)
		}
		if xEst < t {
			start = mid
		} else {
			end = mid
		}
	}
	return calculate(c.curves[1], c.curves[3], mid)
}

// calculate evaluates the Bernstein basis for a single dimension.
func calculate(a, b, m float64) float64 {
	return 3*a*(1-m)*(1-m)*m + 3*b*(1-m)*m*m + m*m*m
}
