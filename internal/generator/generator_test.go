package generator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/pozeydon-code/generator-microservices-go/internal/spec"
)

func TestGenerateProducesDeterministicGoldenOutput(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}

	first, err := gen.Generate(testConfig())
	if err != nil {
		t.Fatalf("generate first: %v", err)
	}
	second, err := gen.Generate(testConfig())
	if err != nil {
		t.Fatalf("generate second: %v", err)
	}

	if len(first) != len(second) {
		t.Fatalf("expected same file count, got %d and %d", len(first), len(second))
	}
	for index := range first {
		if first[index].Path != second[index].Path || !bytes.Equal(first[index].Content, second[index].Content) {
			t.Fatalf("generation is not deterministic at index %d", index)
		}
	}

	expectedFiles := []struct {
		path       string
		goldenName string
	}{
		{path: "CommercePlatform.sln", goldenName: "CommercePlatform.sln"},
		{path: "Directory.Build.props", goldenName: "Directory.Build.props"},
		{path: "Directory.Packages.props", goldenName: "Directory.Packages.props"},
		{path: "README.md", goldenName: "README.md"},
		{path: "microgen.json", goldenName: "microgen.json"},
		{path: "src/ProductService/ProductService.Api/Common/ValidationProblemMapper.cs", goldenName: "ValidationProblemMapper.cs"},
		{path: "src/ProductService/ProductService.Api/Features/Products/ProductEndpoints.cs", goldenName: "ProductEndpoints.cs"},
		{path: "src/ProductService/ProductService.Api/Health/HealthEndpoints.cs", goldenName: "HealthEndpoints.cs"},
		{path: "src/ProductService/ProductService.Api/ProductService.Api.csproj", goldenName: "ProductService.Api.csproj"},
		{path: "src/ProductService/ProductService.Application/Common/PaginationPolicy.cs", goldenName: "PaginationPolicy.cs"},
		{path: "src/ProductService/ProductService.Application/Common/Readiness.cs", goldenName: "Readiness.cs"},
		{path: "src/ProductService/ProductService.Application/Common/Results.cs", goldenName: "Results.cs"},
		{path: "src/ProductService/ProductService.Application/Features/Products/IProductRepository.cs", goldenName: "IProductRepository.cs"},
		{path: "src/ProductService/ProductService.Application/Features/Products/IProductUseCases.cs", goldenName: "IProductUseCases.cs"},
		{path: "src/ProductService/ProductService.Application/Features/Products/ProductContracts.cs", goldenName: "ProductContracts.cs"},
		{path: "src/ProductService/ProductService.Application/Features/Products/ProductUseCases.cs", goldenName: "ProductUseCases.cs"},
		{path: "src/ProductService/ProductService.Application/ProductService.Application.csproj", goldenName: "ProductService.Application.csproj"},
		{path: "src/ProductService/ProductService.Domain/Features/Products/Product.cs", goldenName: "Product.cs"},
		{path: "src/ProductService/ProductService.Domain/ProductService.Domain.csproj", goldenName: "ProductService.Domain.csproj"},
		{path: "src/ProductService/ProductService.Domain/Shared/DomainReconstitutionException.cs", goldenName: "DomainReconstitutionException.cs"},
		{path: "src/ProductService/ProductService.Domain/Shared/DomainResult.cs", goldenName: "DomainResult.cs"},
		{path: "src/ProductService/ProductService.Domain/Shared/ValueObjects/ProductName.cs", goldenName: "ProductName.cs"},
		{path: "src/ProductService/ProductService.Domain/Shared/ValueObjects/ProductPrice.cs", goldenName: "ProductPrice.cs"},
		{path: "src/ProductService/ProductService.Host/ProductService.Host.csproj", goldenName: "ProductService.Host.csproj"},
		{path: "src/ProductService/ProductService.Host/Program.cs", goldenName: "Program.cs"},
		{path: "src/ProductService/ProductService.Host/appsettings.Development.json", goldenName: "appsettings.Development.json"},
		{path: "src/ProductService/ProductService.Host/appsettings.json", goldenName: "appsettings.json"},
		{path: "src/ProductService/ProductService.Infrastructure/DependencyInjection.cs", goldenName: "Infrastructure.DependencyInjection.cs"},
		{path: "src/ProductService/ProductService.Infrastructure/Persistence/Features/Products/ProductRepository.cs", goldenName: "ProductRepository.cs"},
		{path: "src/ProductService/ProductService.Infrastructure/Persistence/ProductServiceDbContext.cs", goldenName: "ProductServiceDbContext.cs"},
		{path: "src/ProductService/ProductService.Infrastructure/Persistence/ValueObjectPreflight.sql", goldenName: "ValueObjectPreflight.sql"},
		{path: "src/ProductService/ProductService.Infrastructure/ProductService.Infrastructure.csproj", goldenName: "ProductService.Infrastructure.csproj"},
		{path: "tests/ProductService/ProductService.Api.Tests/AuthenticationTests.cs", goldenName: "AuthenticationTests.cs"},
		{path: "tests/ProductService/ProductService.Api.Tests/Features/Products/ProductEndpointsTests.cs", goldenName: "ProductEndpointsTests.cs"},
		{path: "tests/ProductService/ProductService.Api.Tests/HealthEndpointsTests.cs", goldenName: "HealthEndpointsTests.cs"},
		{path: "tests/ProductService/ProductService.Api.Tests/ProductService.Api.Tests.csproj", goldenName: "ProductService.Api.Tests.csproj"},
		{path: "tests/ProductService/ProductService.Api.Tests/TestApiFactory.cs", goldenName: "TestApiFactory.cs"},
		{path: "tests/ProductService/ProductService.Api.Tests/TestJwtTokens.cs", goldenName: "TestJwtTokens.cs"},
		{path: "tests/ProductService/ProductService.Application.Tests/Features/Products/ProductUseCasesTests.cs", goldenName: "ProductUseCasesTests.cs"},
		{path: "tests/ProductService/ProductService.Application.Tests/ProductService.Application.Tests.csproj", goldenName: "ProductService.Application.Tests.csproj"},
		{path: "tests/ProductService/ProductService.Architecture.Tests/ProductService.Architecture.Tests.csproj", goldenName: "ProductService.Architecture.Tests.csproj"},
		{path: "tests/ProductService/ProductService.Architecture.Tests/ProductServiceArchitectureTests.cs", goldenName: "ProductServiceArchitectureTests.cs"},
		{path: "tests/ProductService/ProductService.Domain.Tests/ProductService.Domain.Tests.csproj", goldenName: "ProductService.Domain.Tests.csproj"},
		{path: "tests/ProductService/ProductService.Domain.Tests/ProductServiceDomainTests.cs", goldenName: "ProductServiceDomainTests.cs"},
		{path: "tests/ProductService/ProductService.Infrastructure.Tests/ProductService.Infrastructure.Tests.csproj", goldenName: "ProductService.Infrastructure.Tests.csproj"},
		{path: "tests/ProductService/ProductService.Infrastructure.Tests/ProductServiceInfrastructureTests.cs", goldenName: "ProductServiceInfrastructureTests.cs"},
	}
	actualPaths := make([]string, 0, len(first))
	for _, file := range first {
		actualPaths = append(actualPaths, file.Path)
	}
	expectedPaths := make([]string, 0, len(expectedFiles))
	for _, file := range expectedFiles {
		expectedPaths = append(expectedPaths, file.path)
	}
	if !reflect.DeepEqual(actualPaths, expectedPaths) {
		t.Fatalf("generated path set mismatch\nexpected: %#v\nactual:   %#v", expectedPaths, actualPaths)
	}
	for _, file := range expectedFiles {
		assertGoldenFile(t, first, file.path, file.goldenName)
	}
}

