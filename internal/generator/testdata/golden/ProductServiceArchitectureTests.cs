using System.Reflection;
using Xunit;

namespace ProductService.Architecture.Tests;

public sealed class ProductServiceArchitectureTests
{
    [Fact]
    public void RuntimeAssemblyReferencesFollowCleanArchitectureBoundaries()
    {
        var domain = typeof(ProductService.Domain.Entities.Product).Assembly;
        var application = typeof(ProductService.Application.Products.Commands.Create.CreateProductCommand).Assembly;
        var webApi = typeof(ProductService.WebApi.Controllers.Products.ProductController).Assembly;
        var infrastructure = typeof(ProductService.Infrastructure.DependencyInjection).Assembly;
        var webApiCompositionRoot = typeof(Program).Assembly;

        AssertReferencesOnly(domain, []);
        AssertReferencesOnly(application, [domain.GetName().Name!]);
        AssertReferencesOnly(infrastructure, [application.GetName().Name!, domain.GetName().Name!]);
        AssertReferencesOnly(webApi, [application.GetName().Name!, infrastructure.GetName().Name!]);
        AssertReferencesOnly(webApiCompositionRoot, [application.GetName().Name!, infrastructure.GetName().Name!]);
    }

    [Fact]
    public void FeaturesUseMediatRAndDoNotExposeLegacyApplicationServices()
    {
        var root = FindSolutionRoot();
        AssertSourceContains(root, "src/ProductService/ProductService.Application/Products", [
            "CreateProductCommand : IRequest<ErrorOr<ProductDto>>",
            "ListProductQuery(int? Page, int? PageSize) : IRequest<ErrorOr<PagedResult<ProductDto>>>",
            "GetProductByIdQuery(Guid Id) : IRequest<ErrorOr<ProductDto>>",
            "UpdateProductCommand : IRequest<ErrorOr<ProductDto>>",
            "DeleteProductCommand(Guid Id, string ConcurrencyToken) : IRequest<ErrorOr<Deleted>>"
        ]);
        AssertSourceDoesNotContain(root, "src/ProductService/ProductService.Application/Products", ["IProductUseCases", "ProductUseCases"]);
    }

    [Fact]
    public void ProjectFilesDeclareExactReferencesAndAllowedFrameworks()
    {
        var root = FindSolutionRoot();
        AssertProject(root, "src/ProductService/ProductService.Domain/ProductService.Domain.csproj", [], [], []);
        AssertProject(root, "src/ProductService/ProductService.Application/ProductService.Application.csproj", ["..\\ProductService.Domain\\ProductService.Domain.csproj"], ["ErrorOr", "FluentValidation", "MediatR"], []);
        AssertProject(root, "src/ProductService/ProductService.Infrastructure/ProductService.Infrastructure.csproj", ["..\\ProductService.Application\\ProductService.Application.csproj", "..\\ProductService.Domain\\ProductService.Domain.csproj"], ["Microsoft.Data.SqlClient", "Microsoft.EntityFrameworkCore.Design", "Microsoft.EntityFrameworkCore.SqlServer", "Microsoft.EntityFrameworkCore.Tools"], []);
        AssertProject(root, "src/ProductService/ProductService.WebApi/ProductService.WebApi.csproj", ["..\\ProductService.Application\\ProductService.Application.csproj", "..\\ProductService.Infrastructure\\ProductService.Infrastructure.csproj"], ["ErrorOr", "FluentValidation.DependencyInjectionExtensions", "MediatR", "Microsoft.AspNetCore.Authentication.JwtBearer", "OpenTelemetry.Exporter.OpenTelemetryProtocol", "OpenTelemetry.Extensions.Hosting", "OpenTelemetry.Instrumentation.AspNetCore", "OpenTelemetry.Instrumentation.Http"], []);
    }

