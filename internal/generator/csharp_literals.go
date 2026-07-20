package generator

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
)

var (
	decimalDomainMin = mustRat("-79228162514264337593543950335")
	decimalDomainMax = mustRat("79228162514264337593543950335")
)

func numberLiteralFor(fieldType, value string) string {
	canonical := value
	if strings.ContainsAny(value, ".eE") && fieldType == "double" {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			canonical = strconv.FormatFloat(f, 'g', -1, 64)
		}
	}
	switch fieldType {
	case "decimal":
		return canonical + "m"
	case "double":
		return canonical + "d"
	case "long":
		return canonical + "L"
	default:
		return canonical
	}
}

func csharpStringLiteral(value string) string {
	var builder strings.Builder
	builder.WriteByte('"')
	for _, r := range value {
		switch r {
		case '"':
			builder.WriteString("\\\"")
		case '\\':
			builder.WriteString("\\\\")
		default:
			if r >= 0x20 && r <= 0x7e {
				builder.WriteRune(r)
			} else if r <= 0xffff {
				builder.WriteString(fmt.Sprintf("\\u%04X", r))
			} else {
				builder.WriteString(fmt.Sprintf("\\U%08X", r))
			}
		}
	}
	builder.WriteByte('"')
	return builder.String()
}

func lowerBoundInvalid(fieldType, minimum string) string {
	if !hasRepresentableLowerInvalid(fieldType, minimum) {
		return ""
	}
	if fieldType == "double" {
		if invalid := adjacentDoubleInvalid(minimum, -1); invalid != "" {
			return invalid
		}
		return ""
	}
	suffix := ""
	if strings.HasSuffix(minimum, "m") || strings.HasSuffix(minimum, "d") || strings.HasSuffix(minimum, "L") {
		suffix = minimum[len(minimum)-1:]
		minimum = strings.TrimSuffix(minimum, suffix)
	}
	return minimum + suffix + " - 1" + suffix
}

func upperBoundInvalid(fieldType, maximum string) string {
	if !hasRepresentableUpperInvalid(fieldType, maximum) {
		return ""
	}
	if fieldType == "double" {
		if invalid := adjacentDoubleInvalid(maximum, 1); invalid != "" {
			return invalid
		}
		return ""
	}
	suffix := ""
	if strings.HasSuffix(maximum, "m") || strings.HasSuffix(maximum, "d") || strings.HasSuffix(maximum, "L") {
		suffix = maximum[len(maximum)-1:]
		maximum = strings.TrimSuffix(maximum, suffix)
	}
	return maximum + suffix + " + 1" + suffix
}

func hasRepresentableLowerInvalid(fieldType, minimum string) bool {
	value, ok := parseCSharpNumberLiteral(minimum)
	if !ok {
		return true
	}
	switch fieldType {
	case "int":
		return value.Cmp(big.NewRat(math.MinInt32, 1)) > 0
	case "long":
		return value.Cmp(big.NewRat(math.MinInt64, 1)) > 0
	case "decimal":
		return value.Cmp(decimalDomainMin) > 0
	case "double":
		parsed, ok := parseFiniteDoubleLiteral(minimum)
		return ok && parsed > -math.MaxFloat64
	default:
		return true
	}
}

func hasRepresentableUpperInvalid(fieldType, maximum string) bool {
	value, ok := parseCSharpNumberLiteral(maximum)
	if !ok {
		return true
	}
	switch fieldType {
	case "int":
		return value.Cmp(big.NewRat(math.MaxInt32, 1)) < 0
	case "long":
		return value.Cmp(big.NewRat(math.MaxInt64, 1)) < 0
	case "decimal":
		return value.Cmp(decimalDomainMax) < 0
	case "double":
		parsed, ok := parseFiniteDoubleLiteral(maximum)
		return ok && parsed < math.MaxFloat64
	default:
		return true
	}
}

func adjacentDoubleInvalid(value string, direction int) string {
	parsed, ok := parseFiniteDoubleLiteral(value)
	if !ok {
		return ""
	}
	toward := math.Inf(direction)
	adjacent := math.Nextafter(parsed, toward)
	if math.IsInf(adjacent, 0) || math.IsNaN(adjacent) {
		return ""
	}
	return strconv.FormatFloat(adjacent, 'g', 17, 64) + "d"
}

func parseFiniteDoubleLiteral(value string) (float64, bool) {
	trimmed := strings.TrimSuffix(value, "d")
	parsed, err := strconv.ParseFloat(trimmed, 64)
	return parsed, err == nil && !math.IsInf(parsed, 0) && !math.IsNaN(parsed)
}

func parseCSharpNumberLiteral(value string) (*big.Rat, bool) {
	trimmed := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(value, "m"), "d"), "L")
	r, ok := new(big.Rat).SetString(trimmed)
	return r, ok
}

func mustRat(value string) *big.Rat {
	r, ok := new(big.Rat).SetString(value)
	if !ok {
		panic("invalid rational constant " + value)
	}
	return r
}
