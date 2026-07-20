package spec

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestConfigValidateAcceptsValidConfig(t *testing.T) {
	cfg := validConfig()

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}
}

func TestConfigValidateDefaultsMissingSchemaVersionAndTargetFramework(t *testing.T) {
	cfg := validConfig()

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}
	if cfg.TargetFramework() != DefaultTargetFramework {
		t.Fatalf("expected default target framework %q, got %q", DefaultTargetFramework, cfg.TargetFramework())
	}
}

func TestConfigValidateAcceptsFlexibleTargetFrameworks(t *testing.T) {
	for _, targetFramework := range []string{"net6.0", "net7.0", "net9.0", "net10.0", "net11.0"} {
		t.Run(targetFramework, func(t *testing.T) {
			cfg := validConfig()
			cfg.SchemaVersion = ConfigSchemaVersion
			cfg.Generation.TargetFramework = targetFramework

			if err := cfg.Validate(); err != nil {
				t.Fatalf("expected %s config to be valid, got %v", targetFramework, err)
			}
			if cfg.TargetFramework() != targetFramework {
				t.Fatalf("expected selected target framework, got %q", cfg.TargetFramework())
			}
		})
	}
}

func TestConfigValidateRejectsUnsupportedSchemaVersionAndInvalidTargetFramework(t *testing.T) {
	cfg := validConfig()
	cfg.SchemaVersion = 99
	cfg.Generation.TargetFramework = "latest"

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	message := err.Error()
	for _, expected := range []string{"schemaVersion must be 1", "generation.targetFramework must be netN.0"} {
		if !strings.Contains(message, expected) {
			t.Fatalf("expected error to contain %q, got:\n%s", expected, message)
		}
	}
}