func TestGeneratePreservesLayerDependenciesAndSafetyBoundaries(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	files, err := gen.Generate(testConfig())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	contentByPath := map[string]string{}
	for _, file := range files {
		contentByPath[file.Path] = string(file.Content)
	}

	assertContains(t, contentByPath["src/ProductService/ProductService.Application/ProductService.Application.csproj"], "ProductService.Domain.csproj")
	assertNotContains(t, contentByPath["src/ProductService/ProductService.Application/ProductService.Application.csproj"], "ProductService.Infrastructure")
	assertNotContains(t, contentByPath["src/ProductService/ProductService.Application/ProductService.Application.csproj"], "ProductService.Api")
	assertContains(t, contentByPath["src/ProductService/ProductService.Infrastructure/ProductService.Infrastructure.csproj"], "ProductService.Application.csproj")
	assertContains(t, contentByPath["src/ProductService/ProductService.Domain/Shared/ValueObjects/ProductName.cs"], "DomainResult<ProductName>")
	assertContains(t, contentByPath["src/ProductService/ProductService.Infrastructure/Persistence/ProductServiceDbContext.cs"], "HasConversion(value => value.Value, value => ProductName.Rehydrate(value))")
	assertContains(t, contentByPath["Directory.Packages.props"], "Microsoft.EntityFrameworkCore.SqlServer\" Version=\"8.0.28")
	assertContains(t, contentByPath["Directory.Packages.props"], "Microsoft.AspNetCore.Mvc.Testing\" Version=\"8.0.28")
	assertContains(t, contentByPath["src/ProductService/ProductService.Infrastructure/ProductService.Infrastructure.csproj"], "Microsoft.EntityFrameworkCore.SqlServer")
	assertContains(t, contentByPath["src/ProductService/ProductService.Host/ProductService.Host.csproj"], "ProductService.Infrastructure.csproj")
	assertContains(t, contentByPath["src/ProductService/ProductService.Host/ProductService.Host.csproj"], "ProductService.Api.csproj")
	assertNotContains(t, contentByPath["src/ProductService/ProductService.Api/ProductService.Api.csproj"], "ProductService.Infrastructure.csproj")
	assertNotContains(t, contentByPath["src/ProductService/ProductService.Api/ProductService.Api.csproj"], "Microsoft.AspNetCore.Mvc.Testing")
	assertNotContains(t, contentByPath["src/ProductService/ProductService.Api/ProductService.Api.csproj"], "Version=")
	program := contentByPath["src/ProductService/ProductService.Host/Program.cs"]
	assertContains(t, program, "AddInfrastructure(builder.Configuration)")
	assertContains(t, program, "MapHealthEndpoints")
	infraDI := contentByPath["src/ProductService/ProductService.Infrastructure/DependencyInjection.cs"]
	assertContains(t, infraDI, "IReadinessProbe")
	assertContains(t, infraDI, "public const int SqlConnectionTimeoutSeconds = 2")
	assertContains(t, infraDI, "public const int SqlCommandTimeoutSeconds = 2")
	assertContains(t, infraDI, "public const int SqlRetryCount = 1")
	assertContains(t, infraDI, "public const int ReadinessTimeoutSeconds = 2")
	assertContains(t, infraDI, "readinessCts.CancelAfter(TimeSpan.FromSeconds(ResiliencePolicy.ReadinessTimeoutSeconds))")
	assertContains(t, infraDI, "command.CommandTimeout = ResiliencePolicy.SqlCommandTimeoutSeconds")
	assertContains(t, infraDI, "ExpectedSchemaExistsAsync")
	assertContains(t, infraDI, "INFORMATION_SCHEMA.COLUMNS")
	assertContains(t, infraDI, "syscolumns.system_type_id = 189")
	assertNotContains(t, program, "CanConnectAsync")
	assertNotContains(t, program, "foreach (var tableName")
	assertNotContains(t, infraDI, "MapGet")
	assertNotContains(t, infraDI, "Results.StatusCode")

	for path, content := range contentByPath {
		for _, forbidden := range []string{"EnsureCreated", "Migrate(", "Database.Migrate", "Password=", "User Id=", "Server=localhost"} {
			if strings.HasPrefix(path, "tests/") && forbidden == "EnsureCreated" {
				continue
			}
			if strings.HasSuffix(path, ".md") && (forbidden == "Password=" || forbidden == "User Id=" || forbidden == "Server=localhost") {
				continue
			}
			if strings.Contains(content, forbidden) {
				t.Fatalf("generated file %s contains forbidden text %q", path, forbidden)
			}
		}
	}
}

