package generator

import "strings"

func sqlLiteralFor(fieldType, csharpValue string) string {
	switch fieldType {
	case "string":
		return "N'" + strings.ReplaceAll(strings.Trim(csharpValue, "\""), "'", "''") + "'"
	case "Guid":
		return "'" + strings.Split(strings.Split(csharpValue, "\"")[1], "\"")[0] + "'"
	case "DateTime":
		if strings.Contains(csharpValue, "2024, 2, 1") {
			return "'2024-02-01T00:00:00'"
		}
		return "'2024-01-01T00:00:00'"
	case "bool":
		if csharpValue == "true" {
			return "1"
		}
		return "0"
	case "decimal", "double", "long":
		value := strings.TrimRight(strings.TrimRight(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(csharpValue, "m"), "d"), "L"), "0"), ".")
		if value == "" || value == "-" {
			return "0"
		}
		return value
	default:
		return csharpValue
	}
}

func sqlInvalidLiteralFor(fieldType, code, csharpValue string) string {
	switch fieldType {
	case "string":
		if strings.HasSuffix(code, ".MaxLength") {
			return "REPLICATE(N'x', 4096)"
		}
		return sqlLiteralFor(fieldType, csharpValue)
	case "Guid":
		return "'00000000-0000-0000-0000-000000000000'"
	case "DateTime":
		return "'0001-01-01T00:00:00'"
	case "decimal", "double", "long":
		return strings.NewReplacer("m", "", "d", "", "L", "").Replace(csharpValue)
	default:
		return csharpValue
	}
}
