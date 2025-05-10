package x_transaction_id

import "fmt"

// Interpolate linearly mixes two same-length slices element-wise.
func Interpolate(from, to []float64, f float64) ([]float64, error) {
	if len(from) != len(to) {
		return nil, fmt.Errorf("interpolate: mismatched lengths %d vs %d", len(from), len(to))
	}
	out := make([]float64, len(from))
	for i := range from {
		out[i] = from[i]*(1-f) + to[i]*f
	}
	return out, nil
}