    [Fact]
    public void SourceTextKeepsStrictBoundaryVocabularyOutOfInwardLayers()
    {
        var root = FindSolutionRoot();
        AssertSourceDoesNotContain(root, "src/ProductService/ProductService.Domain", ["RowVersion", "Microsoft.EntityFrameworkCore", "Microsoft.AspNetCore", "IServiceCollection", "MediatR", "FluentValidation", "ErrorOr"]);
        var applicationTokenRepresentationTerms = new[] { "Convert.FromBase64String", "Convert.ToBase64String", "Base64", "RowVersion", "byte[8]", "new byte[", "byte[]" };
        AssertSourceDoesNotContain(root, "src/ProductService/ProductService.Application", [.. applicationTokenRepresentationTerms, "Microsoft.EntityFrameworkCore", "Microsoft.AspNetCore", "ProductService.Infrastructure", "IServiceCollection", "PersistenceMutationStatus"]);
        AssertSourceDoesNotContain(root, "tests/ProductService/ProductService.Application.Tests", applicationTokenRepresentationTerms);
        AssertSourceDoesNotContain(root, "src/ProductService/ProductService.WebApi", ["Microsoft.EntityFrameworkCore", "DbContext", "INFORMATION_SCHEMA", "Sql", "IEndpointRouteBuilder", "MapGet"]);
        AssertSourceDoesNotContain(root, "src/ProductService/ProductService.Infrastructure", ["IEndpointRouteBuilder", "Results.", "StatusCodes", "MapGet", "Microsoft.AspNetCore.Routing", "Microsoft.AspNetCore.Http"]);
        AssertSourceDoesNotContain(root, "src/ProductService/ProductService.Infrastructure", ["$\"SELECT", "'{id.Value}'", "WHERE [Id] = '"]);
    }

    private static void AssertReferencesOnly(Assembly assembly, IReadOnlyCollection<string> allowedProjectReferences)
    {
        var actualProjectReferences = assembly.GetReferencedAssemblies()
            .Select(reference => reference.Name!)
            .Where(name => name.StartsWith("ProductService.", StringComparison.Ordinal))
            .OrderBy(name => name)
            .ToArray();
        Assert.Equal(allowedProjectReferences.OrderBy(name => name).ToArray(), actualProjectReferences);
    }

    private static void AssertProject(string root, string relativePath, IReadOnlyCollection<string> projectReferences, IReadOnlyCollection<string> packageReferences, IReadOnlyCollection<string> frameworkReferences)
    {
        var content = File.ReadAllText(Path.Combine(root, relativePath));
        Assert.Equal(projectReferences.OrderBy(x => x), ExtractIncludes(content, "ProjectReference").OrderBy(x => x));
        Assert.Equal(packageReferences.OrderBy(x => x), ExtractIncludes(content, "PackageReference").OrderBy(x => x));
        Assert.Equal(frameworkReferences.OrderBy(x => x), ExtractIncludes(content, "FrameworkReference").OrderBy(x => x));
    }

    private static string[] ExtractIncludes(string content, string elementName) => content.Split('\n')
        .Where(line => line.Contains("<" + elementName, StringComparison.Ordinal))
        .Select(line => line.Split("Include=\"", StringSplitOptions.None)[1].Split('"')[0])
        .ToArray();

    private static void AssertSourceDoesNotContain(string root, string relativeDirectory, IReadOnlyCollection<string> forbidden)
    {
        var source = Directory.EnumerateFiles(Path.Combine(root, relativeDirectory), "*.cs", SearchOption.AllDirectories)
            .Select(File.ReadAllText);
        var content = string.Join('\n', source);
        foreach (var forbiddenText in forbidden)
        {
            Assert.DoesNotContain(forbiddenText, content, StringComparison.Ordinal);
        }
    }

    private static void AssertSourceContains(string root, string relativeDirectory, IReadOnlyCollection<string> expected)
    {
        var source = Directory.EnumerateFiles(Path.Combine(root, relativeDirectory), "*.cs", SearchOption.AllDirectories)
            .Select(File.ReadAllText);
        var content = string.Join('\n', source);
        foreach (var expectedText in expected)
        {
            Assert.Contains(expectedText, content, StringComparison.Ordinal);
        }
    }

    private static string FindSolutionRoot()
    {
        var current = new DirectoryInfo(AppContext.BaseDirectory);
        while (current is not null && !current.EnumerateFiles("*.sln").Any() && !current.EnumerateFiles("*.slnx").Any())
        {
            current = current.Parent;
        }
        return current?.FullName ?? throw new InvalidOperationException("Could not locate generated solution root.");
    }
}