func TestGenerateRejectsReservedRowVersionFieldBeforeWritingFiles(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	cfg := testConfig()
	cfg.Services[0].Entities[0].Fields = append(cfg.Services[0].Entities[0].Fields, spec.Field{Name: "RowVersion", Type: "string"})

	files, err := gen.Generate(cfg)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if len(files) != 0 {
		t.Fatalf("expected no generated files after validation failure, got %d", len(files))
	}
	if !strings.Contains(err.Error(), "fields[4].name is reserved for infrastructure concurrency storage") {
		t.Fatalf("expected RowVersion collision error, got %v", err)
	}
}

func TestGenerateRejectsCaseInsensitiveConcurrencyTokenJsonContractCollisionBeforeWritingFiles(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	cfg := testConfig()
	cfg.Services[0].Entities[0].Fields = append(cfg.Services[0].Entities[0].Fields, spec.Field{Name: "concurrencyToken", Type: "string"})

	files, err := gen.Generate(cfg)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if len(files) != 0 {
		t.Fatalf("expected no generated files after validation failure, got %d", len(files))
	}
	if !strings.Contains(err.Error(), "fields[4].name must not collide case-insensitively with generated JSON contract field \"ConcurrencyToken\"") {
		t.Fatalf("expected concurrencyToken JSON contract collision error, got %v", err)
	}
}

