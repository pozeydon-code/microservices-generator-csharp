package generator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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

func TestGenerateIdOnlyEntityOmitsUnsafeSchemaTransition(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	cfg := spec.Config{
		Solution: spec.Solution{Name: "IdentityPlatform", Description: "Id-only entity regression."},
		Services: []spec.Service{{
			Name: "IdentityService",
			Entities: []spec.Entity{{
				Name:   "Identity",
				Fields: []spec.Field{{Name: "Id", Type: "Guid"}},
			}},
		}},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected Id-only config to remain valid: %v", err)
	}

	files, err := gen.Generate(cfg)
	if err != nil {
		t.Fatalf("generate Id-only config: %v", err)
	}
	infrastructureTests := string(generatedContent(t, files, "tests/IdentityService/IdentityService.Infrastructure.Tests/IdentityServiceInfrastructureTests.cs"))
	assertContains(t, infrastructureTests, "Assert.Equal(ReadinessStatus.Ready, healthy.Status);")
	assertNotContains(t, infrastructureTests, "DROP COLUMN")
	assertNotContains(t, infrastructureTests, "<no value>")
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
	assertContains(t, contentByPath["Directory.Packages.props"], "Microsoft.Data.SqlClient\" Version=\"6.1.1")
	assertContains(t, contentByPath["Directory.Packages.props"], "System.Security.Cryptography.Xml\" Version=\"8.0.4")
	assertContains(t, contentByPath["Directory.Packages.props"], "Package versions are generated from one target-framework policy table.")
	assertContains(t, contentByPath["Directory.Packages.props"], "Allows central PackageVersion entries to override vulnerable transitives; NuGet rejects downgrades with NU1109.")
	assertContains(t, contentByPath["Directory.Packages.props"], "Pinned for NuGet audit safety when EF/Core build transitives request vulnerable XML versions.")
	assertContains(t, contentByPath["Directory.Packages.props"], "CentralPackageTransitivePinningEnabled>true")
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

func TestGenerateDirectoryPackagesPropsUsesDependencyPolicy(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	tests := []struct {
		name            string
		targetFramework string
		aspNetCore      string
		aspNetCoreTest  string
		entityFramework string
		sqlClient       string
		cryptographyXML string
	}{
		{name: "net8", targetFramework: "net8.0", aspNetCore: "8.0.28", aspNetCoreTest: "8.0.28", entityFramework: "8.0.28", sqlClient: "6.1.1", cryptographyXML: "8.0.4"},
		{name: "net9", targetFramework: "net9.0", aspNetCore: "9.0.7", aspNetCoreTest: "9.0.7", entityFramework: "9.0.7", sqlClient: "6.1.1", cryptographyXML: "9.0.18"},
		{name: "net10", targetFramework: "net10.0", aspNetCore: "10.0.0", aspNetCoreTest: "10.0.0", entityFramework: "10.0.0", sqlClient: "6.1.1", cryptographyXML: "10.0.10"},
		{name: "net11", targetFramework: "net11.0", aspNetCore: "11.0.0", aspNetCoreTest: "11.0.0", entityFramework: "11.0.0", sqlClient: "6.1.1", cryptographyXML: "10.0.10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testConfig()
			cfg.Generation.TargetFramework = tt.targetFramework

			files, err := gen.Generate(cfg)
			if err != nil {
				t.Fatalf("generate: %v", err)
			}
			packages := string(generatedContent(t, files, "Directory.Packages.props"))

			assertContains(t, packages, "<ManagePackageVersionsCentrally>true</ManagePackageVersionsCentrally>")
			assertContains(t, packages, "<CentralPackageTransitivePinningEnabled>true</CentralPackageTransitivePinningEnabled>")
			assertContains(t, packages, `Microsoft.AspNetCore.Authentication.JwtBearer" Version="`+tt.aspNetCore+`"`)
			assertContains(t, packages, `Microsoft.AspNetCore.Mvc.Testing" Version="`+tt.aspNetCoreTest+`"`)
			assertContains(t, packages, `Microsoft.EntityFrameworkCore.Design" Version="`+tt.entityFramework+`"`)
			assertContains(t, packages, `Microsoft.EntityFrameworkCore.SqlServer" Version="`+tt.entityFramework+`"`)
			assertContains(t, packages, `Microsoft.Data.SqlClient" Version="`+tt.sqlClient+`"`)
			assertContains(t, packages, `System.Security.Cryptography.Xml" Version="`+tt.cryptographyXML+`"`)
		})
	}
}