func TestNormalizeTargetFramework(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
		ok    bool
	}{
		{name: "major only", input: "7", want: "net7.0", ok: true},
		{name: "tfm", input: "net10.0", want: "net10.0", ok: true},
		{name: "case and spaces", input: " NET11.0 ", want: "net11.0", ok: true},
		{name: "minor not zero", input: "net8.1", ok: false},
		{name: "zero", input: "0", ok: false},
		{name: "too high", input: "100", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := NormalizeTargetFramework(tt.input)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("NormalizeTargetFramework(%q) = %q, %t; want %q, %t", tt.input, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestDefaultSolutionFormat(t *testing.T) {
	tests := []struct {
		targetFramework string
		want            string
	}{
		{targetFramework: "net6.0", want: "sln"},
		{targetFramework: "net9.0", want: "sln"},
		{targetFramework: "net10.0", want: "slnx"},
		{targetFramework: "11", want: "slnx"},
	}
	for _, tt := range tests {
		t.Run(tt.targetFramework, func(t *testing.T) {
			if got := DefaultSolutionFormat(tt.targetFramework); got != tt.want {
				t.Fatalf("expected %s, got %s", tt.want, got)
			}
		})
	}
}

func TestConfigSolutionFormatUsesExplicitOrTargetDefault(t *testing.T) {
	cfg := validConfig()
	cfg.Generation.TargetFramework = "net10.0"
	if got := cfg.SolutionFormat(); got != "slnx" {
		t.Fatalf("expected net10.0 to default to slnx, got %q", got)
	}
	cfg.Generation.SolutionFormat = "sln"
	if got := cfg.SolutionFormat(); got != "sln" {
		t.Fatalf("expected explicit sln, got %q", got)
	}
	cfg.Generation.SolutionFormat = "zip"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "generation.solutionFormat must be sln or slnx") {
		t.Fatalf("expected solution format validation error, got %v", err)
	}
}

func TestConfigValidateAggregatesActionableErrors(t *testing.T) {
	cfg := Config{
		Solution: Solution{Name: "class"},
		Services: []Service{
			{
				Name: "ProductService",
				Entities: []Entity{
					{
						Name: "Product",
						Fields: []Field{
							{Name: "Id", Type: "Guid"},
							{Name: "id", Type: "uuid"},
							{Name: "1Name", Type: "string"},
						},
					},
					{Name: "product"},
				},
			},
			{Name: "productservice"},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}

	message := err.Error()
	expectedParts := []string{
		"solution.name must not be a C# keyword",
		"duplicate field in entity Product name \"id\"",
		"services[0].entities[0].fields[1].type must be one of",
		"services[0].entities[0].fields[2].name must be a valid C# identifier",
		"duplicate entity in service ProductService name \"product\"",
		"services[0].entities[1].fields must contain at least 1 item",
		"duplicate service name \"productservice\"",
		"services[1].entities must contain at least 1 item",
	}
	for _, expected := range expectedParts {
		if !strings.Contains(message, expected) {
			t.Fatalf("expected error to contain %q, got:\n%s", expected, message)
		}
	}
}

func TestConfigValidateRejectsBoundedCountsAndPortablePathNames(t *testing.T) {
	tooManyServices := make([]Service, MaxServices+1)
	for index := range tooManyServices {
		tooManyServices[index] = Service{
			Name: fmt.Sprintf("Service%d", index),
			Entities: []Entity{{
				Name:   "Entity",
				Fields: []Field{{Name: "Id", Type: "Guid"}},
			}},
		}
	}
	cfg := Config{
		Solution: Solution{Name: strings.Repeat("A", MaxIdentifierLength+1)},
		Services: append(tooManyServices, Service{
			Name: "CON",
			Entities: []Entity{{
				Name: "LPT1",
				Fields: []Field{
					{Name: "NUL", Type: "string"},
				},
			}},
		}),
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	message := err.Error()
	expectedParts := []string{
		"solution.name must be at most 64 characters",
		"services must contain at most 20 items",
		"services[21].name must not be a Windows reserved path segment",
		"services[21].entities[0].name must not be a Windows reserved path segment",
		"services[21].entities[0].fields[0].name must not be a Windows reserved path segment",
	}
	for _, expected := range expectedParts {
		if !strings.Contains(message, expected) {
			t.Fatalf("expected error to contain %q, got:\n%s", expected, message)
		}
	}
}

func TestConfigValidateAllowsExactMaximumIdentifierLength(t *testing.T) {
	cfg := validConfig()
	cfg.Solution.Name = strings.Repeat("A", MaxIdentifierLength)
	cfg.Services[0].Name = strings.Repeat("B", MaxIdentifierLength)
	cfg.Services[0].Entities[0].Name = strings.Repeat("C", MaxIdentifierLength)
	cfg.Services[0].Entities[0].Fields[0].Name = strings.Repeat("D", MaxIdentifierLength)

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected exact maximum identifier lengths to be valid, got %v", err)
	}
}

func TestConfigValidateRejectsFieldNameEqualToEntityName(t *testing.T) {
	cfg := validConfig()
	cfg.Services[0].Entities[0].Fields = append(cfg.Services[0].Entities[0].Fields, Field{Name: "Product", Type: "string"})

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "fields[2].name must not equal its enclosing entity name") {
		t.Fatalf("expected entity/member collision error, got %v", err)
	}
}

func TestConfigValidateRejectsFieldNamesThatCollideWithGeneratedTypes(t *testing.T) {
	tests := []string{"Product", "ProductDto", "CreateProductRequest", "UpdateProductRequest", "IProductRepository", "IProductUseCases", "ProductUseCases", "ProductRepository", "ProductEndpoints"}
	for _, fieldName := range tests {
		t.Run(fieldName, func(t *testing.T) {
			cfg := validConfig()
			cfg.Services[0].Entities[0].Fields = append(cfg.Services[0].Entities[0].Fields, Field{Name: fieldName, Type: "string"})

			err := cfg.Validate()
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), "must not collide with generated C# type") {
				t.Fatalf("expected generated type collision error, got %v", err)
			}
		})
	}
}

