package generator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pozeydon-code/generator-microservices-go/internal/spec"
)

func invalidSamplesFor(valueObject ValueObjectView) []InvalidSampleView {
	var samples []InvalidSampleView
	if valueObject.HasRequired {
		samples = append(samples, InvalidSampleView{FieldValue: "\"\"", Code: valueObject.Name + ".Required", Message: valueObject.Name + " is required.", TestName: "Required"})
	}
	if valueObject.MinLength != "" {
		samples = append(samples, InvalidSampleView{FieldValue: "\"x\"", Code: valueObject.Name + ".MinLength", Message: valueObject.Name + " must be at least " + valueObject.MinLength + " characters.", TestName: "MinLength"})
	}
	if valueObject.MaxLength != "" {
		samples = append(samples, InvalidSampleView{FieldValue: "new string('x', " + valueObject.MaxLength + " + 1)", Code: valueObject.Name + ".MaxLength", Message: valueObject.Name + " must be at most " + valueObject.MaxLength + " characters.", TestName: "MaxLength"})
	}
	if valueObject.Pattern != "" {
		invalid := valueObject.PatternInvalidValue
		if invalid != "" {
			samples = append(samples, InvalidSampleView{FieldValue: invalid, Code: valueObject.Name + ".Pattern", Message: valueObject.Name + " has an invalid format.", TestName: "Pattern"})
		}
	}
	if valueObject.Minimum != "" {
		if invalid := lowerBoundInvalid(valueObject.Type, valueObject.Minimum); invalid != "" {
			samples = append(samples, InvalidSampleView{FieldValue: invalid, Code: valueObject.Name + ".Minimum", Message: valueObject.Name + " must be greater than or equal to " + valueObject.Minimum + ".", TestName: "Minimum"})
		}
	}
	if valueObject.Maximum != "" {
		if invalid := upperBoundInvalid(valueObject.Type, valueObject.Maximum); invalid != "" {
			samples = append(samples, InvalidSampleView{FieldValue: invalid, Code: valueObject.Name + ".Maximum", Message: valueObject.Name + " must be less than or equal to " + valueObject.Maximum + ".", TestName: "Maximum"})
		}
	}
	if valueObject.HasNotEmpty {
		samples = append(samples, InvalidSampleView{FieldValue: "Guid.Empty", Code: valueObject.Name + ".NotEmpty", Message: valueObject.Name + " must not be empty.", TestName: "NotEmpty"})
	}
	if valueObject.HasNotDefault {
		samples = append(samples, InvalidSampleView{FieldValue: "default", Code: valueObject.Name + ".NotDefault", Message: valueObject.Name + " must not be the default value.", TestName: "NotDefault"})
	}
	return samples
}

func patternInvalidSampleFor(rules spec.ValidationRules) string {
	if rules.Pattern == nil {
		return ""
	}
	if rules.InvalidExample != nil && stringRulesBeforePatternAcceptForGenerator(*rules.InvalidExample, rules) && stringPatternRejectsForGenerator(*rules.InvalidExample, *rules.Pattern) {
		return *rules.InvalidExample
	}

	minLength := 1
	if rules.MinLength != nil && *rules.MinLength > minLength {
		minLength = *rules.MinLength
	}
	maxLength := minLength
	if rules.MaxLength != nil && *rules.MaxLength < maxLength {
		return ""
	}

	for _, seed := range []string{"!", "*", "_", "#", " ", "0"} {
		candidate := strings.Repeat(seed, maxLength)
		if stringRulesBeforePatternAcceptForGenerator(candidate, rules) && stringPatternRejectsForGenerator(candidate, *rules.Pattern) {
			return candidate
		}
	}
	return ""
}

func stringRulesBeforePatternAcceptForGenerator(value string, rules spec.ValidationRules) bool {
	if rules.Required != nil && *rules.Required && strings.TrimSpace(value) == "" {
		return false
	}
	if rules.MinLength != nil && len(value) < *rules.MinLength {
		return false
	}
	if rules.MaxLength != nil && len(value) > *rules.MaxLength {
		return false
	}
	return true
}

func stringPatternRejectsForGenerator(value, pattern string) bool {
	re, err := regexp.Compile(pattern)
	return err == nil && !re.MatchString(value)
}

func stringRulesAcceptForGenerator(value string, rules spec.ValidationRules) bool {
	if rules.Required != nil && *rules.Required && strings.TrimSpace(value) == "" {
		return false
	}
	if rules.MinLength != nil && len(value) < *rules.MinLength {
		return false
	}
	if rules.MaxLength != nil && len(value) > *rules.MaxLength {
		return false
	}
	if rules.Pattern != nil {
		re, err := regexp.Compile(*rules.Pattern)
		if err != nil || !re.MatchString(value) {
			return false
		}
	}
	return true
}

func assertionFor(fieldType, expected, name string) string {
	if fieldType == "bool" {
		if expected == "true" {
			return fmt.Sprintf("Assert.True(created.%s);", name)
		}
		return fmt.Sprintf("Assert.False(created.%s);", name)
	}
	return fmt.Sprintf("Assert.Equal(%s, created.%s);", expected, name)
}

func sampleValueFor(fieldType, name string) string {
	switch fieldType {
	case "string":
		return fmt.Sprintf("\"%s Value\"", name)
	case "bool":
		return "true"
	case "decimal":
		return "12.34m"
	case "double":
		return "12.34d"
	case "int":
		return "12"
	case "long":
		return "12L"
	case "Guid":
		if name == "Id" {
			return "Guid.Parse(\"11111111-1111-1111-1111-111111111111\")"
		}
		return "Guid.Parse(\"00000000-0000-0000-0000-000000000001\")"
	case "DateTime":
		return "new DateTime(2024, 1, 1, 0, 0, 0, DateTimeKind.Utc)"
	default:
		return "default"
	}
}

func updatedValueFor(fieldType, name string) string {
	switch fieldType {
	case "string":
		return fmt.Sprintf("\"Updated %s\"", name)
	case "bool":
		return "false"
	case "decimal":
		return "56.78m"
	case "double":
		return "56.78d"
	case "int":
		return "34"
	case "long":
		return "34L"
	case "Guid":
		return "Guid.Parse(\"00000000-0000-0000-0000-000000000002\")"
	case "DateTime":
		return "new DateTime(2024, 2, 1, 0, 0, 0, DateTimeKind.Utc)"
	default:
		return "default"
	}
}

func initializerFor(fieldType string) string {
	switch fieldType {
	case "string":
		return " = string.Empty;"
	case "bool", "DateTime", "decimal", "double", "Guid", "int", "long":
		return ""
	default:
		return " = null!;"
	}
}