func TestGenerateUsesSelectedTargetFramework(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	cfg := testConfig()
	cfg.SchemaVersion = spec.ConfigSchemaVersion
	cfg.Generation.TargetFramework = "net9.0"

	files, err := gen.Generate(cfg)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	props := string(generatedContent(t, files, "Directory.Build.props"))
	packages := string(generatedContent(t, files, "Directory.Packages.props"))
	metadata := string(generatedContent(t, files, "microgen.json"))
	readme := string(generatedContent(t, files, "README.md"))

	assertContains(t, props, "<TargetFramework>net9.0</TargetFramework>")
	assertContains(t, packages, `Microsoft.AspNetCore.Mvc.Testing" Version="9.0.7`)
	assertContains(t, packages, `Microsoft.EntityFrameworkCore.SqlServer" Version="9.0.7`)
	assertContains(t, packages, `System.Security.Cryptography.Xml" Version="9.0.18`)
	assertContains(t, metadata, `"targetFramework": "net9.0"`)
	assertNotContains(t, props, "net8.0")
	assertNotContains(t, packages, "Version=\"8.0.28")
	assertNotContains(t, metadata, "net8.0")
	assertContains(t, readme, "Minimal generated .NET 8 microservice workspace for product management.")
}

func TestGenerateDirectoryBuildPropsOwnsQualityDefaults(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	files, err := gen.Generate(testConfig())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	props := string(generatedContent(t, files, "Directory.Build.props"))

	for _, expected := range []string{
		"<Nullable>enable</Nullable>",
		"<ImplicitUsings>enable</ImplicitUsings>",
		"<AnalysisLevel>latest-recommended</AnalysisLevel>",
		"<AnalysisMode>Recommended</AnalysisMode>",
		"<EnforceCodeStyleInBuild>true</EnforceCodeStyleInBuild>",
		"<TreatWarningsAsErrors>true</TreatWarningsAsErrors>",
	} {
		assertContains(t, props, expected)
	}
}

func TestGenerateDefaultsSolutionFileFormatFromTargetFramework(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	tests := []struct {
		name               string
		targetFramework    string
		expectedSolution   string
		unexpectedSolution string
	}{
		{name: "below net10", targetFramework: "net7.0", expectedSolution: "CommercePlatform.sln", unexpectedSolution: "CommercePlatform.slnx"},
		{name: "net10", targetFramework: "net10.0", expectedSolution: "CommercePlatform.slnx", unexpectedSolution: "CommercePlatform.sln"},
		{name: "future", targetFramework: "net11.0", expectedSolution: "CommercePlatform.slnx", unexpectedSolution: "CommercePlatform.sln"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testConfig()
			cfg.Generation.TargetFramework = tt.targetFramework

			files, err := gen.Generate(cfg)
			if err != nil {
				t.Fatalf("generate: %v", err)
			}
			contentByPath := map[string]string{}
			for _, file := range files {
				contentByPath[file.Path] = string(file.Content)
			}
			if _, ok := contentByPath[tt.expectedSolution]; !ok {
				t.Fatalf("expected %s to be generated", tt.expectedSolution)
			}
			if _, ok := contentByPath[tt.unexpectedSolution]; ok {
				t.Fatalf("did not expect %s to be generated", tt.unexpectedSolution)
			}
			readme := contentByPath["README.md"]
			assertContains(t, readme, "dotnet build ./"+tt.expectedSolution)
			assertContains(t, readme, "dotnet test ./"+tt.expectedSolution)
		})
	}
}

func TestGenerateSlnxReferencesAllProjectsDeterministically(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	cfg := testConfig()
	cfg.Generation.TargetFramework = "net10.0"

	files, err := gen.Generate(cfg)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	slnx := string(generatedContent(t, files, "CommercePlatform.slnx"))
	packages := string(generatedContent(t, files, "Directory.Packages.props"))

	assertContains(t, slnx, "<Solution>\n")
	assertContains(t, slnx, `  <Project Path="src/ProductService/ProductService.Api/ProductService.Api.csproj" />`)
	assertContains(t, slnx, `  <Project Path="tests/ProductService/ProductService.Infrastructure.Tests/ProductService.Infrastructure.Tests.csproj" />`)
	assertNotContains(t, slnx, "ProjectConfigurationPlatforms")
	assertContains(t, packages, `System.Security.Cryptography.Xml" Version="10.0.10`)
}

