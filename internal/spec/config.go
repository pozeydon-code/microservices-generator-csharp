package spec

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	MaxIdentifierLength   = 64
	MaxServices           = 20
	MaxEntitiesPerService = 100
	MaxFieldsPerEntity    = 100
)

type Config struct {
	Solution Solution  `json:"solution"`
	Services []Service `json:"services"`
}

type Solution struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Service struct {
	Name         string        `json:"name"`
	ValueObjects []ValueObject `json:"valueObjects"`
	Entities     []Entity      `json:"entities"`
}

type ValueObject struct {
	Name        string          `json:"name"`
	Type        string          `json:"type"`
	Validations ValidationRules `json:"validations"`
}

type ValidationRules struct {
	Required       *bool        `json:"required,omitempty"`
	MinLength      *int         `json:"minLength,omitempty"`
	MaxLength      *int         `json:"maxLength,omitempty"`
	Pattern        *string      `json:"pattern,omitempty"`
	ValidExample   *string      `json:"validExample,omitempty"`
	InvalidExample *string      `json:"invalidExample,omitempty"`
	Minimum        *json.Number `json:"minimum,omitempty"`
	Maximum        *json.Number `json:"maximum,omitempty"`
	NotEmpty       *bool        `json:"notEmpty,omitempty"`
	NotDefault     *bool        `json:"notDefault,omitempty"`
}

type Entity struct {
	Name   string  `json:"name"`
	Fields []Field `json:"fields"`
}

type Field struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type ValidationError struct {
	Problems []string
}

func (e ValidationError) Error() string {
	return "invalid config:\n- " + strings.Join(e.Problems, "\n- ")
}

var csharpIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

var (
	decimalMin = mustRat("-79228162514264337593543950335")
	decimalMax = mustRat("79228162514264337593543950335")
)

var supportedFieldTypes = map[string]struct{}{
	"bool": {}, "DateTime": {}, "decimal": {}, "double": {},
	"Guid": {}, "int": {}, "long": {}, "string": {},
}

var csharpKeywords = map[string]struct{}{
	"abstract": {}, "as": {}, "base": {}, "bool": {}, "break": {}, "byte": {}, "case": {}, "catch": {}, "char": {}, "checked": {},
	"class": {}, "const": {}, "continue": {}, "decimal": {}, "default": {}, "delegate": {}, "do": {}, "double": {}, "else": {},
	"enum": {}, "event": {}, "explicit": {}, "extern": {}, "false": {}, "finally": {}, "fixed": {}, "float": {}, "for": {},
	"foreach": {}, "goto": {}, "if": {}, "implicit": {}, "in": {}, "int": {}, "interface": {}, "internal": {}, "is": {},
	"lock": {}, "long": {}, "namespace": {}, "new": {}, "null": {}, "object": {}, "operator": {}, "out": {}, "override": {},
	"params": {}, "private": {}, "protected": {}, "public": {}, "readonly": {}, "ref": {}, "return": {}, "sbyte": {}, "sealed": {},
	"short": {}, "sizeof": {}, "stackalloc": {}, "static": {}, "string": {}, "struct": {}, "switch": {}, "this": {}, "throw": {},
	"true": {}, "try": {}, "typeof": {}, "uint": {}, "ulong": {}, "unchecked": {}, "unsafe": {}, "ushort": {}, "using": {},
	"virtual": {}, "void": {}, "volatile": {}, "while": {},
}

var windowsReservedPathSegments = map[string]struct{}{
	"CON": {}, "PRN": {}, "AUX": {}, "NUL": {},
	"COM1": {}, "COM2": {}, "COM3": {}, "COM4": {}, "COM5": {}, "COM6": {}, "COM7": {}, "COM8": {}, "COM9": {},
	"LPT1": {}, "LPT2": {}, "LPT3": {}, "LPT4": {}, "LPT5": {}, "LPT6": {}, "LPT7": {}, "LPT8": {}, "LPT9": {},
}