func TestGenerateCreateTestsAssertAllSupportedScalarRequestFields(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	cfg := testConfig()
	cfg.Services[0].Entities[0].Fields = []spec.Field{
		{Name: "Id", Type: "Guid"},
		{Name: "IsAvailable", Type: "bool"},
		{Name: "PublishedAt", Type: "DateTime"},
		{Name: "Price", Type: "decimal"},
		{Name: "Score", Type: "double"},
		{Name: "ExternalId", Type: "Guid"},
		{Name: "Quantity", Type: "int"},
		{Name: "Inventory", Type: "long"},
		{Name: "Name", Type: "string"},
	}

	files, err := gen.Generate(cfg)
	if err != nil {
		t.Fatalf("generate all-scalar config: %v", err)
	}
	content := string(generatedContent(t, files, "tests/ProductService/ProductService.Application.Tests/Features/Products/ProductUseCasesTests.cs"))
	for _, expected := range []string{
		"Assert.True(created.IsAvailable);",
		"Assert.Equal(new DateTime(2024, 1, 1, 0, 0, 0, DateTimeKind.Utc), created.PublishedAt);",
		"Assert.Equal(12.34m, created.Price);",
		"Assert.Equal(12.34d, created.Score);",
		"Assert.Equal(Guid.Parse(\"00000000-0000-0000-0000-000000000001\"), created.ExternalId);",
		"Assert.Equal(12, created.Quantity);",
		"Assert.Equal(12L, created.Inventory);",
		"Assert.Equal(\"Name Value\", created.Name);",
	} {
		assertContains(t, content, expected)
	}
}