func TestGenerateExampleSolutionFilesAreRuntimeParseable(t *testing.T) {
	tests := []struct {
		name               string
		targetFramework    string
		solutionFormat     string
		expectedSolution   string
		unexpectedSolution string
		validateWithDotnet bool
	}{
		{name: "net8 default", targetFramework: "net8.0", expectedSolution: "CommercePlatform.sln", unexpectedSolution: "CommercePlatform.slnx"},
		{name: "net8 explicit sln", targetFramework: "net8.0", solutionFormat: "sln", expectedSolution: "CommercePlatform.sln", unexpectedSolution: "CommercePlatform.slnx"},
		{name: "net10 default", targetFramework: "net10.0", expectedSolution: "CommercePlatform.slnx", unexpectedSolution: "CommercePlatform.sln", validateWithDotnet: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspace := generateExampleWorkspace(t, tt.targetFramework, tt.solutionFormat)

			assertPathExists(t, filepath.Join(workspace.dir, tt.expectedSolution))
			assertPathMissing(t, filepath.Join(workspace.dir, tt.unexpectedSolution))

			if !tt.validateWithDotnet {
				return
			}
			dotnet := locateDotnet(t)
			runDotnetRuntimeCommand(t, dotnet, workspace, "sln", workspace.solutionPath, "list")
		})
	}
}

func TestGenerateNet10DefaultSlnxRuntimeValidation(t *testing.T) {
	workspace := generateExampleWorkspace(t, "net10.0", "")
	dotnet := locateDotnet(t)

	runDotnetRuntimeCommand(t, dotnet, workspace, "restore", workspace.solutionPath)
	runDotnetRuntimeCommand(t, dotnet, workspace, "build", "--no-restore", workspace.solutionPath)
	runDotnetRuntimeCommand(t, dotnet, workspace, "test", "--no-build", workspace.solutionPath)
}

func TestGenerateUsesPluralizedEntityNamesForFeatureAndRoute(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	cfg := testConfig()
	cfg.Services[0].Entities[0].Name = "Category"

	files, err := gen.Generate(cfg)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	endpoints := string(generatedContent(t, files, "src/ProductService/ProductService.Api/Features/Categories/CategoryEndpoints.cs"))

	assertContains(t, endpoints, "MapGroup(\"/categories\")")
	assertNotContains(t, endpoints, "Categorys")
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

func TestGenerateRecordRepositoryDiagnosticsAvoidsInstanceAnalyzerWarningWithoutValueObjects(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}

	tests := []struct {
		name                 string
		cfg                  spec.Config
		expectedSignature    string
		unexpectedSignature  string
		expectedDbContextUse string
	}{
		{
			name:                "zero value objects",
			cfg:                 recordCollisionConfig(false),
			expectedSignature:   "private static Task<(string Field, IReadOnlyList<Guid> RecordIds)> FindReconstitutionDiagnosticsAsync",
			unexpectedSignature: "private async Task<(string Field, IReadOnlyList<Guid> RecordIds)> FindReconstitutionDiagnosticsAsync",
		},
		{
			name:                 "value object backed",
			cfg:                  recordCollisionConfig(true),
			expectedSignature:    "private async Task<(string Field, IReadOnlyList<Guid> RecordIds)> FindReconstitutionDiagnosticsAsync",
			unexpectedSignature:  "private static Task<(string Field, IReadOnlyList<Guid> RecordIds)> FindReconstitutionDiagnosticsAsync",
			expectedDbContextUse: "await dbContext.Database.SqlQueryRaw<Guid>(sql, id.Value).ToListAsync(cancellationToken)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := gen.Generate(tt.cfg)
			if err != nil {
				t.Fatalf("generate: %v", err)
			}
			repository := string(generatedContent(t, files, "src/RecordService/RecordService.Infrastructure/Persistence/Features/Records/RecordRepository.cs"))

			assertContains(t, repository, tt.expectedSignature)
			assertNotContains(t, repository, tt.unexpectedSignature)
			if tt.expectedDbContextUse != "" {
				assertContains(t, repository, tt.expectedDbContextUse)
			}
		})
	}
}