func (c Config) Validate() error {
	var problems []string

	validateRequiredIdentifier(&problems, "solution.name", c.Solution.Name)
	validateCount(&problems, "services", len(c.Services), 1, MaxServices)

	serviceNames := map[string]struct{}{}
	for serviceIndex, service := range c.Services {
		servicePath := fmt.Sprintf("services[%d]", serviceIndex)
		validateRequiredIdentifier(&problems, servicePath+".name", service.Name)
		addUnique(&problems, serviceNames, service.Name, "service")
		validateCount(&problems, servicePath+".entities", len(service.Entities), 1, MaxEntitiesPerService)

		entityNames := map[string]struct{}{}
		for _, entity := range service.Entities {
			addUnique(&problems, entityNames, entity.Name, "entity in service "+service.Name)
		}

		valueObjectNames := map[string]ValueObject{}
		for valueObjectIndex, valueObject := range service.ValueObjects {
			valueObjectPath := fmt.Sprintf("%s.valueObjects[%d]", servicePath, valueObjectIndex)
			validateRequiredIdentifier(&problems, valueObjectPath+".name", valueObject.Name)
			if _, primitive := supportedFieldTypes[valueObject.Name]; primitive {
				problems = append(problems, valueObjectPath+".name must not collide with a supported primitive type")
			}
			if _, entityCollision := entityNames[strings.ToLower(valueObject.Name)]; entityCollision {
				problems = append(problems, fmt.Sprintf("%s.name must not collide with entity %q", valueObjectPath, valueObject.Name))
			}
			for _, generatedName := range generatedServiceTypeNames(service.Name) {
				if strings.EqualFold(valueObject.Name, generatedName) {
					problems = append(problems, fmt.Sprintf("%s.name must not collide with generated C# type %q", valueObjectPath, generatedName))
				}
			}
			addUniqueValueObject(&problems, valueObjectNames, valueObject, "value object in service "+service.Name)
			if _, ok := supportedFieldTypes[valueObject.Type]; !ok {
				problems = append(problems, fmt.Sprintf("%s.type must be a supported scalar primitive: %s", valueObjectPath, strings.Join(SupportedFieldTypes(), ", ")))
			}
			validateRules(&problems, valueObjectPath+".validations", valueObject.Type, valueObject.Validations)
		}

		for entityIndex, entity := range service.Entities {
			entityPath := fmt.Sprintf("%s.entities[%d]", servicePath, entityIndex)
			validateRequiredIdentifier(&problems, entityPath+".name", entity.Name)
			validateCount(&problems, entityPath+".fields", len(entity.Fields), 1, MaxFieldsPerEntity)
			generatedTypeNames := generatedTypeNamesFor(entity.Name)

			fieldNames := map[string]struct{}{}
			idFieldCount := 0
			for fieldIndex, field := range entity.Fields {
				fieldPath := fmt.Sprintf("%s.fields[%d]", entityPath, fieldIndex)
				validateRequiredIdentifier(&problems, fieldPath+".name", field.Name)
				if field.Name == entity.Name {
					problems = append(problems, fieldPath+".name must not equal its enclosing entity name")
				}
				if strings.EqualFold(field.Name, "RowVersion") {
					problems = append(problems, fmt.Sprintf("%s.name is reserved for infrastructure concurrency storage", fieldPath))
				}
				if strings.EqualFold(field.Name, "ConcurrencyToken") {
					problems = append(problems, fmt.Sprintf("%s.name must not collide case-insensitively with generated JSON contract field \"ConcurrencyToken\"", fieldPath))
				}
				if _, collides := generatedTypeNames[field.Name]; collides {
					problems = append(problems, fmt.Sprintf("%s.name must not collide with generated C# type %q", fieldPath, field.Name))
				}
				if strings.EqualFold(field.Name, "Id") {
					idFieldCount++
					if field.Name != "Id" {
						problems = append(problems, fieldPath+".name must be exactly \"Id\" for the entity identity field")
					}
					if field.Type != "Guid" {
						problems = append(problems, fieldPath+".type must be \"Guid\" for the entity identity field")
					}
				}
				addUnique(&problems, fieldNames, field.Name, "field in entity "+entity.Name)
				if _, ok := supportedFieldTypes[field.Type]; !ok {
					if _, valueObject := valueObjectNames[strings.ToLower(field.Type)]; !valueObject {
						problems = append(problems, fmt.Sprintf("%s.type must be one of %s or a declared service value object", fieldPath, strings.Join(SupportedFieldTypes(), ", ")))
					}
				}
			}
			if idFieldCount == 0 {
				problems = append(problems, entityPath+".fields must contain exactly one Id field of type Guid")
			}
			if idFieldCount > 1 {
				problems = append(problems, entityPath+".fields must contain only one Id field")
			}
		}
	}

	if len(problems) > 0 {
		return ValidationError{Problems: problems}
	}
	return nil
}

