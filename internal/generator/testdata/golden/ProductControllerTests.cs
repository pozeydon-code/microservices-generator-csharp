using System.Net;
using System.Net.Http.Json;
using System.Text.Json;
using Microsoft.AspNetCore.Mvc;
using Microsoft.AspNetCore.TestHost;
using Microsoft.Extensions.DependencyInjection;
using ProductService.Application.Common;
using ProductService.Application.Features.Products;
using ProductService.Application.Features.Products.Create;
using ProductService.Application.Features.Products.Delete;
using ProductService.Application.Features.Products.GetById;
using ProductService.Application.Features.Products.List;
using ProductService.Application.Features.Products.Update;
using ErrorOr;
using MediatR;
using Xunit;

namespace ProductService.WebApi.Tests.Features.Products;

public sealed class ProductEndpointsTests
{
    private static readonly string[] ExpectedValidationErrorKeys = ["name", "price"];
    private static readonly string[] ExpectedProductJsonProperties = ["concurrencyToken", "id", "isActive", "name", "price", ];
    private static readonly string[] AllowedValidationProblemProperties = ["errors", "status", "title", "traceId", "type"];
    private static readonly ProductDto Item = new(
        Id: Guid.Parse("11111111-1111-1111-1111-111111111111"),
        IsActive: true,
        Name: "Product Prime",
        Price: 0m,
        ConcurrencyToken: ValidToken);

    [Fact]
    public async Task CrudRoutesRequireAuthentication()
    {
        await using var factory = new TestWebApiFactory();
        using var client = factory.CreateClient();

        Assert.Equal(HttpStatusCode.Unauthorized, (await client.GetAsync("/products")).StatusCode);
    }

    [Fact]
    public async Task AuthorizedListAndGetMapSuccessAndNotFound()
    {
        await using var factory = CreateFactory();
        using var client = factory.CreateAuthenticatedClient();

        var pageResponse = await client.GetAsync("/products?page=1&pageSize=20");
        Assert.Equal(HttpStatusCode.OK, pageResponse.StatusCode);
        var page = await pageResponse.Content.ReadFromJsonAsync<PagedResult<ProductDto>>();
        Assert.NotNull(page);
        Assert.Single(page.Items);

        var getResponse = await client.GetAsync("/products/11111111-1111-1111-1111-111111111111");
        Assert.Equal(HttpStatusCode.OK, getResponse.StatusCode);
        Assert.Equal(HttpStatusCode.NotFound, (await client.GetAsync("/products/22222222-2222-2222-2222-222222222222")).StatusCode);
    }

    [Fact]
    public async Task AuthorizedCreateReturnsCreatedLocationAndBody()
    {
        await using var factory = CreateFactory();
        using var client = factory.CreateAuthenticatedClient();
        var request = new CreateProductRequest { IsActive = true, Name = "Product Prime", Price = 0m,  };

        var response = await client.PostAsJsonAsync("/products", request);

        Assert.Equal(HttpStatusCode.Created, response.StatusCode);
        Assert.Equal("https://localhost/products/11111111-1111-1111-1111-111111111111", response.Headers.Location!.ToString());
        var created = await response.Content.ReadFromJsonAsync<ProductDto>();
        Assert.NotNull(created);
        Assert.True(EqualityComparer<bool>.Default.Equals(request.IsActive, created.IsActive));
        Assert.True(EqualityComparer<string>.Default.Equals(request.Name, created.Name));
        Assert.True(EqualityComparer<decimal>.Default.Equals(request.Price, created.Price));
        Assert.Equal(ValidToken, created.ConcurrencyToken);
    }

    [Fact]
    public async Task AuthorizedCreateAndUpdateMapValidationToProblemDetails()
    {
        await using var factory = CreateFactory();
        using var client = factory.CreateAuthenticatedClient();
        var createResponse = await client.PostAsJsonAsync("/products", new CreateProductRequest { IsActive = true, Name = "", Price = 0m - 1m,  });
        var updateResponse = await client.PutAsJsonAsync("/products/11111111-1111-1111-1111-111111111111", new UpdateProductRequest { IsActive = true, Name = "", Price = 0m - 1m,  ConcurrencyToken = ValidToken });

        await AssertValidationProblem(createResponse, ExpectedValidationErrorKeys);
        await AssertValidationProblem(updateResponse, ExpectedValidationErrorKeys);
    }

