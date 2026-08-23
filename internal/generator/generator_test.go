package generator

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/pozeydon-code/microservices-generator-csharp/internal/spec"
)

var updateGolden = flag.Bool("update", false, "update generator golden files")

func TestGeneratedWebApiTestsDoNotContainUnusedJsonPropertyFields(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}

	files, err := gen.Generate(testConfig())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	content := string(generatedContent(
		t,
		files,
		"ProductService/tests/ProductService.WebApi.Tests/Features/Products/ProductControllerTests.cs",
	))

	assertNotContains(t, content, "ExpectedProductJsonProperties")
	assertNotContains(t, content, "AllowedValidationProblemProperties")
}

func TestGeneratedReadmeUsesSimpleLocalMigrationCommands(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}

	files, err := gen.Generate(testConfig())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	content := string(generatedContent(t, files, "README.md"))
	assertContains(t, content, "dotnet ef migrations add InitialCreate")
	assertContains(t, content, "dotnet ef database update --project ./ProductService/src/ProductService.Infrastructure --startup-project ./ProductService/src/ProductService.Infrastructure")
	assertContains(t, content, "For shared, staging, or production environments, use your team's deployment process.")
	assertNotContains(t, content, "dotnet ef migrations script --idempotent")
	assertNotContains(t, content, "sqlcmd")
	assertNotContains(t, content, "BACKUP DATABASE")
}

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
		{path: "Directory.Build.props", goldenName: "Directory.Build.props"},
		{path: "Directory.Packages.props", goldenName: "Directory.Packages.props"},
		{path: "ProductService/Directory.Build.props", goldenName: "Directory.Build.props"},
		{path: "ProductService/Directory.Packages.props", goldenName: "Directory.Packages.props"},
		{path: "ProductService/ProductService.sln", goldenName: "ProductService.sln"},
		{path: "ProductService/src/ProductService.Application/ApplicationAssemblyReference.cs", goldenName: "ApplicationAssemblyReference.cs"},
		{path: "ProductService/src/ProductService.Application/Common/PaginationPolicy.cs", goldenName: "PaginationPolicy.cs"},
		{path: "ProductService/src/ProductService.Application/Common/Readiness.cs", goldenName: "Readiness.cs"},
		{path: "ProductService/src/ProductService.Application/Common/Results.cs", goldenName: "Results.cs"},
		{path: "ProductService/src/ProductService.Application/Common/UnitOfWork.cs", goldenName: "UnitOfWork.cs"},
		{path: "ProductService/src/ProductService.Application/Common/ValidationBehavior.cs", goldenName: "ValidationBehavior.cs"},
		{path: "ProductService/src/ProductService.Application/DependencyInjection.cs", goldenName: "Application.DependencyInjection.cs"},
		{path: "ProductService/src/ProductService.Application/ProductService.Application.csproj", goldenName: "ProductService.Application.csproj"},
		{path: "ProductService/src/ProductService.Application/Products/Commands/Create/CreateProductCommand.cs", goldenName: "CreateProductCommand.cs"},
		{path: "ProductService/src/ProductService.Application/Products/Commands/Create/CreateProductCommandHandler.cs", goldenName: "CreateProductCommandHandler.cs"},
		{path: "ProductService/src/ProductService.Application/Products/Commands/Create/CreateProductCommandValidator.cs", goldenName: "CreateProductCommandValidator.cs"},
		{path: "ProductService/src/ProductService.Application/Products/Commands/Delete/DeleteProductCommand.cs", goldenName: "DeleteProductCommand.cs"},
		{path: "ProductService/src/ProductService.Application/Products/Commands/Delete/DeleteProductCommandHandler.cs", goldenName: "DeleteProductCommandHandler.cs"},
		{path: "ProductService/src/ProductService.Application/Products/Commands/Delete/DeleteProductCommandValidator.cs", goldenName: "DeleteProductCommandValidator.cs"},
		{path: "ProductService/src/ProductService.Application/Products/Commands/Update/UpdateProductCommand.cs", goldenName: "UpdateProductCommand.cs"},
		{path: "ProductService/src/ProductService.Application/Products/Commands/Update/UpdateProductCommandHandler.cs", goldenName: "UpdateProductCommandHandler.cs"},
		{path: "ProductService/src/ProductService.Application/Products/Commands/Update/UpdateProductCommandValidator.cs", goldenName: "UpdateProductCommandValidator.cs"},
		{path: "ProductService/src/ProductService.Application/Products/Dtos/CreateProductRequest.cs", goldenName: "CreateProductRequest.cs"},
		{path: "ProductService/src/ProductService.Application/Products/Dtos/ProductDto.cs", goldenName: "ProductDto.cs"},
		{path: "ProductService/src/ProductService.Application/Products/Dtos/UpdateProductRequest.cs", goldenName: "UpdateProductRequest.cs"},
		{path: "ProductService/src/ProductService.Application/Products/Interfaces/IProductRepository.cs", goldenName: "IProductRepository.cs"},
		{path: "ProductService/src/ProductService.Application/Products/Queries/GetById/GetProductByIdQuery.cs", goldenName: "GetProductByIdQuery.cs"},
		{path: "ProductService/src/ProductService.Application/Products/Queries/GetById/GetProductByIdQueryHandler.cs", goldenName: "GetProductByIdQueryHandler.cs"},
		{path: "ProductService/src/ProductService.Application/Products/Queries/List/ListProductQuery.cs", goldenName: "ListProductQuery.cs"},
		{path: "ProductService/src/ProductService.Application/Products/Queries/List/ListProductQueryHandler.cs", goldenName: "ListProductQueryHandler.cs"},
		{path: "ProductService/src/ProductService.Application/Products/Queries/List/ListProductQueryValidator.cs", goldenName: "ListProductQueryValidator.cs"},
		{path: "ProductService/src/ProductService.Domain/Entities/Product.cs", goldenName: "Product.cs"},
		{path: "ProductService/src/ProductService.Domain/Primitives/DomainReconstitutionException.cs", goldenName: "DomainReconstitutionException.cs"},
		{path: "ProductService/src/ProductService.Domain/Primitives/DomainResult.cs", goldenName: "DomainResult.cs"},
		{path: "ProductService/src/ProductService.Domain/ProductService.Domain.csproj", goldenName: "ProductService.Domain.csproj"},
		{path: "ProductService/src/ProductService.Domain/ValueObjects/ProductName.cs", goldenName: "ProductName.cs"},
		{path: "ProductService/src/ProductService.Domain/ValueObjects/ProductPrice.cs", goldenName: "ProductPrice.cs"},
		{path: "ProductService/src/ProductService.Infrastructure/DependencyInjection.cs", goldenName: "Infrastructure.DependencyInjection.cs"},
		{path: "ProductService/src/ProductService.Infrastructure/Health/SqlReadinessProbe.cs", goldenName: "SqlReadinessProbe.cs"},
		{path: "ProductService/src/ProductService.Infrastructure/Persistence/Configurations/ProductConfiguration.cs", goldenName: "ProductConfiguration.cs"},
		{path: "ProductService/src/ProductService.Infrastructure/Persistence/Features/Products/ProductRepository.cs", goldenName: "ProductRepository.cs"},
		{path: "ProductService/src/ProductService.Infrastructure/Persistence/ProductServiceDbContext.cs", goldenName: "ProductServiceDbContext.cs"},
		{path: "ProductService/src/ProductService.Infrastructure/Persistence/ProductServiceDbContextFactory.cs", goldenName: "ProductServiceDbContextFactory.cs"},
		{path: "ProductService/src/ProductService.Infrastructure/Persistence/UnitOfWork.cs", goldenName: "Infrastructure.UnitOfWork.cs"},
		{path: "ProductService/src/ProductService.Infrastructure/ProductService.Infrastructure.csproj", goldenName: "ProductService.Infrastructure.csproj"},
		{path: "ProductService/src/ProductService.WebApi/Common/Errors/ApiProblemDetailsFactory.cs", goldenName: "ApiProblemDetailsFactory.cs"},
		{path: "ProductService/src/ProductService.WebApi/Common/Errors/HttpContextItemKeys.cs", goldenName: "HttpContextItemKeys.cs"},
		{path: "ProductService/src/ProductService.WebApi/Controllers/ApiController.cs", goldenName: "ApiController.cs"},
		{path: "ProductService/src/ProductService.WebApi/Controllers/Products/ProductController.cs", goldenName: "ProductController.cs"},
		{path: "ProductService/src/ProductService.WebApi/DependencyInjection.cs", goldenName: "WebApi.DependencyInjection.cs"},
		{path: "ProductService/src/ProductService.WebApi/Health/HealthController.cs", goldenName: "HealthController.cs"},
		{path: "ProductService/src/ProductService.WebApi/ProductService.WebApi.csproj", goldenName: "ProductService.WebApi.csproj"},
		{path: "ProductService/src/ProductService.WebApi/Program.cs", goldenName: "Program.cs"},
		{path: "ProductService/src/ProductService.WebApi/appsettings.Development.json", goldenName: "appsettings.Development.json"},
		{path: "ProductService/src/ProductService.WebApi/appsettings.json", goldenName: "appsettings.json"},
		{path: "ProductService/tests/ProductService.Application.Tests/Features/Products/ProductApplicationTests.cs", goldenName: "ProductApplicationTests.cs"},
		{path: "ProductService/tests/ProductService.Application.Tests/ProductService.Application.Tests.csproj", goldenName: "ProductService.Application.Tests.csproj"},
		{path: "ProductService/tests/ProductService.Architecture.Tests/ProductService.Architecture.Tests.csproj", goldenName: "ProductService.Architecture.Tests.csproj"},
		{path: "ProductService/tests/ProductService.Architecture.Tests/ProductServiceArchitectureTests.cs", goldenName: "ProductServiceArchitectureTests.cs"},
		{path: "ProductService/tests/ProductService.Domain.Tests/ProductService.Domain.Tests.csproj", goldenName: "ProductService.Domain.Tests.csproj"},
		{path: "ProductService/tests/ProductService.Domain.Tests/ProductServiceDomainTests.cs", goldenName: "ProductServiceDomainTests.cs"},
		{path: "ProductService/tests/ProductService.Infrastructure.Tests/ProductService.Infrastructure.Tests.csproj", goldenName: "ProductService.Infrastructure.Tests.csproj"},
		{path: "ProductService/tests/ProductService.Infrastructure.Tests/ProductServiceInfrastructureTests.cs", goldenName: "ProductServiceInfrastructureTests.cs"},
		{path: "ProductService/tests/ProductService.WebApi.Tests/AuthenticationTests.cs", goldenName: "AuthenticationTests.cs"},
		{path: "ProductService/tests/ProductService.WebApi.Tests/Features/Products/ProductControllerTests.cs", goldenName: "ProductControllerTests.cs"},
		{path: "ProductService/tests/ProductService.WebApi.Tests/HealthControllerTests.cs", goldenName: "HealthControllerTests.cs"},
		{path: "ProductService/tests/ProductService.WebApi.Tests/ProductService.WebApi.Tests.csproj", goldenName: "ProductService.WebApi.Tests.csproj"},
		{path: "ProductService/tests/ProductService.WebApi.Tests/TestJwtTokens.cs", goldenName: "TestJwtTokens.cs"},
		{path: "ProductService/tests/ProductService.WebApi.Tests/TestWebApiFactory.cs", goldenName: "TestWebApiFactory.cs"},
		{path: "README.md", goldenName: "README.md"},
		{path: "microgen.json", goldenName: "microgen.json"},
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