func TestGenerateRecordCollisionZeroValueObjectRuntimeBuild(t *testing.T) {
	cfg := recordCollisionConfig(false)
	cfg.Generation.TargetFramework = "net10.0"
	workspace := generateWorkspace(t, cfg)
	dotnet := locateDotnet(t)

	runDotnetRuntimeCommand(t, dotnet, workspace, "restore", workspace.solutionPath)
	runDotnetRuntimeCommand(t, dotnet, workspace, "build", "--no-restore", workspace.solutionPath)
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

func TestGenerateDerivesPatternInvalidSampleAfterEarlierStringRules(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	cfg := testConfig()
	cfg.Services[0].ValueObjects = []spec.ValueObject{{
		Name: "ShortPatternName",
		Type: "string",
		Validations: spec.ValidationRules{
			Required:       boolPtr(true),
			MinLength:      intPtr(3),
			Pattern:        stringPtr("^[A-Za-z]+$"),
			ValidExample:   stringPtr("Valid"),
			InvalidExample: stringPtr("!!"),
		},
	}}
	cfg.Services[0].Entities[0].Fields = []spec.Field{{Name: "Id", Type: "Guid"}, {Name: "Name", Type: "ShortPatternName"}}

	files, err := gen.Generate(cfg)
	if err != nil {
		t.Fatalf("generate pattern invalid sample config: %v", err)
	}
	domainTests := string(generatedContent(t, files, "tests/ProductService/ProductService.Domain.Tests/ProductServiceDomainTests.cs"))

	assertContains(t, domainTests, `Assert.Equal("ShortPatternName.Pattern", Assert.Single(ShortPatternName.Create("!!!").Errors).Code);`)
	assertContains(t, domainTests, `var result = ShortPatternName.Create("!!!", "Field");`)
	assertNotContains(t, domainTests, `Assert.Single(ShortPatternName.Create("!!").Errors).Code`)
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

func exampleConfig(t *testing.T) spec.Config {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("..", "..", "examples", "product-service.json"))
	if err != nil {
		t.Fatalf("read example config: %v", err)
	}
	var cfg spec.Config
	if err := json.Unmarshal(content, &cfg); err != nil {
		t.Fatalf("decode example config: %v", err)
	}
	return cfg
}

type generatedWorkspace struct {
	dir          string
	solutionName string
	solutionPath string
}

func generateExampleWorkspace(t *testing.T, targetFramework, solutionFormat string) generatedWorkspace {
	t.Helper()
	cfg := exampleConfig(t)
	cfg.Generation.TargetFramework = targetFramework
	cfg.Generation.SolutionFormat = solutionFormat
	return generateWorkspace(t, cfg)
}

func generateWorkspace(t *testing.T, cfg spec.Config) generatedWorkspace {
	t.Helper()
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	files, err := gen.Generate(cfg)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	outputDir := t.TempDir()
	writeGeneratedFiles(t, outputDir, files)

	solutionName := cfg.Solution.Name + "." + cfg.SolutionFormat()
	return generatedWorkspace{
		dir:          outputDir,
		solutionName: solutionName,
		solutionPath: filepath.Join(outputDir, solutionName),
	}
}

func locateDotnet(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping dotnet runtime validation in short mode")
	}
	dotnet, err := exec.LookPath("dotnet")
	if err != nil {
		t.Skipf("dotnet not installed: %v", err)
	}
	return dotnet
}

func runDotnetRuntimeCommand(t *testing.T, dotnet string, workspace generatedWorkspace, args ...string) {
	t.Helper()
	cmd := exec.Command(dotnet, args...)
	cmd.Dir = workspace.dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("dotnet %s failed: %v\nworking directory: %s\nsolution: %s\noutput:\n%s", strings.Join(args, " "), err, workspace.dir, workspace.solutionName, output)
	}
}

func writeGeneratedFiles(t *testing.T, outputDir string, files []GeneratedFile) {
	t.Helper()
	for _, file := range files {
		path := filepath.Join(outputDir, file.Path)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("create generated parent: %v", err)
		}
		if err := os.WriteFile(path, file.Content, 0644); err != nil {
			t.Fatalf("write generated file: %v", err)
		}
	}
}

func assertPathExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
}

func assertPathMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be missing, got %v", path, err)
	}
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