func TestConfigValidateRejectsReservedRowVersionFieldCollisions(t *testing.T) {
	tests := []string{"RowVersion", "rowVersion", "ROWVERSION", "RowVERSION"}
	for _, fieldName := range tests {
		t.Run(fieldName, func(t *testing.T) {
			cfg := validConfig()
			cfg.Services[0].Entities[0].Fields = append(cfg.Services[0].Entities[0].Fields, Field{Name: fieldName, Type: "string"})

			err := cfg.Validate()
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), "is reserved for infrastructure concurrency storage") {
				t.Fatalf("expected reserved RowVersion collision error, got %v", err)
			}
		})
	}
}

func TestConfigValidateRejectsCaseInsensitiveGeneratedJsonConcurrencyTokenFieldCollisions(t *testing.T) {
	tests := []string{"ConcurrencyToken", "concurrencyToken", "CONCURRENCYTOKEN", "ConcurrencyTOKEN"}
	for _, fieldName := range tests {
		t.Run(fieldName, func(t *testing.T) {
			cfg := validConfig()
			cfg.Services[0].Entities[0].Fields = append(cfg.Services[0].Entities[0].Fields, Field{Name: fieldName, Type: "string"})

			err := cfg.Validate()
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), "must not collide case-insensitively with generated JSON contract field \"ConcurrencyToken\"") {
				t.Fatalf("expected generated JSON ConcurrencyToken collision error, got %v", err)
			}
		})
	}
}

func TestConfigValidateRequiresExactlyOneGuidIdField(t *testing.T) {
	tests := []struct {
		name        string
		fields      []Field
		expectedErr string
	}{
		{
			name:        "missing Id",
			fields:      []Field{{Name: "Name", Type: "string"}},
			expectedErr: "services[0].entities[0].fields must contain exactly one Id field of type Guid",
		},
		{
			name:        "wrong Id type",
			fields:      []Field{{Name: "Id", Type: "string"}, {Name: "Name", Type: "string"}},
			expectedErr: "services[0].entities[0].fields[0].type must be \"Guid\" for the entity identity field",
		},
		{
			name:        "incorrect Id casing",
			fields:      []Field{{Name: "id", Type: "Guid"}, {Name: "Name", Type: "string"}},
			expectedErr: "services[0].entities[0].fields[0].name must be exactly \"Id\" for the entity identity field",
		},
		{
			name:        "duplicate Id",
			fields:      []Field{{Name: "Id", Type: "Guid"}, {Name: "id", Type: "Guid"}},
			expectedErr: "services[0].entities[0].fields must contain only one Id field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			cfg.Services[0].Entities[0].Fields = tt.fields

			err := cfg.Validate()
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tt.expectedErr) {
				t.Fatalf("expected %q, got %v", tt.expectedErr, err)
			}
		})
	}
}

func TestConfigValidateAcceptsDeclaredValueObjectFieldTypes(t *testing.T) {
	cfg := validConfig()
	cfg.Services[0].ValueObjects = []ValueObject{{Name: "ProductName", Type: "string", Validations: ValidationRules{Required: boolPtr(true), MinLength: intPtr(3), MaxLength: intPtr(100), Pattern: stringPtr("^[A-Za-z0-9 .'-]+$"), ValidExample: stringPtr("Product Prime"), InvalidExample: stringPtr("***")}}}
	cfg.Services[0].Entities[0].Fields[0].Type = "ProductName"

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected value object field type to be valid, got %v", err)
	}
}

func TestConfigValidateRejectsValueObjectFieldTypeCasingMismatch(t *testing.T) {
	cfg := validConfig()
	cfg.Services[0].ValueObjects = []ValueObject{{Name: "ProductName", Type: "string", Validations: ValidationRules{Required: boolPtr(true)}}}
	cfg.Services[0].Entities[0].Fields[0].Type = "productname"

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "services[0].entities[0].fields[0].type must be one of") {
		t.Fatalf("expected value object casing mismatch error, got %v", err)
	}
}