    [Fact]
    public async Task AuthorizedUpdateMapsSuccessNotFoundConflictAndRequestValues()
    {
        await using var factory = CreateFactory();
        using var client = factory.CreateAuthenticatedClient();
        var request = new UpdateProductRequest { IsActive = false, Name = "Product Prime2", Price = 999999.99m,  ConcurrencyToken = ValidToken };

        var response = await client.PutAsJsonAsync("/products/11111111-1111-1111-1111-111111111111", request);

        Assert.Equal(HttpStatusCode.OK, response.StatusCode);
        var updated = await response.Content.ReadFromJsonAsync<ProductDto>();
        Assert.NotNull(updated);
        Assert.True(EqualityComparer<bool>.Default.Equals(request.IsActive, updated.IsActive));
        Assert.True(EqualityComparer<string>.Default.Equals(request.Name, updated.Name));
        Assert.True(EqualityComparer<decimal>.Default.Equals(request.Price, updated.Price));
        Assert.Equal(ValidToken, updated.ConcurrencyToken);
        Assert.Equal(HttpStatusCode.NotFound, (await client.PutAsJsonAsync("/products/22222222-2222-2222-2222-222222222222", request)).StatusCode);
        Assert.Equal(HttpStatusCode.Conflict, (await client.PutAsJsonAsync("/products/33333333-3333-3333-3333-333333333333", request)).StatusCode);
    }

    [Theory]
    [InlineData("", HttpStatusCode.BadRequest)]
    [InlineData("unknown-token", HttpStatusCode.BadRequest)]
    [InlineData("stale-token", HttpStatusCode.Conflict)]
    public async Task AuthorizedUpdateMapsConcurrencyTokenOutcomes(string token, HttpStatusCode expectedStatus)
    {
        await using var factory = CreateFactory();
        using var client = factory.CreateAuthenticatedClient();
        var response = await client.PutAsJsonAsync("/products/11111111-1111-1111-1111-111111111111", new UpdateProductRequest { IsActive = false, Name = "Product Prime2", Price = 999999.99m,  ConcurrencyToken = token });

        Assert.Equal(expectedStatus, response.StatusCode);
        if (token == "unknown-token")
        {
            var error = await response.Content.ReadFromJsonAsync<Dictionary<string, string>>();
            Assert.Equal("Invalid concurrency token.", error!["error"]);
        }
    }

    [Fact]
    public async Task AuthorizedDeleteMapsNoContentNotFoundConflictAndInvalidToken()
    {
        await using var factory = CreateFactory();
        using var client = factory.CreateAuthenticatedClient();
        var route = "/products";

        Assert.Equal(HttpStatusCode.NoContent, (await client.DeleteAsync($"{route}/11111111-1111-1111-1111-111111111111?concurrencyToken={Uri.EscapeDataString(ValidToken)}")).StatusCode);
        Assert.Equal(HttpStatusCode.NotFound, (await client.DeleteAsync($"{route}/22222222-2222-2222-2222-222222222222?concurrencyToken={Uri.EscapeDataString(ValidToken)}")).StatusCode);
        Assert.Equal(HttpStatusCode.Conflict, (await client.DeleteAsync($"{route}/33333333-3333-3333-3333-333333333333?concurrencyToken=stale-token")).StatusCode);
        Assert.Equal(HttpStatusCode.BadRequest, (await client.DeleteAsync($"{route}/11111111-1111-1111-1111-111111111111?concurrencyToken=bad-token")).StatusCode);
    }

    private static async Task AssertValidationProblem(HttpResponseMessage response, string[] expectedKeys)
    {
        Assert.Equal(HttpStatusCode.BadRequest, response.StatusCode);
        Assert.StartsWith("application/problem+json", response.Content.Headers.ContentType?.MediaType, StringComparison.Ordinal);
        var problem = await response.Content.ReadFromJsonAsync<ValidationProblemDetails>();
        Assert.NotNull(problem);
        Assert.Equal("Validation failed", problem.Title);
        Assert.Equal(expectedKeys, problem.Errors.Keys.OrderBy(key => key, StringComparer.Ordinal).ToArray());
    }

    private const string ValidToken = "token-v1";