func TestGenerateRecordEntityTestsUseDomainAliases(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}

	tests := []struct {
		name string
		cfg  spec.Config
	}{
		{name: "zero value objects", cfg: recordCollisionConfig(false)},
		{name: "value object backed", cfg: recordCollisionConfig(true)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := gen.Generate(tt.cfg)
			if err != nil {
				t.Fatalf("generate: %v", err)
			}

			applicationTests := string(generatedContent(t, files, "tests/RecordService/RecordService.Application.Tests/Features/Records/RecordUseCasesTests.cs"))
			infrastructureTests := string(generatedContent(t, files, "tests/RecordService/RecordService.Infrastructure.Tests/RecordServiceInfrastructureTests.cs"))

			for _, content := range []string{applicationTests, infrastructureTests} {
				assertContains(t, content, "using DomainRecord = RecordService.Domain.Features.Records.Record;")
			}
			for _, expected := range []string{
				"DomainRecord.Create(new RecordState",
				"List<DomainRecord>",
				"EntitySnapshot<DomainRecord>",
				"AddAsync(DomainRecord entity",
				"UpdateAsync(DomainRecord entity",
				"DeleteAsync(DomainRecord entity",
			} {
				assertContains(t, applicationTests, expected)
			}
			assertContains(t, infrastructureTests, "DomainRecord.Create(new RecordState")
		})
	}
}

func TestGenerateUsesParameterizedRepositoryDiagnostics(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	files, err := gen.Generate(testConfig())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	repository := string(generatedContent(t, files, "src/ProductService/ProductService.Infrastructure/Persistence/Features/Products/ProductRepository.cs"))
	assertContains(t, repository, "SqlQueryRaw<Guid>(sql, id.Value)")
	assertNotContains(t, repository, "$\"SELECT")
	assertNotContains(t, repository, "'{id.Value}'")
}

func TestDoubleInvalidWitnessesUseAdjacentRepresentableValuesAndOmitExtrema(t *testing.T) {
	lower := lowerBoundInvalid("double", numberLiteralFor("double", "1e308"))
	upper := upperBoundInvalid("double", numberLiteralFor("double", "1e308"))
	if lower == "" || upper == "" {
		t.Fatalf("expected adjacent witnesses, got lower=%q upper=%q", lower, upper)
	}
	for _, witness := range []string{lower, upper} {
		assertContains(t, witness, "d")
		assertNotContains(t, witness, " - 1")
		assertNotContains(t, witness, " + 1")
	}
	if got := lowerBoundInvalid("double", numberLiteralFor("double", "-1.7976931348623157e308")); got != "" {
		t.Fatalf("expected no lower witness below double minimum, got %q", got)
	}
	if got := upperBoundInvalid("double", numberLiteralFor("double", "1.7976931348623157e308")); got != "" {
		t.Fatalf("expected no upper witness above double maximum, got %q", got)
	}
}

func TestCSharpStringLiteralUsesFixedWidthEscapes(t *testing.T) {
	actual := csharpStringLiteral("quote\" slash\\ control\x1F adjacent\x01A café 😀")
	expected := "\"quote\\\" slash\\\\ control\\u001F adjacent\\u0001A caf\\u00E9 \\U0001F600\""
	if actual != expected {
		t.Fatalf("unexpected C# literal\nexpected: %s\nactual:   %s", expected, actual)
	}
}

func TestGenerateRequestContractsUseInitPropertiesForHundredFields(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	cfg := hundredFieldConfig()
	files, err := gen.Generate(cfg)
	if err != nil {
		t.Fatalf("generate hundred-field config: %v", err)
	}
	contracts := string(generatedContent(t, files, "src/HundredFieldService/HundredFieldService.Application/Features/HundredRecords/HundredRecordContracts.cs"))
	assertContains(t, contracts, "public sealed record CreateHundredRecordRequest\n{")
	assertContains(t, contracts, "public string Field099 { get; init; } = string.Empty;")
	assertNotContains(t, contracts, "public sealed record CreateHundredRecordRequest(")
	assertNotContains(t, contracts, "public sealed record UpdateHundredRecordRequest(")
	tests := string(generatedContent(t, files, "tests/HundredFieldService/HundredFieldService.Application.Tests/Features/HundredRecords/HundredRecordUseCasesTests.cs"))
	assertContains(t, tests, "CreateAsync(new() { Field001 =")
	assertContains(t, tests, "ConcurrencyToken = validToken")
	assertNotContains(t, tests, "new UpdateHundredRecordRequest(")
}

