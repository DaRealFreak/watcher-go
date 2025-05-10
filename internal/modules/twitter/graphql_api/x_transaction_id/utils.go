package x_transaction_id

import (
	"encoding/base64"
	"math"
	"strings"
)

// solve replicates Python's value*(max-min)/255+min, with an optional floor.
func solve(value, minVal, maxVal float64, rounding bool) float64 {
	res := value*(maxVal-minVal)/255 + minVal
	if rounding {
		return math.Floor(res)
	}
	return math.Round(res*100) / 100
}

// FloatToHex returns a hex representation of `x`, matching the Python algorithm.
// e.g., 0.47695… → ".7AE147AE147AE", 1.25 → "1.4"
func FloatToHex(x float64) string {
	// Handle negative by sign‐agnostic conversion
	neg := x < 0
	if neg {
		x = -x
	}

	// Split integer and fractional parts
	integerPart := int64(x)
	fraction := x - float64(integerPart)

	// Convert integer part
	var result []rune
	if integerPart > 0 {
		// build digits in reverse
		var rev []rune
		q := integerPart
		for q > 0 {
			rem := q % 16
			var c rune
			if rem > 9 {
				c = rune('A' + (rem - 10))
			} else {
				c = rune('0' + rem)
			}
			rev = append(rev, c)
			q /= 16
		}
		// reverse into result
		for i := len(rev) - 1; i >= 0; i-- {
			result = append(result, rev[i])
		}
	}

	// If no integer digits, Python returns just fraction (leading dot), so leave result empty.

	// If there's no fraction, we’re done
	if fraction == 0 {
		if neg {
			return "-" + string(result)
		}
		return string(result)
	}

	// Otherwise append dot and fraction digits
	result = append(result, '.')
	f := fraction
	// Loop until fraction is zero (or you may choose to cap iterations if necessary)
	for f > 0 {
		f *= 16
		digit := int64(f)
		f -= float64(digit)
		var c rune
		if digit > 9 {
			c = rune('A' + (digit - 10))
		} else {
			c = rune('0' + digit)
		}
		result = append(result, c)
	}

	out := string(result)
	if neg {
		return "-" + out
	}
	return out
}

// Base64Encode returns standard base64 without padding.
func Base64Encode(data []byte) string {
	s := base64.StdEncoding.EncodeToString(data)
	return strings.TrimRight(s, "=")
}