func TestGenerateGatewayProducesDeterministicGoldenOutput(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	cfg := testConfig()
	cfg.Solution.Name = "ShopPlatform"
	cfg.Generation.Gateway.Enabled = true

	first, err := gen.Generate(cfg)
	if err != nil {
		t.Fatalf("generate first gateway: %v", err)
	}
	second, err := gen.Generate(cfg)
	if err != nil {
		t.Fatalf("generate second gateway: %v", err)
	}

	if len(first) != len(second) {
		t.Fatalf("expected same gateway file count, got %d and %d", len(first), len(second))
	}
	for index := range first {
		if first[index].Path != second[index].Path || !bytes.Equal(first[index].Content, second[index].Content) {
			t.Fatalf("gateway generation is not deterministic at index %d", index)
		}
	}

	expectedFiles := []struct {
		path       string
		goldenName string
	}{
		{path: "Directory.Build.props", goldenName: filepath.Join("gateway-enabled", "Directory.Build.props")},
		{path: "Directory.Packages.props", goldenName: filepath.Join("gateway-enabled", "Directory.Packages.props")},
		{path: "README.md", goldenName: filepath.Join("gateway-enabled", "README.md")},
		{path: "ShopPlatform.sln", goldenName: filepath.Join("gateway-enabled", "ShopPlatform.sln")},
		{path: "microgen.json", goldenName: filepath.Join("gateway-enabled", "microgen.json")},
		{path: "ProductService/Directory.Build.props", goldenName: filepath.Join("gateway-enabled", "Directory.Build.props")},
		{path: "ProductService/Directory.Packages.props", goldenName: filepath.Join("gateway-enabled", "Directory.Packages.props")},
		{path: "Gateway/Directory.Build.props", goldenName: filepath.Join("gateway-enabled", "Directory.Build.props")},
		{path: "Gateway/Directory.Packages.props", goldenName: filepath.Join("gateway-enabled", "Directory.Packages.props")},
		{path: "Gateway/ShopPlatform.Gateway.csproj", goldenName: filepath.Join("gateway-enabled", "ShopPlatform.Gateway.csproj")},
		{path: "Gateway/Program.cs", goldenName: filepath.Join("gateway-enabled", "Gateway.Program.cs")},
		{path: "Gateway/appsettings.json", goldenName: filepath.Join("gateway-enabled", "Gateway.appsettings.json")},
	}

	for _, file := range expectedFiles {
		assertGoldenFile(t, first, file.path, file.goldenName)
	}
	for _, file := range first {
		if strings.HasPrefix(file.Path, "src/ShopPlatform.Gateway/") {
			t.Fatalf("expected no old gateway path, got %s", file.Path)
		}
	}
}

func TestGenerateEmitsPortablePropsForEveryServiceAndKeepsRootProps(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	cfg := testConfig()
	cfg.Services = append(cfg.Services, spec.Service{
		Name: "OrderingService",
		Entities: []spec.Entity{{
			Name:   "Order",
			Fields: []spec.Field{{Name: "Id", Type: "Guid"}, {Name: "Number", Type: "string"}},
		}},
	})

	files, err := gen.Generate(cfg)
	if err != nil {
		t.Fatalf("generate portable service props: %v", err)
	}
	rootBuildProps := generatedContent(t, files, "Directory.Build.props")
	rootPackagesProps := generatedContent(t, files, "Directory.Packages.props")

	for _, serviceName := range []string{"OrderingService", "ProductService"} {
		serviceBuildProps := generatedContent(t, files, join(serviceName, "Directory.Build.props"))
		if !bytes.Equal(serviceBuildProps, rootBuildProps) {
			t.Fatalf("expected %s Directory.Build.props to match root props", serviceName)
		}
		assertContains(t, string(serviceBuildProps), "<TargetFramework>net8.0</TargetFramework>")
		servicePackagesProps := generatedContent(t, files, join(serviceName, "Directory.Packages.props"))
		if !bytes.Equal(servicePackagesProps, rootPackagesProps) {
			t.Fatalf("expected %s Directory.Packages.props to match root props", serviceName)
		}
		assertContains(t, string(servicePackagesProps), "Microsoft.EntityFrameworkCore.SqlServer\" Version=\"8.0.28")
	}
}