func TestConfigValidateRejectsInvalidValueObjectsAndRulesWithActionablePaths(t *testing.T) {
	cfg := validConfig()
	cfg.Services[0].ValueObjects = []ValueObject{
		{Name: "string", Type: "string"},
		{Name: "Product", Type: "ProductName"},
		{Name: "ProductName", Type: "string", Validations: ValidationRules{MinLength: intPtr(5), MaxLength: intPtr(3), Pattern: stringPtr("[")}},
		{Name: "ProductPrice", Type: "decimal", Validations: ValidationRules{Required: boolPtr(true), Minimum: numberPtr("10"), Maximum: numberPtr("1")}},
		{Name: "ProductId", Type: "Guid", Validations: ValidationRules{NotEmpty: boolPtr(true), Pattern: stringPtr(".*")}},
		{Name: "PublishedAt", Type: "DateTime", Validations: ValidationRules{NotDefault: boolPtr(true), Minimum: numberPtr("0")}},
		{Name: "Enabled", Type: "bool", Validations: ValidationRules{NotEmpty: boolPtr(true)}},
	}
	cfg.Services[0].Entities[0].Fields = append(cfg.Services[0].Entities[0].Fields, Field{Name: "Other", Type: "MissingValueObject"})

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	message := err.Error()
	for _, expected := range []string{
		"services[0].valueObjects[0].name must not collide with a supported primitive type",
		"services[0].valueObjects[1].name must not collide with entity",
		"services[0].valueObjects[1].type must be a supported scalar primitive",
		"services[0].valueObjects[2].validations.minLength must be less than or equal to maxLength",
		"services[0].valueObjects[2].validations.pattern must compile as a regular expression",
		"services[0].valueObjects[2].validations.validExample is required when pattern is set",
		"services[0].valueObjects[2].validations.invalidExample is required when pattern is set",
		"services[0].valueObjects[3].validations.required is not applicable to decimal",
		"services[0].valueObjects[3].validations.minimum must be less than or equal to maximum",
		"services[0].valueObjects[4].validations.pattern is not applicable to Guid",
		"services[0].valueObjects[5].validations.minimum is not applicable to DateTime",
		"services[0].valueObjects[6].validations.notEmpty is not applicable to bool",
		"services[0].entities[0].fields[2].type must be one of",
	} {
		if !strings.Contains(message, expected) {
			t.Fatalf("expected error to contain %q, got:\n%s", expected, message)
		}
	}
}

