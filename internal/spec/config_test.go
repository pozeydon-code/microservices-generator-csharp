package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

func TestConfigGatewayOptionsDefaultDisabledAndParseEnabled(t *testing.T) {
	tests := []struct {
		name string
		data string
		want bool
	}{
		{
			name: "omitted gateway remains disabled",
			data: `{
				"solution":{"name":"CommercePlatform","description":"Product management."},
				"services":[{"name":"ProductService","entities":[{"name":"Product","fields":[{"name":"Id","type":"Guid"}]}]}]
			}`,
			want: false,
		},
		{
			name: "enabled gateway parses true",
			data: `{
				"generation":{"gateway":{"enabled":true}},
				"solution":{"name":"CommercePlatform","description":"Product management."},
				"services":[{"name":"ProductService","entities":[{"name":"Product","fields":[{"name":"Id","type":"Guid"}]}]}]
			}`,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg Config
			if err := json.Unmarshal([]byte(tt.data), &cfg); err != nil {
				t.Fatalf("unmarshal config: %v", err)
			}
			if err := cfg.Validate(); err != nil {
				t.Fatalf("expected gateway config to validate, got %v", err)
			}
			if cfg.Generation.Gateway.Enabled != tt.want {
				t.Fatalf("expected gateway enabled %t, got %t", tt.want, cfg.Generation.Gateway.Enabled)
			}
		})
	}
}