func TestGenerateEmitsPortablePropsForGatewayWhenEnabled(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	cfg := testConfig()
	cfg.Solution.Name = "ShopPlatform"
	cfg.Generation.Gateway.Enabled = true

	files, err := gen.Generate(cfg)
	if err != nil {
		t.Fatalf("generate gateway props: %v", err)
	}
	rootBuildProps := generatedContent(t, files, "Directory.Build.props")
	rootPackagesProps := generatedContent(t, files, "Directory.Packages.props")
	gatewayBuildProps := generatedContent(t, files, "Gateway/Directory.Build.props")
	gatewayPackagesProps := generatedContent(t, files, "Gateway/Directory.Packages.props")

	if !bytes.Equal(gatewayBuildProps, rootBuildProps) {
		t.Fatal("expected gateway Directory.Build.props to match root props")
	}
	if !bytes.Equal(gatewayPackagesProps, rootPackagesProps) {
		t.Fatal("expected gateway Directory.Packages.props to match root props")
	}
}

func TestGenerateOmitsGatewayFolderAndPropsWhenGatewayDisabledOrOmitted(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}

	for _, tt := range []struct {
		name string
		cfg  spec.Config
	}{
		{name: "omitted", cfg: testConfig()},
		{name: "disabled", cfg: func() spec.Config {
			cfg := testConfig()
			cfg.Generation.Gateway.Enabled = false
			return cfg
		}()},
	} {
		t.Run(tt.name, func(t *testing.T) {
			files, err := gen.Generate(tt.cfg)
			if err != nil {
				t.Fatalf("generate disabled gateway props: %v", err)
			}
			for _, file := range files {
				if strings.HasPrefix(file.Path, "Gateway/") || strings.HasPrefix(file.Path, "src/CommercePlatform.Gateway/") {
					t.Fatalf("expected no gateway output when disabled, got %s", file.Path)
				}
			}
		})
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
	infrastructureTests := string(generatedContent(t, files, "IdentityService/tests/IdentityService.Infrastructure.Tests/IdentityServiceInfrastructureTests.cs"))
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

	assertContains(t, contentByPath["ProductService/src/ProductService.Application/ProductService.Application.csproj"], "ProductService.Domain.csproj")
	assertNotContains(t, contentByPath["ProductService/src/ProductService.Application/ProductService.Application.csproj"], "ProductService.Infrastructure")
	assertNotContains(t, contentByPath["ProductService/src/ProductService.Application/ProductService.Application.csproj"], "ProductService.WebApi")
	assertContains(t, contentByPath["ProductService/src/ProductService.Infrastructure/ProductService.Infrastructure.csproj"], "ProductService.Application.csproj")
	assertContains(t, contentByPath["ProductService/src/ProductService.Domain/ValueObjects/ProductName.cs"], "DomainResult<ProductName>")
	assertContains(t, contentByPath["ProductService/src/ProductService.Application/Common/UnitOfWork.cs"], "Task<int> SaveChangesAsync(CancellationToken cancellationToken)")
	assertContains(t, contentByPath["ProductService/src/ProductService.Application/Common/Results.cs"], "public enum MutationPreparationStatus { Prepared, InvalidToken }")
	assertNotContains(t, contentByPath["ProductService/src/ProductService.Application/Common/Results.cs"], "SaveResultStatus")
	assertContains(t, contentByPath["ProductService/src/ProductService.Infrastructure/Persistence/UnitOfWork.cs"], "catch (DbUpdateConcurrencyException)")
	assertContains(t, contentByPath["ProductService/src/ProductService.Infrastructure/Persistence/ProductServiceDbContext.cs"], "ApplyConfigurationsFromAssembly(typeof(ProductServiceDbContext).Assembly)")
	assertNotContains(t, contentByPath["ProductService/src/ProductService.Infrastructure/Persistence/ProductServiceDbContext.cs"], "modelBuilder.Entity<Product>")
	assertContains(t, contentByPath["ProductService/src/ProductService.Domain/Entities/Product.cs"], "public byte[] RowVersion { get; private set; } = [];")
	assertContains(t, contentByPath["ProductService/src/ProductService.Infrastructure/Persistence/Configurations/ProductConfiguration.cs"], "IEntityTypeConfiguration<Product>")
	assertContains(t, contentByPath["ProductService/src/ProductService.Infrastructure/Persistence/Configurations/ProductConfiguration.cs"], "builder.Property(item => item.RowVersion).IsRowVersion()")
	assertNotContains(t, contentByPath["ProductService/src/ProductService.Infrastructure/Persistence/Configurations/ProductConfiguration.cs"], "Property<byte[]>(\"RowVersion\")")
	assertContains(t, contentByPath["ProductService/src/ProductService.Infrastructure/Persistence/Configurations/ProductConfiguration.cs"], "HasConversion(value => value.Value, value => ProductService.Domain.ValueObjects.ProductName.Rehydrate(value))")
	assertContains(t, contentByPath["Directory.Packages.props"], "Microsoft.EntityFrameworkCore.SqlServer\" Version=\"8.0.28")
	assertContains(t, contentByPath["Directory.Packages.props"], "Microsoft.EntityFrameworkCore.Tools\" Version=\"8.0.28")
	assertContains(t, contentByPath["Directory.Packages.props"], "Microsoft.AspNetCore.Mvc.Testing\" Version=\"8.0.28")
	assertContains(t, contentByPath["Directory.Packages.props"], "Microsoft.Data.SqlClient\" Version=\"6.1.1")
	assertContains(t, contentByPath["Directory.Packages.props"], "System.Security.Cryptography.Xml\" Version=\"8.0.4")
	assertContains(t, contentByPath["Directory.Packages.props"], "Package versions are generated from one target-framework policy table.")
	assertContains(t, contentByPath["Directory.Packages.props"], "Allows central PackageVersion entries to override vulnerable transitives; NuGet rejects downgrades with NU1109.")
	assertContains(t, contentByPath["Directory.Packages.props"], "Pinned for NuGet audit safety when EF/Core build transitives request vulnerable XML versions.")
	assertContains(t, contentByPath["Directory.Packages.props"], "CentralPackageTransitivePinningEnabled>true")
	assertContains(t, contentByPath["ProductService/src/ProductService.Infrastructure/ProductService.Infrastructure.csproj"], "Microsoft.EntityFrameworkCore.SqlServer")
	assertContains(t, contentByPath["ProductService/src/ProductService.Infrastructure/ProductService.Infrastructure.csproj"], "Microsoft.EntityFrameworkCore.Tools")
	dbContextFactory := contentByPath["ProductService/src/ProductService.Infrastructure/Persistence/ProductServiceDbContextFactory.cs"]
	assertContains(t, dbContextFactory, "IDesignTimeDbContextFactory<ProductServiceDbContext>")
	assertContains(t, dbContextFactory, "Environment.GetEnvironmentVariable(\"ConnectionStrings__DefaultConnection\")")
	assertContains(t, dbContextFactory, "UseSqlServer(")
	webAPIProject := contentByPath["ProductService/src/ProductService.WebApi/ProductService.WebApi.csproj"]
	assertContains(t, webAPIProject, "ProductService.Infrastructure.csproj")
	assertContains(t, webAPIProject, "ProductService.Application.csproj")
	assertNotContains(t, webAPIProject, "Microsoft.AspNetCore.OpenApi")
	assertNotContains(t, webAPIProject, "Scalar.AspNetCore")
	assertNotContains(t, webAPIProject, "Microsoft.AspNetCore.Mvc.Testing")
	assertNotContains(t, webAPIProject, "Version=")
	program := contentByPath["ProductService/src/ProductService.WebApi/Program.cs"]
	applicationAssemblyReference := contentByPath["ProductService/src/ProductService.Application/ApplicationAssemblyReference.cs"]
	applicationDI := contentByPath["ProductService/src/ProductService.Application/DependencyInjection.cs"]
	assertContains(t, applicationAssemblyReference, "namespace ProductService.Application")
	assertContains(t, applicationAssemblyReference, "internal static readonly Assembly Assembly = typeof(ApplicationAssemblyReference).Assembly")
	assertContains(t, applicationDI, "namespace ProductService.Application")
	assertContains(t, applicationDI, "RegisterServicesFromAssembly(ApplicationAssemblyReference.Assembly)")
	assertContains(t, applicationDI, "AddOpenBehavior(typeof(ValidationBehavior<,>))")
	assertContains(t, applicationDI, "AddValidatorsFromAssembly(ApplicationAssemblyReference.Assembly)")
	assertContains(t, contentByPath["ProductService/src/ProductService.WebApi/DependencyInjection.cs"], "AddPresentation(this IServiceCollection services)")
	assertContains(t, contentByPath["ProductService/src/ProductService.WebApi/DependencyInjection.cs"], "AddControllers()")
	assertContains(t, contentByPath["ProductService/src/ProductService.WebApi/DependencyInjection.cs"], "AddProblemDetails()")
	assertNotContains(t, contentByPath["ProductService/src/ProductService.WebApi/DependencyInjection.cs"], "AddOpenApi()")
	assertContains(t, contentByPath["ProductService/src/ProductService.WebApi/DependencyInjection.cs"], "ProblemDetailsFactory, ApiProblemDetailsFactory")
	assertContains(t, contentByPath["ProductService/src/ProductService.WebApi/Controllers/ApiController.cs"], "protected ActionResult Problem(IReadOnlyList<Error> errors)")
	assertContains(t, contentByPath["ProductService/src/ProductService.WebApi/Common/Errors/ApiProblemDetailsFactory.cs"], "problemDetails.Extensions[\"traceId\"]")
	assertContains(t, contentByPath["ProductService/src/ProductService.WebApi/Common/Errors/ApiProblemDetailsFactory.cs"], "problemDetails.Extensions[\"errorCodes\"]")
	assertContains(t, program, "AddPresentation()")
	assertContains(t, program, ".AddApplication()")
	assertContains(t, program, ".AddInfrastructure(builder.Configuration)")
	assertNotContains(t, program, "builder.Services.AddControllers()")
	assertNotContains(t, program, "builder.Services.AddProblemDetails()")
	assertNotContains(t, program, "Scalar.AspNetCore")
	assertNotContains(t, program, "MapOpenApi()")
	assertNotContains(t, program, "MapScalarApiReference()")
	assertContains(t, program, "MapControllers()")
	assertNotContains(t, program, "AddMediatR")
	assertNotContains(t, program, "AddValidatorsFromAssembly")
	assertNotContains(t, program, "ValidationBehavior")
	infraDI := contentByPath["ProductService/src/ProductService.Infrastructure/DependencyInjection.cs"]
	readinessProbe := contentByPath["ProductService/src/ProductService.Infrastructure/Health/SqlReadinessProbe.cs"]
	readme := contentByPath["README.md"]
	assertContains(t, infraDI, "IReadinessProbe")
	assertContains(t, infraDI, "ProductService.Infrastructure.Health")
	assertNotContains(t, infraDI, "public sealed class SqlReadinessProbe")
	assertContains(t, infraDI, "IUnitOfWork, UnitOfWork")
	assertContains(t, infraDI, "public const int SqlConnectionTimeoutSeconds = 2")
	assertContains(t, infraDI, "public const int SqlCommandTimeoutSeconds = 2")
	assertContains(t, infraDI, "public const int SqlRetryCount = 1")
	assertContains(t, infraDI, "A lost response after commit is ambiguous")
	assertContains(t, infraDI, "idempotency and operation identity belong at the API/Application boundary")
	assertContains(t, readme, "A connection loss after the database commits but before the acknowledgement leaves the outcome ambiguous")
	assertContains(t, readme, "operation-identity/idempotency boundary")
	assertContains(t, infraDI, "public const int ReadinessTimeoutSeconds = 2")
	assertContains(t, readinessProbe, "namespace ProductService.Infrastructure.Health")
	assertContains(t, readinessProbe, "readinessCts.CancelAfter(TimeSpan.FromSeconds(ResiliencePolicy.ReadinessTimeoutSeconds))")
	assertContains(t, readinessProbe, "dbContext.Database.CanConnectAsync(readinessCts.Token)")
	assertContains(t, readinessProbe, "dbContext.Products.AsNoTracking().Take(1).ToListAsync(readinessCts.Token)")
	assertContains(t, readinessProbe, "DependencyInjection.ActivitySource.StartActivity(\"sql.readiness\")")
	assertContains(t, readinessProbe, "db.readiness.can_connect")
	assertNotContains(t, readinessProbe, "ExpectedSchemaExistsAsync")
	assertNotContains(t, readinessProbe, "INFORMATION_SCHEMA.COLUMNS")
	assertNotContains(t, readinessProbe, "syscolumns.system_type_id = 189")
	assertNotContains(t, readinessProbe, "CreateCommand()")
	assertNotContains(t, program, "CanConnectAsync")
	assertNotContains(t, program, "foreach (var tableName")
	assertNotContains(t, infraDI, "MapGet")
	assertNotContains(t, infraDI, "Results.StatusCode")

	for path, content := range contentByPath {
		for _, forbidden := range []string{"EnsureCreated", "Migrate(", "Database.Migrate", "Password=", "User Id=", "Server=localhost"} {
			if strings.Contains(path, "/tests/") && forbidden == "EnsureCreated" {
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

func TestGatewayFoundationIsRootLevelAndDisabledByDefault(t *testing.T) {
	disabled, err := buildSolutionView(testConfig())
	if err != nil {
		t.Fatalf("build disabled solution view: %v", err)
	}
	if disabled.Gateway.Enabled || disabled.Gateway.Project != (ProjectView{}) || len(disabled.Gateway.Routes) != 0 {
		t.Fatalf("expected disabled gateway to have no model footprint, got %#v", disabled.Gateway)
	}

	enabledConfig := testConfig()
	enabledConfig.Solution.Name = "ShopPlatform"
	enabledConfig.Generation.Gateway.Enabled = true
	enabled, err := buildSolutionView(enabledConfig)
	if err != nil {
		t.Fatalf("build enabled solution view: %v", err)
	}
	if enabled.Gateway.Project.Path != "Gateway/ShopPlatform.Gateway.csproj" {
		t.Fatalf("expected root gateway project path, got %q", enabled.Gateway.Project.Path)
	}
	if enabled.Gateway.Project.Directory != "Gateway" || enabled.Gateway.Project.Name != "ShopPlatform.Gateway" || enabled.Gateway.Project.FileName != "ShopPlatform.Gateway.csproj" {
		t.Fatalf("expected fixed folder with preserved gateway identity, got %#v", enabled.Gateway.Project)
	}
	for _, service := range enabled.Services {
		for _, project := range []ProjectView{service.DomainProject, service.ApplicationProject, service.InfrastructureProject} {
			if project.Name == enabled.Gateway.Project.Name || project.Path == enabled.Gateway.Project.Path {
				t.Fatalf("gateway project leaked into service layer: %#v", project)
			}
		}
	}
}

func TestGenerateGatewayTemplatesAndReverseProxyConfigWhenEnabled(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	cfg := testConfig()
	cfg.Solution.Name = "ShopPlatform"
	cfg.Generation.Gateway.Enabled = true
	cfg.Services = append(cfg.Services, spec.Service{
		Name: "OrderingService",
		Entities: []spec.Entity{{
			Name:   "Order",
			Fields: []spec.Field{{Name: "Id", Type: "Guid"}, {Name: "Number", Type: "string"}},
		}},
	})

	files, err := gen.Generate(cfg)
	if err != nil {
		t.Fatalf("generate gateway config: %v", err)
	}

	gatewayProject := string(generatedContent(t, files, "Gateway/ShopPlatform.Gateway.csproj"))
	gatewayProgram := string(generatedContent(t, files, "Gateway/Program.cs"))
	gatewaySettings := string(generatedContent(t, files, "Gateway/appsettings.json"))
	packages := string(generatedContent(t, files, "Directory.Packages.props"))

	assertContains(t, gatewayProject, `<Project Sdk="Microsoft.NET.Sdk.Web">`)
	assertContains(t, gatewayProject, `<PackageReference Include="Yarp.ReverseProxy" />`)
	assertNotContains(t, gatewayProject, "ProjectReference")
	assertContains(t, gatewayProgram, "builder.Services.AddReverseProxy()")
	assertContains(t, gatewayProgram, `.LoadFromConfig(builder.Configuration.GetSection("ReverseProxy"))`)
	assertContains(t, gatewayProgram, "app.MapReverseProxy();")
	assertContains(t, gatewaySettings, `"product-service-route"`)
	assertContains(t, gatewaySettings, `"product-service-cluster"`)
	assertContains(t, gatewaySettings, `"Path": "/product-service/{**catch-all}"`)
	assertContains(t, gatewaySettings, `"PathRemovePrefix": "/product-service"`)
	assertContains(t, gatewaySettings, `"Address": "http://localhost:5101/"`)
	assertContains(t, gatewaySettings, `"ordering-service-route"`)
	assertContains(t, gatewaySettings, `"ordering-service-cluster"`)
	assertContains(t, gatewaySettings, `"Path": "/ordering-service/{**catch-all}"`)
	assertContains(t, gatewaySettings, `"PathRemovePrefix": "/ordering-service"`)
	assertContains(t, gatewaySettings, `"Address": "http://localhost:5100/"`)
	assertContains(t, packages, `Yarp.ReverseProxy" Version="2.3.0`)
}

func TestGenerateGatewayUpdatesRootSolutionReadmeAndScaffoldPlan(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	cfg := testConfig()
	cfg.Solution.Name = "ShopPlatform"
	cfg.Generation.Gateway.Enabled = true

	files, err := gen.Generate(cfg)
	if err != nil {
		t.Fatalf("generate gateway config: %v", err)
	}
	readme := string(generatedContent(t, files, "README.md"))
	solution := string(generatedContent(t, files, "ShopPlatform.sln"))
	metadata := string(generatedContent(t, files, "microgen.json"))
	view, err := buildSolutionView(cfg)
	if err != nil {
		t.Fatalf("build gateway solution view: %v", err)
	}

	assertContains(t, solution, `ShopPlatform.Gateway", "Gateway/ShopPlatform.Gateway.csproj`)
	assertNotContains(t, solution, `src/ShopPlatform.Gateway/`)
	assertContains(t, solution, `ProductService.Application", "ProductService/src/ProductService.Application/ProductService.Application.csproj`)
	assertContains(t, solution, `ProductService.WebApi", "ProductService/src/ProductService.WebApi/ProductService.WebApi.csproj`)
	assertNotContains(t, solution, `ProductService.Application", "src/ProductService.Application/ProductService.Application.csproj`)
	assertContains(t, readme, `Run gateway`)
	assertContains(t, readme, `ASPNETCORE_URLS=http://localhost:5100 dotnet run --project ./ProductService/src/ProductService.WebApi`)
	assertContains(t, readme, `dotnet run --project ./Gateway`)
	assertNotContains(t, readme, `./src/ShopPlatform.Gateway`)
	assertContains(t, metadata, `"gateway": "ShopPlatform.Gateway"`)
	assertContains(t, metadata, `"gatewayPath": "Gateway/ShopPlatform.Gateway.csproj"`)
	assertNotContains(t, metadata, `src/ShopPlatform.Gateway/`)
	assertContains(t, scaffoldCommandText(view.ScaffoldPlan), "dotnet new web --framework 'net8.0' --name 'ShopPlatform.Gateway' --output './Gateway' --no-restore")
	assertContains(t, scaffoldCommandText(view.ScaffoldPlan), "dotnet sln './ShopPlatform.sln' add './Gateway/ShopPlatform.Gateway.csproj'")
	assertNotContains(t, scaffoldCommandText(view.ScaffoldPlan), "src/ShopPlatform.Gateway")
	assertContains(t, scaffoldPackageText(view.ScaffoldPlan), "Gateway/ShopPlatform.Gateway.csproj -> Yarp.ReverseProxy")
	assertNotContains(t, scaffoldPackageText(view.ScaffoldPlan), "src/ShopPlatform.Gateway")
}

func TestGenerateCreateCQRSSliceUsesApplicationPipelineAndWebApiMapping(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	files, err := gen.Generate(testConfig())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	command := string(generatedContent(t, files, "ProductService/src/ProductService.Application/Products/Commands/Create/CreateProductCommand.cs"))
	handler := string(generatedContent(t, files, "ProductService/src/ProductService.Application/Products/Commands/Create/CreateProductCommandHandler.cs"))
	validator := string(generatedContent(t, files, "ProductService/src/ProductService.Application/Products/Commands/Create/CreateProductCommandValidator.cs"))
	dto := string(generatedContent(t, files, "ProductService/src/ProductService.Application/Products/Dtos/ProductDto.cs"))
	createRequest := string(generatedContent(t, files, "ProductService/src/ProductService.Application/Products/Dtos/CreateProductRequest.cs"))
	controller := string(generatedContent(t, files, "ProductService/src/ProductService.WebApi/Controllers/Products/ProductController.cs"))
	program := string(generatedContent(t, files, "ProductService/src/ProductService.WebApi/Program.cs"))
	validationBehavior := string(generatedContent(t, files, "ProductService/src/ProductService.Application/Common/ValidationBehavior.cs"))
	applicationDI := string(generatedContent(t, files, "ProductService/src/ProductService.Application/DependencyInjection.cs"))
	packages := string(generatedContent(t, files, "Directory.Packages.props"))

	assertContains(t, command, "namespace ProductService.Application.Products.Commands.Create")
	assertContains(t, command, "IRequest<ErrorOr<ProductDto>>")
	assertNotContains(t, command, "ConcurrencyToken")
	assertContains(t, dto, "namespace ProductService.Application.Products.Dtos")
	assertContains(t, dto, "public sealed record ProductDto")
	assertContains(t, createRequest, "public sealed record CreateProductRequest")
	assertNotContains(t, createRequest, "UpdateProductRequest")
	assertContains(t, handler, "IProductRepository repository, IUnitOfWork unitOfWork")
	assertContains(t, handler, "ProductName.Create(request.Name, \"name\")")
	assertContains(t, handler, "repository.AddAsync(entity, cancellationToken)")
	assertContains(t, handler, "unitOfWork.SaveChangesAsync(cancellationToken)")
	assertNotContains(t, handler, "Microsoft.EntityFrameworkCore")
	assertContains(t, validator, "AbstractValidator<CreateProductCommand>")
	assertContains(t, validator, "ProductPrice.Create(value, \"price\")")
	assertContains(t, controller, "ISender sender")
	assertContains(t, controller, "sender.Send(new CreateProductCommand")
	assertNotContains(t, controller, "useCases.CreateAsync")
	assertContains(t, validationBehavior, "IPipelineBehavior<TRequest, TResponse>")
	assertContains(t, validationBehavior, "where TRequest : notnull, IRequest<TResponse>")
	assertContains(t, validationBehavior, "where TResponse : IErrorOr")
	assertContains(t, validationBehavior, "IEnumerable<IValidator<TRequest>> validators")
	assertContains(t, validationBehavior, "return (TResponse)(dynamic)errors;")
	assertNotContains(t, validationBehavior, "IPipelineBehavior<TRequest, ErrorOr<TResponse>>")
	assertContains(t, applicationDI, "AddOpenBehavior(typeof(ValidationBehavior<,>))")
	assertContains(t, applicationDI, "AddValidatorsFromAssembly(ApplicationAssemblyReference.Assembly)")
	assertContains(t, program, ".AddApplication()")
	assertNotContains(t, program, "AddBehavior<ValidationBehavior<CreateProductCommand, ProductDto>>()")
	assertNotContains(t, program, "AddValidatorsFromAssemblyContaining<CreateProductCommandValidator>()")
	assertContains(t, packages, `PackageVersion Include="MediatR" Version="14.2.0"`)
	assertContains(t, packages, `PackageVersion Include="FluentValidation" Version="12.1.1"`)
	assertContains(t, packages, `PackageVersion Include="ErrorOr" Version="2.1.1"`)
}

func TestGenerateReadCQRSSliceUsesQueriesAndPreservesLegacyPaginationContract(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	files, err := gen.Generate(testConfig())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	listQuery := string(generatedContent(t, files, "ProductService/src/ProductService.Application/Products/Queries/List/ListProductQuery.cs"))
	listHandler := string(generatedContent(t, files, "ProductService/src/ProductService.Application/Products/Queries/List/ListProductQueryHandler.cs"))
	listValidator := string(generatedContent(t, files, "ProductService/src/ProductService.Application/Products/Queries/List/ListProductQueryValidator.cs"))
	getQuery := string(generatedContent(t, files, "ProductService/src/ProductService.Application/Products/Queries/GetById/GetProductByIdQuery.cs"))
	getHandler := string(generatedContent(t, files, "ProductService/src/ProductService.Application/Products/Queries/GetById/GetProductByIdQueryHandler.cs"))
	updateCommand := string(generatedContent(t, files, "ProductService/src/ProductService.Application/Products/Commands/Update/UpdateProductCommand.cs"))
	updateHandler := string(generatedContent(t, files, "ProductService/src/ProductService.Application/Products/Commands/Update/UpdateProductCommandHandler.cs"))
	updateValidator := string(generatedContent(t, files, "ProductService/src/ProductService.Application/Products/Commands/Update/UpdateProductCommandValidator.cs"))
	deleteCommand := string(generatedContent(t, files, "ProductService/src/ProductService.Application/Products/Commands/Delete/DeleteProductCommand.cs"))
	deleteHandler := string(generatedContent(t, files, "ProductService/src/ProductService.Application/Products/Commands/Delete/DeleteProductCommandHandler.cs"))
	deleteValidator := string(generatedContent(t, files, "ProductService/src/ProductService.Application/Products/Commands/Delete/DeleteProductCommandValidator.cs"))
	repository := string(generatedContent(t, files, "ProductService/src/ProductService.Application/Products/Interfaces/IProductRepository.cs"))
	controller := string(generatedContent(t, files, "ProductService/src/ProductService.WebApi/Controllers/Products/ProductController.cs"))
	program := string(generatedContent(t, files, "ProductService/src/ProductService.WebApi/Program.cs"))

	assertContains(t, listQuery, "namespace ProductService.Application.Products.Queries.List")
	assertContains(t, listQuery, "IRequest<ErrorOr<PagedResult<ProductDto>>>")
	assertContains(t, listHandler, "PaginationPolicy.Normalize(request.Page, request.PageSize)")
	assertContains(t, listHandler, "repository.ListAsync(normalized.Offset, normalized.PageSize, cancellationToken)")
	assertContains(t, listValidator, "ErrorCode = \"Pagination.Page\"")
	assertContains(t, getQuery, "IRequest<ErrorOr<ProductDto>>")
	assertContains(t, getHandler, "Error.NotFound(code: \"Product.NotFound\"")
	assertContains(t, getHandler, "snapshot.ConcurrencyToken")
	assertContains(t, updateCommand, "IRequest<ErrorOr<ProductDto>>")
	assertContains(t, updateCommand, "ConcurrencyToken")
	assertContains(t, updateHandler, "repository.UpdateAsync(snapshot.Entity, request.ConcurrencyToken")
	assertContains(t, updateHandler, "MutationPreparationStatus.InvalidToken")
	assertNotContains(t, updateHandler, "SaveResultStatus.Conflict")
	assertContains(t, updateHandler, "unitOfWork.SaveChangesAsync(cancellationToken)")
	assertContains(t, updateValidator, "ConcurrencyToken.Required")
	assertContains(t, deleteCommand, "IRequest<ErrorOr<Deleted>>")
	assertContains(t, deleteHandler, "Result.Deleted")
	assertContains(t, deleteHandler, "MutationPreparationStatus.InvalidToken")
	assertNotContains(t, deleteHandler, "SaveResultStatus.Conflict")
	assertContains(t, deleteValidator, "ConcurrencyToken.Required")
	assertContains(t, repository, "namespace ProductService.Application.Products.Interfaces")
	assertContains(t, repository, "Task<MutationPreparationStatus> UpdateAsync")
	assertNotContains(t, repository, "SaveResultStatus")
	assertContains(t, controller, "sender.Send(new ListProductQuery(page, pageSize)")
	assertContains(t, controller, "sender.Send(new GetProductByIdQuery(id)")
	assertNotContains(t, controller, "useCases.ListAsync")
	assertNotContains(t, controller, "useCases.GetByIdAsync")
	assertContains(t, controller, "sender.Send(new UpdateProductCommand")
	assertContains(t, controller, "sender.Send(new DeleteProductCommand")
	assertContains(t, program, ".AddApplication()")
	assertNotContains(t, program, "AddBehavior<ValidationBehavior<ListProductQuery, PagedResult<ProductDto>>>()")
	assertNotContains(t, program, "AddBehavior<ValidationBehavior<GetProductByIdQuery, ProductDto>>()")
	assertNotContains(t, program, "AddBehavior<ValidationBehavior<UpdateProductCommand, ProductDto>>()")
	assertNotContains(t, program, "AddBehavior<ValidationBehavior<DeleteProductCommand, Deleted>>()")
	assertNotContains(t, program, "IProductUseCases")
}

func TestGenerateDirectoryPackagesPropsUsesDependencyPolicy(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	tests := []struct {
		name              string
		targetFramework   string
		aspNetCore        string
		aspNetCoreOpenAPI string
		aspNetCoreTest    string
		entityFramework   string
		sqlClient         string
		cryptographyXML   string
	}{
		{name: "net8", targetFramework: "net8.0", aspNetCore: "8.0.28", aspNetCoreOpenAPI: "8.0.28", aspNetCoreTest: "8.0.28", entityFramework: "8.0.28", sqlClient: "6.1.1", cryptographyXML: "8.0.4"},
		{name: "net9", targetFramework: "net9.0", aspNetCore: "9.0.7", aspNetCoreOpenAPI: "9.0.7", aspNetCoreTest: "9.0.7", entityFramework: "9.0.7", sqlClient: "6.1.1", cryptographyXML: "9.0.18"},
		{name: "net10", targetFramework: "net10.0", aspNetCore: "10.0.0", aspNetCoreOpenAPI: "10.0.0", aspNetCoreTest: "10.0.0", entityFramework: "10.0.0", sqlClient: "6.1.1", cryptographyXML: "10.0.10"},
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
			assertContains(t, packages, `Microsoft.AspNetCore.OpenApi" Version="`+tt.aspNetCoreOpenAPI+`"`)
			assertContains(t, packages, `Microsoft.AspNetCore.Mvc.Testing" Version="`+tt.aspNetCoreTest+`"`)
			assertContains(t, packages, `Microsoft.EntityFrameworkCore.Design" Version="`+tt.entityFramework+`"`)
			assertContains(t, packages, `Microsoft.EntityFrameworkCore.Tools" Version="`+tt.entityFramework+`"`)
			assertContains(t, packages, `Microsoft.EntityFrameworkCore.SqlServer" Version="`+tt.entityFramework+`"`)
			assertContains(t, packages, `Microsoft.Data.SqlClient" Version="`+tt.sqlClient+`"`)
			assertContains(t, packages, `System.Security.Cryptography.Xml" Version="`+tt.cryptographyXML+`"`)
		})
	}
}

func TestGenerateRejectsTargetFrameworkWithoutDependencyPolicy(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	cfg := testConfig()
	cfg.Generation.TargetFramework = "net11.0"

	_, err = gen.Generate(cfg)
	if err == nil {
		t.Fatal("expected generation to reject net11.0")
	}
	if !strings.Contains(err.Error(), "has no verified dependency policy entry") || !strings.Contains(err.Error(), "new target requires an explicit verified policy entry") {
		t.Fatalf("expected explicit policy error, got %v", err)
	}
}

func TestScaffoldPlanUsesTargetFrameworkAndCentralPackagePolicy(t *testing.T) {
	tests := []struct {
		name            string
		targetFramework string
		solutionFormat  string
		entityFramework string
		sqlClient       string
	}{
		{name: "net8", targetFramework: "net8.0", solutionFormat: "sln", entityFramework: "8.0.28", sqlClient: "6.1.1"},
		{name: "net10", targetFramework: "net10.0", solutionFormat: "slnx", entityFramework: "10.0.0", sqlClient: "6.1.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testConfig()
			cfg.Generation.TargetFramework = tt.targetFramework
			view, err := buildSolutionView(cfg)
			if err != nil {
				t.Fatalf("build solution view: %v", err)
			}
			commands := scaffoldCommandText(view.ScaffoldPlan)

			assertContains(t, view.Services[0].SolutionFileName, "ProductService."+tt.solutionFormat)
			solutionCommand := "dotnet new sln --name 'ProductService' --output './ProductService'"
			if tt.solutionFormat == "slnx" {
				solutionCommand = "dotnet new sln --format 'slnx' --name 'ProductService' --output './ProductService'"
			}
			assertContains(t, commands, solutionCommand)
			assertContains(t, commands, "dotnet new webapi --use-controllers --framework '"+tt.targetFramework+"' --name 'ProductService.WebApi' --output './ProductService/src/ProductService.WebApi' --no-restore")
			assertContains(t, commands, "dotnet new classlib --framework '"+tt.targetFramework+"' --name 'ProductService.Domain' --output './ProductService/src/ProductService.Domain' --no-restore")
			assertContains(t, commands, "dotnet new xunit --framework '"+tt.targetFramework+"' --name 'ProductService.WebApi.Tests' --output './ProductService/tests/ProductService.WebApi.Tests' --no-restore")
			assertContains(t, commands, "dotnet sln './ProductService/ProductService."+tt.solutionFormat+"' add './ProductService/src/ProductService.WebApi/ProductService.WebApi.csproj'")
			assertNotContains(t, commands, "dotnet sln './CommercePlatform.")
			assertContains(t, commands, "dotnet add './ProductService/src/ProductService.Infrastructure/ProductService.Infrastructure.csproj' reference './ProductService/src/ProductService.Application/ProductService.Application.csproj'")
			assertContains(t, commands, "dotnet add './ProductService/src/ProductService.WebApi/ProductService.WebApi.csproj' reference './ProductService/src/ProductService.Infrastructure/ProductService.Infrastructure.csproj'")
			assertNotContains(t, commands, "dotnet remove")
			assertNotContains(t, commands, "--version")
			assertNotContains(t, commands, "Version=")
			assertNotContains(t, commands, tt.entityFramework)
			assertNotContains(t, commands, tt.sqlClient)

			packages := scaffoldPackageText(view.ScaffoldPlan)
			assertContains(t, packages, "ProductService/src/ProductService.Infrastructure/ProductService.Infrastructure.csproj -> Microsoft.EntityFrameworkCore.Design")
			assertContains(t, packages, "ProductService/src/ProductService.Infrastructure/ProductService.Infrastructure.csproj -> Microsoft.EntityFrameworkCore.Tools")
			assertContains(t, packages, "ProductService/src/ProductService.Infrastructure/ProductService.Infrastructure.csproj -> Microsoft.EntityFrameworkCore.SqlServer")
			assertContains(t, packages, "ProductService/src/ProductService.Infrastructure/ProductService.Infrastructure.csproj -> Microsoft.Data.SqlClient")
			assertContains(t, packages, "ProductService/src/ProductService.Application/ProductService.Application.csproj -> MediatR")
			assertContains(t, packages, "ProductService/src/ProductService.Application/ProductService.Application.csproj -> FluentValidation")
			assertContains(t, packages, "ProductService/src/ProductService.Application/ProductService.Application.csproj -> FluentValidation.DependencyInjectionExtensions")
			assertContains(t, packages, "ProductService/src/ProductService.Application/ProductService.Application.csproj -> ErrorOr")
			assertContains(t, packages, "ProductService/src/ProductService.WebApi/ProductService.WebApi.csproj -> MediatR")
			if tt.targetFramework == "net8.0" {
				assertNotContains(t, packages, "ProductService/src/ProductService.WebApi/ProductService.WebApi.csproj -> Microsoft.AspNetCore.OpenApi")
				assertNotContains(t, packages, "ProductService/src/ProductService.WebApi/ProductService.WebApi.csproj -> Scalar.AspNetCore")
			} else {
				assertContains(t, packages, "ProductService/src/ProductService.WebApi/ProductService.WebApi.csproj -> Microsoft.AspNetCore.OpenApi")
				assertContains(t, packages, "ProductService/src/ProductService.WebApi/ProductService.WebApi.csproj -> Scalar.AspNetCore")
			}
			assertNotContains(t, packages, "ProductService/src/ProductService.WebApi/ProductService.WebApi.csproj -> FluentValidation.DependencyInjectionExtensions")
			assertContains(t, packages, "ProductService/src/ProductService.WebApi/ProductService.WebApi.csproj -> ErrorOr")
			assertNotContains(t, packages, tt.entityFramework)
			assertNotContains(t, packages, tt.sqlClient)
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
		{name: "net8 below net10", targetFramework: "net8.0", expectedSolution: "ProductService/ProductService.sln", unexpectedSolution: "ProductService/ProductService.slnx"},
		{name: "net10", targetFramework: "net10.0", expectedSolution: "ProductService/ProductService.slnx", unexpectedSolution: "ProductService/ProductService.sln"},
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
			if _, ok := contentByPath["CommercePlatform.sln"]; ok {
				t.Fatal("did not expect root aggregate .sln to be generated")
			}
			if _, ok := contentByPath["CommercePlatform.slnx"]; ok {
				t.Fatal("did not expect root aggregate .slnx to be generated")
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
	slnx := string(generatedContent(t, files, "ProductService/ProductService.slnx"))
	packages := string(generatedContent(t, files, "Directory.Packages.props"))

	assertContains(t, slnx, "<Solution>\n")
	assertContains(t, slnx, `  <Project Path="src/ProductService.WebApi/ProductService.WebApi.csproj" />`)
	assertContains(t, slnx, `  <Project Path="tests/ProductService.Infrastructure.Tests/ProductService.Infrastructure.Tests.csproj" />`)
	assertNotContains(t, slnx, "OrderService")
	assertNotContains(t, slnx, "ProjectConfigurationPlatforms")
	assertContains(t, packages, `System.Security.Cryptography.Xml" Version="10.0.10`)
}

func TestGenerateCreatesOneSolutionPerMicroservice(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	cfg := testConfig()
	cfg.Services = append(cfg.Services, spec.Service{
		Name: "OrderService",
		Entities: []spec.Entity{{
			Name:   "Order",
			Fields: []spec.Field{{Name: "Id", Type: "Guid"}, {Name: "Number", Type: "string"}},
		}},
	})

	files, err := gen.Generate(cfg)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	contentByPath := map[string]string{}
	for _, file := range files {
		contentByPath[file.Path] = string(file.Content)
	}

	productSolution := contentByPath["ProductService/ProductService.sln"]
	orderSolution := contentByPath["OrderService/OrderService.sln"]
	if productSolution == "" || orderSolution == "" {
		t.Fatalf("expected per-service solutions, got paths: %#v", keys(contentByPath))
	}
	if _, ok := contentByPath["CommercePlatform.sln"]; ok {
		t.Fatal("did not expect root aggregate solution")
	}
	assertContains(t, productSolution, `"src/ProductService.WebApi/ProductService.WebApi.csproj"`)
	assertContains(t, productSolution, `"tests/ProductService.Infrastructure.Tests/ProductService.Infrastructure.Tests.csproj"`)
	assertNotContains(t, productSolution, "OrderService")
	assertContains(t, orderSolution, `"src/OrderService.WebApi/OrderService.WebApi.csproj"`)
	assertContains(t, orderSolution, `"tests/OrderService.Infrastructure.Tests/OrderService.Infrastructure.Tests.csproj"`)
	assertNotContains(t, orderSolution, "ProductService")
}

func keys(values map[string]string) []string {
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
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
		{name: "net8 default", targetFramework: "net8.0", expectedSolution: "ProductService/ProductService.sln", unexpectedSolution: "ProductService/ProductService.slnx"},
		{name: "net8 explicit sln", targetFramework: "net8.0", solutionFormat: "sln", expectedSolution: "ProductService/ProductService.sln", unexpectedSolution: "ProductService/ProductService.slnx"},
		{name: "net10 default", targetFramework: "net10.0", expectedSolution: "ProductService/ProductService.slnx", unexpectedSolution: "ProductService/ProductService.sln", validateWithDotnet: true},
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
	controller := string(generatedContent(t, files, "ProductService/src/ProductService.WebApi/Controllers/Categories/CategoryController.cs"))

	assertContains(t, controller, "[Route(\"categories\")]")
	assertContains(t, controller, "ListCategories")
	assertNotContains(t, controller, "Categorys")
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
	content := string(generatedContent(t, files, "ProductService/tests/ProductService.Application.Tests/Features/Products/ProductApplicationTests.cs"))
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

			applicationTests := string(generatedContent(t, files, "RecordService/tests/RecordService.Application.Tests/Features/Records/RecordApplicationTests.cs"))
			infrastructureTests := string(generatedContent(t, files, "RecordService/tests/RecordService.Infrastructure.Tests/RecordServiceInfrastructureTests.cs"))

			for _, content := range []string{applicationTests, infrastructureTests} {
				assertContains(t, content, "using DomainRecord = RecordService.Domain.Entities.Record;")
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
			repository := string(generatedContent(t, files, "RecordService/src/RecordService.Infrastructure/Persistence/Features/Records/RecordRepository.cs"))

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
	repository := string(generatedContent(t, files, "ProductService/src/ProductService.Infrastructure/Persistence/Features/Products/ProductRepository.cs"))
	assertContains(t, repository, "SqlQueryRaw<Guid>(sql, id.Value)")
	assertNotContains(t, repository, "$\"SELECT")
	assertNotContains(t, repository, "'{id.Value}'")
}

func TestGenerateEntityConfigurationQualifiesValueObjectRehydrateWhenNamesCollide(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	cfg := spec.Config{
		Solution: spec.Solution{Name: "OrderPlatform", Description: "Configuration name collision regression."},
		Services: []spec.Service{{
			Name: "OrderService",
			ValueObjects: []spec.ValueObject{{
				Name:        "OrderConfiguration",
				Type:        "string",
				Validations: spec.ValidationRules{Required: boolPtr(true), ValidExample: stringPtr("Standard"), InvalidExample: stringPtr("")},
			}},
			Entities: []spec.Entity{{
				Name: "Order",
				Fields: []spec.Field{
					{Name: "Id", Type: "Guid"},
					{Name: "Configuration", Type: "OrderConfiguration"},
				},
			}},
		}},
	}

	files, err := gen.Generate(cfg)
	if err != nil {
		t.Fatalf("generate collision config: %v", err)
	}
	configuration := string(generatedContent(t, files, "OrderService/src/OrderService.Infrastructure/Persistence/Configurations/OrderConfiguration.cs"))

	assertContains(t, configuration, "public sealed class OrderConfiguration : IEntityTypeConfiguration<Order>")
	assertContains(t, configuration, "value => OrderService.Domain.ValueObjects.OrderConfiguration.Rehydrate(value)")
	assertNotContains(t, configuration, "value => OrderConfiguration.Rehydrate(value)")
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
	createRequest := string(generatedContent(t, files, "HundredFieldService/src/HundredFieldService.Application/HundredRecords/Dtos/CreateHundredRecordRequest.cs"))
	updateRequest := string(generatedContent(t, files, "HundredFieldService/src/HundredFieldService.Application/HundredRecords/Dtos/UpdateHundredRecordRequest.cs"))
	assertContains(t, createRequest, "public sealed record CreateHundredRecordRequest\n{")
	assertContains(t, createRequest, "public string Field099 { get; init; } = string.Empty;")
	assertNotContains(t, createRequest, "public sealed record CreateHundredRecordRequest(")
	assertNotContains(t, updateRequest, "public sealed record UpdateHundredRecordRequest(")
	tests := string(generatedContent(t, files, "HundredFieldService/tests/HundredFieldService.Application.Tests/Features/HundredRecords/HundredRecordApplicationTests.cs"))
	assertContains(t, tests, "new CreateHundredRecordCommand")
	assertContains(t, tests, "ConcurrencyToken = \"token-v1\"")
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
	domainTests := string(generatedContent(t, files, "ProductService/tests/ProductService.Domain.Tests/ProductServiceDomainTests.cs"))
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
	domainTests := string(generatedContent(t, files, "ProductService/tests/ProductService.Domain.Tests/ProductServiceDomainTests.cs"))

	assertContains(t, domainTests, `Assert.Equal("ShortPatternName.Pattern", Assert.Single(ShortPatternName.Create("!!!").Errors).Code);`)
	assertContains(t, domainTests, `var result = ShortPatternName.Create("!!!", "Field");`)
	assertNotContains(t, domainTests, `Assert.Single(ShortPatternName.Create("!!").Errors).Code`)
}

func TestGeneratePreflightDoesNotTreatOptionalStringValueObjectsAsRequired(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	cfg := optionalMaxLengthStringConfig()
	cfg.Generation.EnableValueObjectPreflight = true
	files, err := gen.Generate(cfg)
	if err != nil {
		t.Fatalf("generate optional string config: %v", err)
	}
	preflight := string(generatedContent(t, files, "OptionalStringService/src/OptionalStringService.Infrastructure/Persistence/ValueObjectPreflight.sql"))
	assertContains(t, preflight, "OptionalLabel.MaxLength")
	assertNotContains(t, preflight, "OptionalLabel.Required")
	infraTests := string(generatedContent(t, files, "OptionalStringService/tests/OptionalStringService.Infrastructure.Tests/OptionalStringServiceInfrastructureTests.cs"))
	assertContains(t, infraTests, "VALUES ({0}, N''")
	assertContains(t, infraTests, "Assert.Empty(await RunPreflightAsync(context));")
}

func TestGenerateDisablesValueObjectPreflightByDefault(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	files, err := gen.Generate(testConfig())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	paths := generatedPathSet(files)
	if paths["ProductService/src/ProductService.Infrastructure/Persistence/ValueObjectPreflight.sql"] {
		t.Fatal("expected ValueObjectPreflight.sql to be opt-in")
	}
	csproj := string(generatedContent(t, files, "ProductService/src/ProductService.Infrastructure/ProductService.Infrastructure.csproj"))
	assertNotContains(t, csproj, "ValueObjectPreflight.sql")
	infraTests := string(generatedContent(t, files, "ProductService/tests/ProductService.Infrastructure.Tests/ProductServiceInfrastructureTests.cs"))
	assertNotContains(t, infraTests, "RunPreflightAsync")
	assertNotContains(t, infraTests, "ValueObjectPreflight")
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

func scaffoldCommandText(plan ScaffoldPlan) string {
	commands := make([]string, 0, len(plan.Commands))
	for _, command := range plan.Commands {
		commands = append(commands, command.Command)
	}
	return strings.Join(commands, "\n")
}

func scaffoldPackageText(plan ScaffoldPlan) string {
	entries := make([]string, 0, len(plan.PackageEntries))
	for _, entry := range plan.PackageEntries {
		entries = append(entries, entry.Project+" -> "+entry.Package)
	}
	return strings.Join(entries, "\n")
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
	goldenPath := filepath.Join("testdata", "golden", goldenName)
	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0755); err != nil {
			t.Fatalf("create golden parent: %v", err)
		}
		if err := os.WriteFile(goldenPath, actual, 0644); err != nil {
			t.Fatalf("write golden file: %v", err)
		}
	}
	expected, err := os.ReadFile(goldenPath)
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

func generatedPathSet(files []GeneratedFile) map[string]bool {
	paths := make(map[string]bool, len(files))
	for _, file := range files {
		paths[file.Path] = true
	}
	return paths
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

	services := sortedServices(cfg.Services)
	if len(services) == 0 {
		t.Fatal("expected at least one generated service")
	}
	serviceName := services[0].Name
	solutionName := serviceName + "." + cfg.SolutionFormat()
	return generatedWorkspace{
		dir:          outputDir,
		solutionName: solutionName,
		solutionPath: filepath.Join(outputDir, serviceName, solutionName),
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