    private static TestWebApiFactory CreateFactory() => new(builder => builder.ConfigureTestServices(services =>
    {
        services.AddScoped<IRequestHandler<ListProductQuery, ErrorOr<PagedResult<ProductDto>>>, FakeListProductQueryHandler>();
        services.AddScoped<IRequestHandler<GetProductByIdQuery, ErrorOr<ProductDto>>, FakeGetProductByIdQueryHandler>();
        services.AddScoped<IRequestHandler<CreateProductCommand, ErrorOr<ProductDto>>, FakeCreateProductHandler>();
        services.AddScoped<IRequestHandler<UpdateProductCommand, ErrorOr<ProductDto>>, FakeUpdateProductHandler>();
        services.AddScoped<IRequestHandler<DeleteProductCommand, ErrorOr<Deleted>>, FakeDeleteProductHandler>();
    }));

    private sealed class FakeListProductQueryHandler : IRequestHandler<ListProductQuery, ErrorOr<PagedResult<ProductDto>>>
    {
        public Task<ErrorOr<PagedResult<ProductDto>>> Handle(ListProductQuery request, CancellationToken cancellationToken)
        {
            var normalized = PaginationPolicy.Normalize(request.Page, request.PageSize);
            return Task.FromResult<ErrorOr<PagedResult<ProductDto>>>(new PagedResult<ProductDto>([Item], normalized.Page, normalized.PageSize, 1));
        }
    }

    private sealed class FakeGetProductByIdQueryHandler : IRequestHandler<GetProductByIdQuery, ErrorOr<ProductDto>>
    {
        public Task<ErrorOr<ProductDto>> Handle(GetProductByIdQuery request, CancellationToken cancellationToken) =>
            Task.FromResult<ErrorOr<ProductDto>>(request.Id == Item.Id ? Item : Error.NotFound(code: "Product.NotFound", description: "Product was not found."));
    }

    private sealed class FakeCreateProductHandler : IRequestHandler<CreateProductCommand, ErrorOr<ProductDto>>
    {
        public Task<ErrorOr<ProductDto>> Handle(CreateProductCommand request, CancellationToken cancellationToken) =>
            Task.FromResult<ErrorOr<ProductDto>>(new ProductDto(
                Id: Item.Id,
                IsActive: request.IsActive,
                Name: request.Name,
                Price: request.Price,
                ConcurrencyToken: ValidToken));
    }

    private sealed class FakeUpdateProductHandler : IRequestHandler<UpdateProductCommand, ErrorOr<ProductDto>>
    {
        public Task<ErrorOr<ProductDto>> Handle(UpdateProductCommand request, CancellationToken cancellationToken)
        {
            if (request.ConcurrencyToken == "stale-token") return Task.FromResult<ErrorOr<ProductDto>>(Error.Conflict(code: "Product.ConcurrencyConflict", description: "Product was changed by another request."));
            if (request.ConcurrencyToken != ValidToken) return Task.FromResult<ErrorOr<ProductDto>>(Error.Validation(code: "ConcurrencyToken.Invalid", description: "Invalid concurrency token."));
            if (request.Id == Item.Id) return Task.FromResult<ErrorOr<ProductDto>>(new ProductDto(
                Id: Item.Id,
                IsActive: request.IsActive,
                Name: request.Name,
                Price: request.Price,
                ConcurrencyToken: ValidToken));
            return Task.FromResult<ErrorOr<ProductDto>>(request.Id.ToString().StartsWith("33333333", StringComparison.Ordinal) ? Error.Conflict(code: "Product.ConcurrencyConflict", description: "Product was changed by another request.") : Error.NotFound(code: "Product.NotFound", description: "Product was not found."));
        }
    }

    private sealed class FakeDeleteProductHandler : IRequestHandler<DeleteProductCommand, ErrorOr<Deleted>>
    {
        public Task<ErrorOr<Deleted>> Handle(DeleteProductCommand request, CancellationToken cancellationToken)
        {
            if (request.ConcurrencyToken == "stale-token") return Task.FromResult<ErrorOr<Deleted>>(Error.Conflict(code: "Product.ConcurrencyConflict", description: "Product was changed by another request."));
            if (request.ConcurrencyToken != ValidToken) return Task.FromResult<ErrorOr<Deleted>>(Error.Validation(code: "ConcurrencyToken.Invalid", description: "Invalid concurrency token."));
            return Task.FromResult<ErrorOr<Deleted>>(request.Id == Item.Id ? Result.Deleted : Error.NotFound(code: "Product.NotFound", description: "Product was not found."));
        }
    }
}
