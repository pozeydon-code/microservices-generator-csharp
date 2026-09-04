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
	ConfigSchemaVersion         = 1
	DefaultTargetFramework      = "net8.0"
	minimumTargetFrameworkMajor = 8
	MaxIdentifierLength         = 64
	MaxServices                 = 20
	MaxEntitiesPerService       = 100
	MaxFieldsPerEntity          = 100
)

var reservedRelationshipForeignKeyNames = []string{"Id", "RowVersion", "ConcurrencyToken"}

type Config struct {
	SchemaVersion int               `json:"schemaVersion,omitempty"`
	Generation    GenerationOptions `json:"generation,omitempty"`
	Solution      Solution          `json:"solution"`
	Services      []Service         `json:"services"`
}

type GenerationOptions struct {
	TargetFramework            string         `json:"targetFramework,omitempty"`
	SolutionFormat             string         `json:"solutionFormat,omitempty"`
	EnableValueObjectPreflight bool           `json:"enableValueObjectPreflight,omitempty"`
	Gateway                    GatewayOptions `json:"gateway,omitempty"`
}

type GatewayOptions struct {
	Enabled bool `json:"enabled,omitempty"`
}

type Solution struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Service struct {
	Name          string         `json:"name"`
	ValueObjects  []ValueObject  `json:"valueObjects"`
	Entities      []Entity       `json:"entities"`
	Relationships []Relationship `json:"relationships,omitempty"`
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

type Relationship struct {
	Name                string   `json:"name,omitempty"`
	Multiplicity        string   `json:"multiplicity"`
	PrincipalEntity     string   `json:"principalEntity"`
	DependentEntity     string   `json:"dependentEntity"`
	ForeignKeyName      string   `json:"foreignKeyName,omitempty"`
	ForeignKeyType      string   `json:"foreignKeyType,omitempty"`
	Required            *bool    `json:"required,omitempty"`
	PrincipalNavigation string   `json:"principalNavigation,omitempty"`
	DependentNavigation string   `json:"dependentNavigation,omitempty"`
	ForeignKeyNames     []string `json:"foreignKeyNames,omitempty"`
	PrincipalKeyName    string   `json:"principalKeyName,omitempty"`
	DeleteBehavior      string   `json:"deleteBehavior,omitempty"`
}

type CanonicalRelationship struct {
	Name                string
	Multiplicity        string
	PrincipalEntity     string
	DependentEntity     string
	ForeignKeyName      string
	ForeignKeyType      string
	Required            bool
	PrincipalNavigation string
	DependentNavigation string
}

func (r CanonicalRelationship) Nullable() bool {
	return !r.Required
}

func (s Service) CanonicalRelationships() []CanonicalRelationship {
	relationships := make([]CanonicalRelationship, 0, len(s.Relationships))
	for _, relationship := range s.Relationships {
		relationships = append(relationships, relationship.canonical())
	}
	return relationships
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

var targetFrameworkPattern = regexp.MustCompile(`^net([1-9][0-9]?)\.0$`)

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

	if c.SchemaVersion != 0 && c.SchemaVersion != ConfigSchemaVersion {
		problems = append(problems, fmt.Sprintf("schemaVersion must be %d", ConfigSchemaVersion))
	}
	if !IsSupportedTargetFramework(c.TargetFramework()) {
		problems = append(problems, "generation.targetFramework must be net8.0 or newer (netN.0 with a numeric major version from 8 through 99)")
	}
	if !isSupportedSolutionFormat(c.Generation.SolutionFormat) {
		problems = append(problems, "generation.solutionFormat must be sln or slnx")
	}
	validateRequiredIdentifier(&problems, "solution.name", c.Solution.Name)
	validateCount(&problems, "services", len(c.Services), 1, MaxServices)

	serviceNames := map[string]struct{}{}
	for serviceIndex, service := range c.Services {
		servicePath := fmt.Sprintf("services[%d]", serviceIndex)
		validateRequiredIdentifier(&problems, servicePath+".name", service.Name)
		addUnique(&problems, serviceNames, service.Name, "service")
		validateCount(&problems, servicePath+".entities", len(service.Entities), 1, MaxEntitiesPerService)

		entityNames := map[string]struct{}{}
		entityExactNames := map[string]Entity{}
		for _, entity := range service.Entities {
			addUnique(&problems, entityNames, entity.Name, "entity in service "+service.Name)
			if strings.TrimSpace(entity.Name) != "" {
				entityExactNames[entity.Name] = entity
			}
		}

		valueObjectNames := map[string]ValueObject{}
		valueObjectExactNames := map[string]struct{}{}
		for valueObjectIndex, valueObject := range service.ValueObjects {
			valueObjectPath := fmt.Sprintf("%s.valueObjects[%d]", servicePath, valueObjectIndex)
			validateRequiredIdentifier(&problems, valueObjectPath+".name", valueObject.Name)
			if strings.TrimSpace(valueObject.Name) != "" {
				valueObjectExactNames[valueObject.Name] = struct{}{}
			}
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
					if _, valueObject := valueObjectExactNames[field.Type]; !valueObject {
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
		validateRelationships(&problems, servicePath, service, entityExactNames)
	}

	if len(problems) > 0 {
		return ValidationError{Problems: problems}
	}
	return nil
}

func (c Config) TargetFramework() string {
	if strings.TrimSpace(c.Generation.TargetFramework) == "" {
		return DefaultTargetFramework
	}
	return strings.TrimSpace(c.Generation.TargetFramework)
}

func (c Config) SolutionFormat() string {
	if strings.TrimSpace(c.Generation.SolutionFormat) == "" {
		return DefaultSolutionFormat(c.TargetFramework())
	}
	return strings.TrimSpace(strings.ToLower(c.Generation.SolutionFormat))
}

func NormalizeTargetFramework(value string) (string, bool) {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return "", false
	}
	if major, err := strconv.Atoi(trimmed); err == nil {
		return normalizeTargetFrameworkMajor(major)
	}
	match := targetFrameworkPattern.FindStringSubmatch(trimmed)
	if len(match) != 2 {
		return "", false
	}
	major, err := strconv.Atoi(match[1])
	if err != nil {
		return "", false
	}
	return normalizeTargetFrameworkMajor(major)
}

func IsSupportedTargetFramework(value string) bool {
	major, ok := TargetFrameworkMajor(value)
	return ok && major >= minimumTargetFrameworkMajor
}

func DefaultSolutionFormat(targetFramework string) string {
	framework, ok := NormalizeTargetFramework(targetFramework)
	if !ok {
		return "sln"
	}
	major, ok := TargetFrameworkMajor(framework)
	if !ok || major < 10 {
		return "sln"
	}
	return "slnx"
}

func TargetFrameworkMajor(targetFramework string) (int, bool) {
	match := targetFrameworkPattern.FindStringSubmatch(strings.TrimSpace(strings.ToLower(targetFramework)))
	if len(match) != 2 {
		return 0, false
	}
	major, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, false
	}
	return major, true
}

func normalizeTargetFrameworkMajor(major int) (string, bool) {
	if major < 1 || major > 99 {
		return "", false
	}
	return fmt.Sprintf("net%d.0", major), true
}

func isSupportedSolutionFormat(value string) bool {
	format := strings.TrimSpace(strings.ToLower(value))
	return format == "" || format == "sln" || format == "slnx"
}

func generatedTypeNamesFor(entityName string) map[string]struct{} {
	return map[string]struct{}{
		entityName:                                 {},
		entityName + "Dto":                         {},
		"Create" + entityName + "Request":          {},
		"Update" + entityName + "Request":          {},
		"I" + entityName + "Repository":            {},
		entityName + "Repository":                  {},
		"Create" + entityName + "Command":          {},
		"Create" + entityName + "CommandValidator": {},
		"Create" + entityName + "CommandHandler":   {},
		"Update" + entityName + "Command":          {},
		"Update" + entityName + "CommandValidator": {},
		"Update" + entityName + "CommandHandler":   {},
		"Delete" + entityName + "Command":          {},
		"Delete" + entityName + "CommandValidator": {},
		"Delete" + entityName + "CommandHandler":   {},
		"List" + entityName + "Query":              {},
		"List" + entityName + "QueryValidator":     {},
		"List" + entityName + "QueryHandler":       {},
		"Get" + entityName + "ByIdQuery":           {},
		"Get" + entityName + "ByIdQueryHandler":    {},
		entityName + "Controller":                  {},
	}
}

func generatedServiceTypeNames(serviceName string) []string {
	return []string{"DomainError", "DomainResult", serviceName + "DbContext", serviceName + "ArchitectureTests", serviceName + "InfrastructureTests"}
}

func validateRelationships(problems *[]string, servicePath string, service Service, entities map[string]Entity) {
	seenEdges := map[string]struct{}{}
	seenPrincipalNavigations := map[string]struct{}{}
	seenDependentNavigations := map[string]struct{}{}
	seenForeignKeysByDependent := map[string]map[string]string{}
	seenDependentNavigationsByDependent := map[string]map[string]string{}

	for relationshipIndex, relationship := range service.Relationships {
		relationshipPath := fmt.Sprintf("%s.relationships[%d]", servicePath, relationshipIndex)
		canonical := relationship.canonical()

		if strings.TrimSpace(relationship.Name) != "" {
			validateRequiredIdentifier(problems, relationshipPath+".name", relationship.Name)
		}
		if relationship.Multiplicity != "one-to-many" && relationship.Multiplicity != "many-to-one" && relationship.Multiplicity != "one-to-one" {
			*problems = append(*problems, relationshipPath+".multiplicity must be one-to-many, many-to-one, or one-to-one")
		}
		if len(relationship.ForeignKeyNames) > 0 {
			*problems = append(*problems, relationshipPath+".foreignKeyNames is not supported; use foreignKeyName for a single dependent FK")
		}
		if strings.TrimSpace(relationship.PrincipalKeyName) != "" {
			*problems = append(*problems, relationshipPath+".principalKeyName is not supported; one-to-one uses the principal Id key")
		}
		if strings.TrimSpace(relationship.DeleteBehavior) != "" {
			*problems = append(*problems, relationshipPath+".deleteBehavior is not supported; one-to-one uses Restrict delete behavior")
		}
		validateRequiredIdentifier(problems, relationshipPath+".principalEntity", relationship.PrincipalEntity)
		validateRequiredIdentifier(problems, relationshipPath+".dependentEntity", relationship.DependentEntity)
		if strings.TrimSpace(canonical.ForeignKeyName) != "" {
			validateRequiredIdentifier(problems, relationshipPath+".foreignKeyName", canonical.ForeignKeyName)
		}
		if strings.TrimSpace(canonical.PrincipalNavigation) != "" {
			validateRequiredIdentifier(problems, relationshipPath+".principalNavigation", canonical.PrincipalNavigation)
		}
		if strings.TrimSpace(canonical.DependentNavigation) != "" {
			validateRequiredIdentifier(problems, relationshipPath+".dependentNavigation", canonical.DependentNavigation)
		}

		principal, principalOK := entities[relationship.PrincipalEntity]
		if !principalOK && strings.TrimSpace(relationship.PrincipalEntity) != "" {
			*problems = append(*problems, fmt.Sprintf("%s.principalEntity must reference an entity in service %s", relationshipPath, service.Name))
		}
		dependent, dependentOK := entities[relationship.DependentEntity]
		if !dependentOK && strings.TrimSpace(relationship.DependentEntity) != "" {
			*problems = append(*problems, fmt.Sprintf("%s.dependentEntity must reference an entity in service %s", relationshipPath, service.Name))
		}
		if relationship.PrincipalEntity != "" && strings.EqualFold(relationship.PrincipalEntity, relationship.DependentEntity) {
			*problems = append(*problems, relationshipPath+" principalEntity and dependentEntity must be different")
		}

		edgeKey := strings.ToLower(canonical.PrincipalEntity + "\x00" + canonical.DependentEntity + "\x00" + canonical.ForeignKeyName)
		if _, exists := seenEdges[edgeKey]; exists {
			*problems = append(*problems, fmt.Sprintf("%s duplicates canonical relationship %s-%s-%s", relationshipPath, canonical.PrincipalEntity, canonical.DependentEntity, canonical.ForeignKeyName))
		} else {
			seenEdges[edgeKey] = struct{}{}
		}
		if canonical.Multiplicity == "one-to-one" {
			inverseKey := strings.ToLower(canonical.DependentEntity + "\x00" + canonical.PrincipalEntity)
			if _, exists := seenEdges[inverseKey]; exists {
				*problems = append(*problems, fmt.Sprintf("%s duplicates inverse one-to-one relationship %s-%s", relationshipPath, canonical.PrincipalEntity, canonical.DependentEntity))
			}
			seenEdges[strings.ToLower(canonical.PrincipalEntity+"\x00"+canonical.DependentEntity)] = struct{}{}
		}

		if _, ok := supportedFieldTypes[canonical.ForeignKeyType]; !ok {
			*problems = append(*problems, fmt.Sprintf("%s.foreignKeyType must be a supported scalar primitive: %s", relationshipPath, strings.Join(SupportedFieldTypes(), ", ")))
		}
		if canonical.Multiplicity == "one-to-one" && canonical.ForeignKeyType != "Guid" {
			*problems = append(*problems, relationshipPath+".foreignKeyType must be Guid for one-to-one relationships because the principal key is Id")
		}
		if strings.EqualFold(canonical.ForeignKeyName, canonical.DependentNavigation) {
			*problems = append(*problems, fmt.Sprintf("%s.foreignKeyName must not equal dependentNavigation because both generate members on %s", relationshipPath, canonical.DependentEntity))
		}
		dependentKey := strings.ToLower(canonical.DependentEntity)
		foreignKeyNameKey := strings.ToLower(canonical.ForeignKeyName)
		dependentNavigationKey := strings.ToLower(canonical.DependentNavigation)
		if generatedForeignKeys := seenForeignKeysByDependent[dependentKey]; generatedForeignKeys != nil {
			if collidingForeignKeyName, exists := generatedForeignKeys[dependentNavigationKey]; exists {
				*problems = append(*problems, fmt.Sprintf("%s.dependentNavigation %s must not collide with generated foreignKeyName %s on dependent entity %s", relationshipPath, canonical.DependentNavigation, collidingForeignKeyName, canonical.DependentEntity))
			}
		}
		if dependentNavigations := seenDependentNavigationsByDependent[dependentKey]; dependentNavigations != nil {
			if collidingDependentNavigation, exists := dependentNavigations[foreignKeyNameKey]; exists {
				*problems = append(*problems, fmt.Sprintf("%s.foreignKeyName %s must not collide with dependentNavigation %s on dependent entity %s", relationshipPath, canonical.ForeignKeyName, collidingDependentNavigation, canonical.DependentEntity))
			}
		}
		if seenForeignKeysByDependent[dependentKey] == nil {
			seenForeignKeysByDependent[dependentKey] = map[string]string{}
		}
		seenForeignKeysByDependent[dependentKey][foreignKeyNameKey] = canonical.ForeignKeyName
		if seenDependentNavigationsByDependent[dependentKey] == nil {
			seenDependentNavigationsByDependent[dependentKey] = map[string]string{}
		}
		seenDependentNavigationsByDependent[dependentKey][dependentNavigationKey] = canonical.DependentNavigation
		if reservedName, reserved := reservedRelationshipForeignKeyName(canonical.ForeignKeyName); reserved {
			*problems = append(*problems, fmt.Sprintf("%s.foreignKeyName %s is reserved for generated %s members", relationshipPath, canonical.ForeignKeyName, reservedName))
		}
		if dependentOK {
			if field, exists := findField(dependent, canonical.ForeignKeyName); exists && field.Type != canonical.ForeignKeyType {
				*problems = append(*problems, fmt.Sprintf("%s.foreignKeyName %s must have type %s", relationshipPath, canonical.ForeignKeyName, canonical.ForeignKeyType))
			}
			if fieldNameExists(dependent, canonical.DependentNavigation) {
				*problems = append(*problems, fmt.Sprintf("%s.dependentNavigation must not collide with a field on %s", relationshipPath, dependent.Name))
			}
		}
		if principalOK && fieldNameExists(principal, canonical.PrincipalNavigation) {
			*problems = append(*problems, fmt.Sprintf("%s.principalNavigation must not collide with a field on %s", relationshipPath, principal.Name))
		}
		addUnique(problems, seenPrincipalNavigations, canonical.PrincipalEntity+"."+canonical.PrincipalNavigation, "principal navigation in service "+service.Name)
		addUnique(problems, seenDependentNavigations, canonical.DependentEntity+"."+canonical.DependentNavigation, "dependent navigation in service "+service.Name)
	}
}

func reservedRelationshipForeignKeyName(name string) (string, bool) {
	for _, reservedName := range reservedRelationshipForeignKeyNames {
		if strings.EqualFold(name, reservedName) {
			return reservedName, true
		}
	}
	return "", false
}

func (r Relationship) canonical() CanonicalRelationship {
	foreignKeyName := strings.TrimSpace(r.ForeignKeyName)
	if foreignKeyName == "" {
		foreignKeyName = strings.TrimSpace(r.PrincipalEntity) + "Id"
	}
	foreignKeyType := strings.TrimSpace(r.ForeignKeyType)
	if foreignKeyType == "" {
		foreignKeyType = "Guid"
	}
	required := true
	if r.Required != nil {
		required = *r.Required
	}
	principalNavigation := strings.TrimSpace(r.PrincipalNavigation)
	if principalNavigation == "" {
		principalNavigation = strings.TrimSpace(r.DependentEntity) + "s"
		if strings.TrimSpace(r.Multiplicity) == "one-to-one" {
			principalNavigation = strings.TrimSpace(r.DependentEntity)
		}
	}
	dependentNavigation := strings.TrimSpace(r.DependentNavigation)
	if dependentNavigation == "" {
		dependentNavigation = strings.TrimSpace(r.PrincipalEntity)
	}
	return CanonicalRelationship{
		Name:                strings.TrimSpace(r.Name),
		Multiplicity:        strings.TrimSpace(r.Multiplicity),
		PrincipalEntity:     strings.TrimSpace(r.PrincipalEntity),
		DependentEntity:     strings.TrimSpace(r.DependentEntity),
		ForeignKeyName:      foreignKeyName,
		ForeignKeyType:      foreignKeyType,
		Required:            required,
		PrincipalNavigation: principalNavigation,
		DependentNavigation: dependentNavigation,
	}
}

func findField(entity Entity, name string) (Field, bool) {
	for _, field := range entity.Fields {
		if strings.EqualFold(field.Name, name) {
			return field, true
		}
	}
	return Field{}, false
}

func fieldNameExists(entity Entity, name string) bool {
	_, exists := findField(entity, name)
	return exists
}

func SupportedFieldTypes() []string {
	types := make([]string, 0, len(supportedFieldTypes))
	for fieldType := range supportedFieldTypes {
		types = append(types, fieldType)
	}
	sort.Strings(types)
	return types
}

func SupportedTargetFrameworks() []string {
	return []string{"net10.0", "net9.0", "net8.0"}
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