func TestConfigValidateAcceptsSupportedTargetFrameworks(t *testing.T) {
	for _, targetFramework := range []string{"net8.0", "net9.0", "net10.0"} {
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

func TestConfigValidateRejectsTargetFrameworksBelowNet8(t *testing.T) {
	for _, targetFramework := range []string{"net1.0", "net6.0", "net7.0"} {
		t.Run(targetFramework, func(t *testing.T) {
			cfg := validConfig()
			cfg.Generation.TargetFramework = targetFramework

			err := cfg.Validate()
			if err == nil {
				t.Fatal("expected validation error")
			}
			const expected = "generation.targetFramework must be net8.0 or newer (netN.0 with a numeric major version from 8 through 99)"
			if !strings.Contains(err.Error(), expected) {
				t.Fatalf("expected error to contain %q, got %v", expected, err)
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
	for _, expected := range []string{"schemaVersion must be 1", "generation.targetFramework must be net8.0 or newer"} {
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

func TestSupportedTargetFrameworksStartAtNet8(t *testing.T) {
	want := []string{"net10.0", "net9.0", "net8.0"}
	if got := SupportedTargetFrameworks(); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("expected supported target frameworks %v, got %v", want, got)
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
	tests := []string{"Product", "ProductDto", "CreateProductRequest", "UpdateProductRequest", "IProductRepository", "ProductRepository", "CreateProductCommand", "CreateProductCommandValidator", "UpdateProductCommand", "UpdateProductCommandValidator", "DeleteProductCommand", "DeleteProductCommandValidator", "ListProductQuery", "ListProductQueryValidator", "GetProductByIdQuery", "ProductController"}
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

func TestDocumentedValueObjectExamplesValidateAgainstTheirRules(t *testing.T) {
	readme, err := os.ReadFile(filepath.Join("..", "..", "README.md"))
	if err != nil {
		t.Fatalf("read README: %v", err)
	}
	for _, expected := range []string{`"validExample": "Product Prime"`, `"invalidExample": "***"`} {
		if !strings.Contains(string(readme), expected) {
			t.Fatalf("README is missing documented value-object example %q", expected)
		}
	}

	cfg := validConfig()
	cfg.Services[0].ValueObjects = []ValueObject{{
		Name: "ProductName",
		Type: "string",
		Validations: ValidationRules{
			Required:       boolPtr(true),
			MinLength:      intPtr(3),
			MaxLength:      intPtr(100),
			Pattern:        stringPtr("^[A-Za-z0-9 .'-]+$"),
			ValidExample:   stringPtr("Product Prime"),
			InvalidExample: stringPtr("***"),
		},
	}}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected documented value-object examples to validate, got %v", err)
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

func TestConfigValidateAcceptsRelationshipsWithDefaultsAndCanonicalEdges(t *testing.T) {
	tests := []struct {
		name         string
		multiplicity string
	}{
		{name: "one-to-many input", multiplicity: "one-to-many"},
		{name: "many-to-one input", multiplicity: "many-to-one"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validRelationshipConfig()
			cfg.Services[0].Relationships = []Relationship{{
				Name:                "OrderItems",
				Multiplicity:        tt.multiplicity,
				PrincipalEntity:     "Order",
				DependentEntity:     "OrderItem",
				ForeignKeyName:      "OrderId",
				PrincipalNavigation: "Items",
				DependentNavigation: "Order",
			}}

			if err := cfg.Validate(); err != nil {
				t.Fatalf("expected relationship config to validate, got %v", err)
			}
			got := cfg.Services[0].CanonicalRelationships()
			if len(got) != 1 {
				t.Fatalf("expected 1 canonical relationship, got %d", len(got))
			}
			want := CanonicalRelationship{
				Name:                "OrderItems",
				PrincipalEntity:     "Order",
				DependentEntity:     "OrderItem",
				ForeignKeyName:      "OrderId",
				ForeignKeyType:      "Guid",
				Required:            true,
				PrincipalNavigation: "Items",
				DependentNavigation: "Order",
			}
			if got[0] != want {
				t.Fatalf("canonical relationship = %+v; want %+v", got[0], want)
			}
		})
	}
}

func TestConfigValidateAcceptsFlatConfigsWithoutRelationshipsUnchanged(t *testing.T) {
	cfg := validConfig()

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected flat config to validate, got %v", err)
	}
	if got := cfg.Services[0].CanonicalRelationships(); len(got) != 0 {
		t.Fatalf("expected no canonical relationships for flat config, got %+v", got)
	}
}

func TestConfigValidatePreservesOptionalRelationshipNullabilityPolicy(t *testing.T) {
	cfg := validRelationshipConfig()
	cfg.Services[0].Relationships = []Relationship{{
		Multiplicity:        "one-to-many",
		PrincipalEntity:     "Order",
		DependentEntity:     "OrderItem",
		ForeignKeyName:      "OrderId",
		Required:            boolPtr(false),
		PrincipalNavigation: "Items",
		DependentNavigation: "Order",
	}}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected optional relationship config to validate, got %v", err)
	}
	got := cfg.Services[0].CanonicalRelationships()
	if len(got) != 1 {
		t.Fatalf("expected 1 canonical relationship, got %d", len(got))
	}
	if got[0].Required || !got[0].Nullable() {
		t.Fatalf("expected optional relationship to be represented as nullable, got %+v", got[0])
	}
}

func TestConfigValidateRejectsReservedRelationshipForeignKeyNames(t *testing.T) {
	tests := []struct {
		name           string
		foreignKeyName string
		reservedName   string
	}{
		{name: "id", foreignKeyName: "Id", reservedName: "Id"},
		{name: "id case-insensitive", foreignKeyName: "id", reservedName: "Id"},
		{name: "row version", foreignKeyName: "RowVersion", reservedName: "RowVersion"},
		{name: "row version case-insensitive", foreignKeyName: "rowversion", reservedName: "RowVersion"},
		{name: "concurrency token", foreignKeyName: "ConcurrencyToken", reservedName: "ConcurrencyToken"},
		{name: "concurrency token case-insensitive", foreignKeyName: "concurrencytoken", reservedName: "ConcurrencyToken"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validRelationshipConfig()
			cfg.Services[0].Relationships = []Relationship{{
				Multiplicity:        "one-to-many",
				PrincipalEntity:     "Order",
				DependentEntity:     "OrderItem",
				ForeignKeyName:      tt.foreignKeyName,
				PrincipalNavigation: "Items",
				DependentNavigation: "Order",
			}}

			err := cfg.Validate()
			if err == nil {
				t.Fatal("expected validation error")
			}
			expected := fmt.Sprintf("relationships[0].foreignKeyName %s is reserved for generated %s members", tt.foreignKeyName, tt.reservedName)
			if !strings.Contains(err.Error(), expected) {
				t.Fatalf("expected error to contain %q, got %v", expected, err)
			}
		})
	}
}

func TestConfigValidateRejectsInvalidRelationshipMetadata(t *testing.T) {
	tests := []struct {
		name        string
		mutate      func(*Config)
		expectedErr string
	}{
		{
			name: "unsupported multiplicity",
			mutate: func(cfg *Config) {
				cfg.Services[0].Relationships[0].Multiplicity = "many-to-many"
			},
			expectedErr: "relationships[0].multiplicity must be one-to-many or many-to-one",
		},
		{
			name: "missing endpoint",
			mutate: func(cfg *Config) {
				cfg.Services[0].Relationships[0].DependentEntity = "Invoice"
			},
			expectedErr: "relationships[0].dependentEntity must reference an entity in service ProductService",
		},
		{
			name: "same endpoint",
			mutate: func(cfg *Config) {
				cfg.Services[0].Relationships[0].DependentEntity = "Order"
			},
			expectedErr: "relationships[0] principalEntity and dependentEntity must be different",
		},
		{
			name: "duplicate inverse declaration",
			mutate: func(cfg *Config) {
				cfg.Services[0].Relationships = append(cfg.Services[0].Relationships, Relationship{
					Multiplicity:        "many-to-one",
					PrincipalEntity:     "Order",
					DependentEntity:     "OrderItem",
					ForeignKeyName:      "OrderId",
					PrincipalNavigation: "Lines",
					DependentNavigation: "Order",
				})
			},
			expectedErr: "relationships[1] duplicates canonical relationship Order-OrderItem-OrderId",
		},
		{
			name: "foreign key type collision",
			mutate: func(cfg *Config) {
				cfg.Services[0].Entities[1].Fields[1].Type = "string"
			},
			expectedErr: "relationships[0].foreignKeyName OrderId must have type Guid",
		},
		{
			name: "navigation collides with dependent field",
			mutate: func(cfg *Config) {
				cfg.Services[0].Relationships[0].DependentNavigation = "Sku"
			},
			expectedErr: "relationships[0].dependentNavigation must not collide with a field on OrderItem",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validRelationshipConfig()
			cfg.Services[0].Relationships = []Relationship{{
				Multiplicity:        "one-to-many",
				PrincipalEntity:     "Order",
				DependentEntity:     "OrderItem",
				ForeignKeyName:      "OrderId",
				PrincipalNavigation: "Items",
				DependentNavigation: "Order",
			}}
			tt.mutate(&cfg)

			err := cfg.Validate()
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tt.expectedErr) {
				t.Fatalf("expected error to contain %q, got %v", tt.expectedErr, err)
			}
		})
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

func validRelationshipConfig() Config {
	return Config{
		Solution: Solution{Name: "CommercePlatform", Description: "Order management."},
		Services: []Service{{
			Name: "ProductService",
			Entities: []Entity{
				{Name: "Order", Fields: []Field{{Name: "Id", Type: "Guid"}, {Name: "Number", Type: "string"}}},
				{Name: "OrderItem", Fields: []Field{{Name: "Id", Type: "Guid"}, {Name: "OrderId", Type: "Guid"}, {Name: "Sku", Type: "string"}}},
			},
		}},
	}
}

func boolPtr(value bool) *bool            { return &value }
func intPtr(value int) *int               { return &value }
func stringPtr(value string) *string      { return &value }
func numberPtr(value string) *json.Number { number := json.Number(value); return &number }