func TestGenerateEscapedStringValueObjectsAndLargeDoubleWitnesses(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	cfg := testConfig()
	cfg.Services[0].ValueObjects = []spec.ValueObject{
		{Name: "EscapedName", Type: "string", Validations: spec.ValidationRules{Required: boolPtr(true), MinLength: intPtr(1), MaxLength: intPtr(80), Pattern: stringPtr("^[A-Za-z0-9 .\\-]+$"), ValidExample: stringPtr("Cafe Unicode 1"), InvalidExample: stringPtr("bad\x1Fé😀")}},
		{Name: "HugeScore", Type: "double", Validations: spec.ValidationRules{Minimum: numberPtr("1e308"), Maximum: numberPtr("1.7976931348623157e308")}},
	}
	cfg.Services[0].Entities[0].Fields = []spec.Field{{Name: "Id", Type: "Guid"}, {Name: "Name", Type: "EscapedName"}, {Name: "Score", Type: "HugeScore"}}
	files, err := gen.Generate(cfg)
	if err != nil {
		t.Fatalf("generate escaped/double config: %v", err)
	}
	domainTests := string(generatedContent(t, files, "tests/ProductService/ProductService.Domain.Tests/ProductServiceDomainTests.cs"))
	assertContains(t, domainTests, "\"bad\\u001F\\u00E9\\U0001F600\"")
	assertContains(t, domainTests, lowerBoundInvalid("double", numberLiteralFor("double", "1e308")))
	assertNotContains(t, domainTests, "1E+308d - 1d")
}

func TestGeneratePreflightDoesNotTreatOptionalStringValueObjectsAsRequired(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	files, err := gen.Generate(optionalMaxLengthStringConfig())
	if err != nil {
		t.Fatalf("generate optional string config: %v", err)
	}
	preflight := string(generatedContent(t, files, "src/OptionalStringService/OptionalStringService.Infrastructure/Persistence/ValueObjectPreflight.sql"))
	assertContains(t, preflight, "OptionalLabel.MaxLength")
	assertNotContains(t, preflight, "OptionalLabel.Required")
	infraTests := string(generatedContent(t, files, "tests/OptionalStringService/OptionalStringService.Infrastructure.Tests/OptionalStringServiceInfrastructureTests.cs"))
	assertContains(t, infraTests, "VALUES ({0}, N''")
	assertContains(t, infraTests, "Assert.Empty(await RunPreflightAsync(context));")
}

func assertContains(t *testing.T, content, expected string) {
	t.Helper()
	if !strings.Contains(content, expected) {
		t.Fatalf("expected generated content to contain %q\ncontent:\n%s", expected, content)
	}
}

func assertNotContains(t *testing.T, content, unexpected string) {
	t.Helper()
	if strings.Contains(content, unexpected) {
		t.Fatalf("expected generated content not to contain %q\ncontent:\n%s", unexpected, content)
	}
}

func assertGoldenFile(t *testing.T, files []GeneratedFile, generatedPath, goldenName string) {
	t.Helper()
	var actual []byte
	for _, file := range files {
		if file.Path == generatedPath {
			actual = file.Content
			break
		}
	}
	if actual == nil {
		t.Fatalf("generated file %s not found", generatedPath)
	}
	expected, err := os.ReadFile(filepath.Join("testdata", "golden", goldenName))
	if err != nil {
		t.Fatalf("read golden file: %v", err)
	}
	if !bytes.Equal(actual, expected) {
		t.Fatalf("golden mismatch for %s\nexpected:\n%s\nactual:\n%s", generatedPath, expected, actual)
	}
}