func generatedTypeNamesFor(entityName string) map[string]struct{} {
	return map[string]struct{}{
		entityName:                        {},
		entityName + "Dto":                {},
		"Create" + entityName + "Request": {},
		"Update" + entityName + "Request": {},
		"I" + entityName + "Repository":   {},
		"I" + entityName + "UseCases":     {},
		entityName + "UseCases":           {},
		entityName + "Repository":         {},
		entityName + "Endpoints":          {},
	}
}

func generatedServiceTypeNames(serviceName string) []string {
	return []string{"DomainError", "DomainResult", serviceName + "DbContext", serviceName + "ArchitectureTests", serviceName + "InfrastructureTests"}
}

func SupportedFieldTypes() []string {
	types := make([]string, 0, len(supportedFieldTypes))
	for fieldType := range supportedFieldTypes {
		types = append(types, fieldType)
	}
	sort.Strings(types)
	return types
}

func validateRequiredIdentifier(problems *[]string, path, value string) {
	if strings.TrimSpace(value) == "" {
		*problems = append(*problems, path+" is required")
		return
	}
	if len(value) > MaxIdentifierLength {
		*problems = append(*problems, fmt.Sprintf("%s must be at most %d characters", path, MaxIdentifierLength))
	}
	if !csharpIdentifierPattern.MatchString(value) {
		*problems = append(*problems, path+" must be a valid C# identifier")
		return
	}
	if _, keyword := csharpKeywords[value]; keyword {
		*problems = append(*problems, path+" must not be a C# keyword")
	}
	if _, reserved := windowsReservedPathSegments[strings.ToUpper(value)]; reserved {
		*problems = append(*problems, path+" must not be a Windows reserved path segment")
	}
}

func validateCount(problems *[]string, path string, count, min, max int) {
	if count < min {
		*problems = append(*problems, fmt.Sprintf("%s must contain at least %d item", path, min))
	}
	if count > max {
		*problems = append(*problems, fmt.Sprintf("%s must contain at most %d items", path, max))
	}
}

func addUnique(problems *[]string, seen map[string]struct{}, value, label string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	key := strings.ToLower(value)
	if _, exists := seen[key]; exists {
		*problems = append(*problems, fmt.Sprintf("duplicate %s name %q", label, value))
		return
	}
	seen[key] = struct{}{}
}

func addUniqueValueObject(problems *[]string, seen map[string]ValueObject, value ValueObject, label string) {
	if strings.TrimSpace(value.Name) == "" {
		return
	}
	key := strings.ToLower(value.Name)
	if _, exists := seen[key]; exists {
		*problems = append(*problems, fmt.Sprintf("duplicate %s name %q", label, value.Name))
		return
	}
	seen[key] = value
}

