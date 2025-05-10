package x_transaction_id

import "math"

// ConvertRotationToMatrix returns the 2×2 rotation matrix flattened as [cos, -sin, sin, cos].
func ConvertRotationToMatrix(deg float64) []float64 {
	rad := deg * math.Pi / 180
	return []float64{
		math.Cos(rad), -math.Sin(rad),
		math.Sin(rad), math.Cos(rad),
	}
}