func generatedContent(t *testing.T, files []GeneratedFile, generatedPath string) []byte {
	t.Helper()
	for _, file := range files {
		if file.Path == generatedPath {
			return file.Content
		}
	}
	t.Fatalf("generated file %s not found", generatedPath)
	return nil
}

func testConfig() spec.Config {
	return spec.Config{
		Solution: spec.Solution{Name: "CommercePlatform", Description: "Minimal generated .NET 8 microservice workspace for product management."},
		Services: []spec.Service{
			{
				Name: "ProductService",
				ValueObjects: []spec.ValueObject{
					{Name: "ProductName", Type: "string", Validations: spec.ValidationRules{Required: boolPtr(true), MinLength: intPtr(3), MaxLength: intPtr(100), Pattern: stringPtr("^[A-Za-z0-9 .'-]+$"), ValidExample: stringPtr("Product Prime"), InvalidExample: stringPtr("***")}},
					{Name: "ProductPrice", Type: "decimal", Validations: spec.ValidationRules{Minimum: numberPtr("0"), Maximum: numberPtr("999999.99")}},
				},
				Entities: []spec.Entity{
					{
						Name: "Product",
						Fields: []spec.Field{
							{Name: "Id", Type: "Guid"},
							{Name: "Name", Type: "ProductName"},
							{Name: "Price", Type: "ProductPrice"},
							{Name: "IsActive", Type: "bool"},
						},
					},
				},
			},
		},
	}
}

func recordCollisionConfig(withValueObject bool) spec.Config {
	fields := []spec.Field{
		{Name: "Id", Type: "Guid"},
		{Name: "Title", Type: "string"},
		{Name: "Enabled", Type: "bool"},
	}
	valueObjects := []spec.ValueObject(nil)
	if withValueObject {
		valueObjects = []spec.ValueObject{{Name: "RecordTitle", Type: "string", Validations: spec.ValidationRules{Required: boolPtr(true), MinLength: intPtr(3), MaxLength: intPtr(80), ValidExample: stringPtr("Valid Record"), InvalidExample: stringPtr("")}}}
		fields[1].Type = "RecordTitle"
	}
	return spec.Config{
		Solution: spec.Solution{Name: "RecordPlatform", Description: "Record collision regression."},
		Services: []spec.Service{
			{
				Name:         "RecordService",
				ValueObjects: valueObjects,
				Entities: []spec.Entity{
					{
						Name:   "Record",
						Fields: fields,
					},
				},
			},
		},
	}
}

func hundredFieldConfig() spec.Config {
	fields := []spec.Field{{Name: "Id", Type: "Guid"}}
	for index := 1; index < 100; index++ {
		fields = append(fields, spec.Field{Name: fmt.Sprintf("Field%03d", index), Type: "string"})
	}
	return spec.Config{
		Solution: spec.Solution{Name: "HundredFieldPlatform", Description: "Hundred field regression."},
		Services: []spec.Service{{
			Name: "HundredFieldService",
			Entities: []spec.Entity{{
				Name:   "HundredRecord",
				Fields: fields,
			}},
		}},
	}
}

func optionalMaxLengthStringConfig() spec.Config {
	return spec.Config{
		Solution: spec.Solution{Name: "OptionalStringPlatform", Description: "Optional string preflight regression."},
		Services: []spec.Service{{
			Name: "OptionalStringService",
			ValueObjects: []spec.ValueObject{{
				Name:        "OptionalLabel",
				Type:        "string",
				Validations: spec.ValidationRules{MaxLength: intPtr(80)},
			}},
			Entities: []spec.Entity{{
				Name:   "OptionalRecord",
				Fields: []spec.Field{{Name: "Id", Type: "Guid"}, {Name: "Label", Type: "OptionalLabel"}},
			}},
		}},
	}
}

func boolPtr(value bool) *bool            { return &value }
func intPtr(value int) *int               { return &value }
func stringPtr(value string) *string      { return &value }
func numberPtr(value string) *json.Number { number := json.Number(value); return &number }