func validateRules(problems *[]string, path, primitiveType string, rules ValidationRules) {
	allowed := map[string]bool{}
	switch primitiveType {
	case "string":
		allowed = map[string]bool{"required": true, "minLength": true, "maxLength": true, "pattern": true, "validExample": true, "invalidExample": true}
	case "int", "long", "double", "decimal":
		allowed = map[string]bool{"minimum": true, "maximum": true}
	case "Guid":
		allowed = map[string]bool{"notEmpty": true}
	case "DateTime":
		allowed = map[string]bool{"notDefault": true}
	case "bool":
		allowed = map[string]bool{}
	}
	if rules.Required != nil && !allowed["required"] {
		*problems = append(*problems, path+".required is not applicable to "+primitiveType)
	}
	if rules.MinLength != nil && !allowed["minLength"] {
		*problems = append(*problems, path+".minLength is not applicable to "+primitiveType)
	}
	if rules.MaxLength != nil && !allowed["maxLength"] {
		*problems = append(*problems, path+".maxLength is not applicable to "+primitiveType)
	}
	if rules.Pattern != nil && !allowed["pattern"] {
		*problems = append(*problems, path+".pattern is not applicable to "+primitiveType)
	}
	if rules.ValidExample != nil && !allowed["validExample"] {
		*problems = append(*problems, path+".validExample is not applicable to "+primitiveType)
	}
	if rules.InvalidExample != nil && !allowed["invalidExample"] {
		*problems = append(*problems, path+".invalidExample is not applicable to "+primitiveType)
	}
	if rules.Minimum != nil && !allowed["minimum"] {
		*problems = append(*problems, path+".minimum is not applicable to "+primitiveType)
	}
	if rules.Maximum != nil && !allowed["maximum"] {
		*problems = append(*problems, path+".maximum is not applicable to "+primitiveType)
	}
	if rules.NotEmpty != nil && !allowed["notEmpty"] {
		*problems = append(*problems, path+".notEmpty is not applicable to "+primitiveType)
	}
	if rules.NotDefault != nil && !allowed["notDefault"] {
		*problems = append(*problems, path+".notDefault is not applicable to "+primitiveType)
	}
	if rules.MinLength != nil && *rules.MinLength < 0 {
		*problems = append(*problems, path+".minLength must be nonnegative")
	}
	if rules.MaxLength != nil && *rules.MaxLength < 0 {
		*problems = append(*problems, path+".maxLength must be nonnegative")
	}
	if rules.MinLength != nil && rules.MaxLength != nil && *rules.MinLength > *rules.MaxLength {
		*problems = append(*problems, path+".minLength must be less than or equal to maxLength")
	}
	if rules.Pattern != nil {
		if err := validatePortablePattern(*rules.Pattern); err != nil {
			*problems = append(*problems, path+".pattern "+err.Error())
		} else if _, err := regexp.Compile(*rules.Pattern); err != nil {
			*problems = append(*problems, path+".pattern must compile as a regular expression")
		}
		if rules.ValidExample == nil {
			*problems = append(*problems, path+".validExample is required when pattern is set")
		}
		if rules.InvalidExample == nil {
			*problems = append(*problems, path+".invalidExample is required when pattern is set")
		}
	}
	if primitiveType == "string" {
		if rules.ValidExample != nil && !stringRulesAccept(*rules.ValidExample, rules) {
			*problems = append(*problems, path+".validExample must satisfy all string validations")
		}
		if rules.InvalidExample != nil && stringRulesAccept(*rules.InvalidExample, rules) {
			*problems = append(*problems, path+".invalidExample must violate at least one string validation")
		}
	}
	if rules.Minimum != nil || rules.Maximum != nil {
		if err := validateNumericBounds(primitiveType, rules.Minimum, rules.Maximum); err != nil {
			*problems = append(*problems, path+err.Error())
		}
	}
}