func TestConfigValidateRejectsUnsafeRegexAndNumericBounds(t *testing.T) {
	cfg := validConfig()
	cfg.Services[0].ValueObjects = []ValueObject{
		{Name: "BadPattern", Type: "string", Validations: ValidationRules{Pattern: stringPtr("(?P<name>x)"), ValidExample: stringPtr("x"), InvalidExample: stringPtr("y")}},
		{Name: "InlineMode", Type: "string", Validations: ValidationRules{Pattern: stringPtr("(?U)x"), ValidExample: stringPtr("x"), InvalidExample: stringPtr("y")}},
		{Name: "QuotedPattern", Type: "string", Validations: ValidationRules{Pattern: stringPtr("\\Qx\\E"), ValidExample: stringPtr("x"), InvalidExample: stringPtr("y")}},
		{Name: "PosixClassPattern", Type: "string", Validations: ValidationRules{Pattern: stringPtr("^[[:alpha:]]+$"), ValidExample: stringPtr("x"), InvalidExample: stringPtr("1")}},
		{Name: "NewlinePattern", Type: "string", Validations: ValidationRules{Pattern: stringPtr("^x\n$"), ValidExample: stringPtr("x"), InvalidExample: stringPtr("y")}},
		{Name: "BadEscape", Type: "string", Validations: ValidationRules{Pattern: stringPtr("^\\ax$"), ValidExample: stringPtr("x"), InvalidExample: stringPtr("y")}},
		{Name: "BadInt", Type: "int", Validations: ValidationRules{Minimum: numberPtr("1.5")}},
		{Name: "BadLong", Type: "long", Validations: ValidationRules{Maximum: numberPtr("9223372036854775808")}},
		{Name: "BadDecimal", Type: "decimal", Validations: ValidationRules{Maximum: numberPtr("79228162514264337593543950336")}},
		{Name: "BadDecimalExponent", Type: "decimal", Validations: ValidationRules{Minimum: numberPtr("1e2")}},
		{Name: "BadDecimalScale", Type: "decimal", Validations: ValidationRules{Minimum: numberPtr("0.12345678901234567890123456789")}},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	message := err.Error()
	for _, expected := range []string{"unsupported portable regex construct", "must not contain control characters", "unsupported portable regex escape", "minimum must be an integer literal", "maximum must be within Int64 range", "maximum must be within System.Decimal range", "minimum must fit .NET decimal precision", "minimum must fit .NET decimal precision and scale"} {
		if !strings.Contains(message, expected) {
			t.Fatalf("expected %q in %s", expected, message)
		}
	}
}

func TestConfigValidateAllowsExactMaximumCounts(t *testing.T) {
	cfg := Config{Solution: Solution{Name: "CommercePlatform"}}
	cfg.Services = make([]Service, MaxServices)
	for serviceIndex := range cfg.Services {
		entities := make([]Entity, MaxEntitiesPerService)
		for entityIndex := range entities {
			fields := make([]Field, MaxFieldsPerEntity)
			fields[0] = Field{Name: "Id", Type: "Guid"}
			for fieldIndex := range fields {
				if fieldIndex == 0 {
					continue
				}
				fields[fieldIndex] = Field{Name: fmt.Sprintf("Field%d", fieldIndex), Type: "string"}
			}
			entities[entityIndex] = Entity{Name: fmt.Sprintf("Entity%d", entityIndex), Fields: fields}
		}
		cfg.Services[serviceIndex] = Service{Name: fmt.Sprintf("Service%d", serviceIndex), Entities: entities}
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected exact maximum counts to be valid, got %v", err)
	}
}

func TestConfigValidateRejectsOverMaximumEntityAndFieldCounts(t *testing.T) {
	fields := make([]Field, MaxFieldsPerEntity+1)
	for index := range fields {
		fields[index] = Field{Name: fmt.Sprintf("Field%d", index), Type: "string"}
	}
	entities := make([]Entity, MaxEntitiesPerService+1)
	for index := range entities {
		entities[index] = Entity{Name: fmt.Sprintf("Entity%d", index), Fields: []Field{{Name: "Id", Type: "Guid"}}}
	}
	entities[0].Fields = fields
	cfg := Config{
		Solution: Solution{Name: "CommercePlatform"},
		Services: []Service{{Name: "ProductService", Entities: entities}},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	message := err.Error()
	for _, expected := range []string{
		"services[0].entities must contain at most 100 items",
		"services[0].entities[0].fields must contain at most 100 items",
	} {
		if !strings.Contains(message, expected) {
			t.Fatalf("expected error to contain %q, got:\n%s", expected, message)
		}
	}
}

func validConfig() Config {
	return Config{
		Solution: Solution{Name: "CommercePlatform", Description: "Product management."},
		Services: []Service{
			{
				Name: "ProductService",
				Entities: []Entity{
					{
						Name: "Product",
						Fields: []Field{
							{Name: "Name", Type: "string"},
							{Name: "Id", Type: "Guid"},
						},
					},
				},
			},
		},
	}
}

func boolPtr(value bool) *bool            { return &value }
func intPtr(value int) *int               { return &value }
func stringPtr(value string) *string      { return &value }
func numberPtr(value string) *json.Number { number := json.Number(value); return &number }