func validatePortablePattern(pattern string) error {
	if len(pattern) > 256 {
		return fmt.Errorf("must be at most 256 characters")
	}
	for _, r := range pattern {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("must not contain control characters")
		}
	}
	unsupported := []string{"(?", "[[:", "\\Q", "\\E", "\\k<", "\\1", "\\2", "\\3", "\\4", "\\5", "\\6", "\\7", "\\8", "\\9"}
	for _, token := range unsupported {
		if strings.Contains(pattern, token) {
			return fmt.Errorf("uses unsupported portable regex construct %q", token)
		}
	}
	for index := 0; index < len(pattern); index++ {
		if pattern[index] != '\\' {
			continue
		}
		if index == len(pattern)-1 {
			return fmt.Errorf("uses unsupported portable regex escape %q", "\\")
		}
		index++
		if !strings.ContainsRune(`\\.-^$|?*+()[]{}dDsSwW`, rune(pattern[index])) {
			return fmt.Errorf("uses unsupported portable regex escape %q", "\\"+string(pattern[index]))
		}
	}
	return nil
}

func stringRulesAccept(value string, rules ValidationRules) bool {
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

func validateNumericBounds(primitiveType string, minimum, maximum *json.Number) error {
	parse := func(label string, n *json.Number) (*big.Rat, error) {
		if n == nil {
			return nil, nil
		}
		s := n.String()
		if strings.ContainsAny(strings.ToLower(s), "nif") {
			return nil, fmt.Errorf(".%s must be a finite JSON number", label)
		}
		if primitiveType == "int" || primitiveType == "long" {
			if strings.ContainsAny(s, ".eE") {
				return nil, fmt.Errorf(".%s must be an integer literal without fraction or exponent", label)
			}
		}
		if primitiveType == "decimal" {
			if strings.ContainsAny(s, "eE") {
				return nil, fmt.Errorf(".%s must fit .NET decimal precision without exponent", label)
			}
			digits, scale := decimalDigitsAndScale(s)
			if digits > 29 || scale > 28 {
				return nil, fmt.Errorf(".%s must fit .NET decimal precision and scale", label)
			}
		}
		r, ok := new(big.Rat).SetString(s)
		if !ok {
			return nil, fmt.Errorf(".%s must be a valid JSON number", label)
		}
		if primitiveType == "double" {
			f, err := strconv.ParseFloat(s, 64)
			if err != nil || math.IsInf(f, 0) || math.IsNaN(f) {
				return nil, fmt.Errorf(".%s must be a finite double", label)
			}
		}
		if primitiveType == "int" && (r.Cmp(big.NewRat(math.MinInt32, 1)) < 0 || r.Cmp(big.NewRat(math.MaxInt32, 1)) > 0) {
			return nil, fmt.Errorf(".%s must be within Int32 range", label)
		}
		if primitiveType == "long" && (r.Cmp(big.NewRat(math.MinInt64, 1)) < 0 || r.Cmp(big.NewRat(math.MaxInt64, 1)) > 0) {
			return nil, fmt.Errorf(".%s must be within Int64 range", label)
		}
		if primitiveType == "decimal" && (r.Cmp(decimalMin) < 0 || r.Cmp(decimalMax) > 0) {
			return nil, fmt.Errorf(".%s must be within System.Decimal range", label)
		}
		return r, nil
	}
	min, err := parse("minimum", minimum)
	if err != nil {
		return err
	}
	max, err := parse("maximum", maximum)
	if err != nil {
		return err
	}
	if min != nil && max != nil && min.Cmp(max) > 0 {
		return fmt.Errorf(".minimum must be less than or equal to maximum")
	}
	return nil
}

func mustRat(value string) *big.Rat {
	r, ok := new(big.Rat).SetString(value)
	if !ok {
		panic("invalid rational constant " + value)
	}
	return r
}

func decimalDigitsAndScale(value string) (int, int) {
	unsigned := strings.TrimPrefix(value, "-")
	parts := strings.SplitN(unsigned, ".", 2)
	whole := strings.TrimLeft(parts[0], "0")
	fraction := ""
	if len(parts) == 2 {
		fraction = strings.TrimRight(parts[1], "0")
	}
	digits := len(whole) + len(fraction)
	if digits == 0 {
		digits = 1
	}
	return digits, len(fraction)
}
