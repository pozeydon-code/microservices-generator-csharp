using System.Net;
using System.Net.Http.Json;
using System.Text.Json;
using Microsoft.AspNetCore.TestHost;
using Microsoft.AspNetCore.Mvc;
using Microsoft.Extensions.DependencyInjection;
using ProductService.Application.Common;
using ProductService.Application.Features.Products;
using Xunit;

namespace ProductService.Api.Tests.Features.Products;

public sealed class ProductEndpointsTests
{
    private static readonly string[] ExpectedValidationErrorKeys = ["name", "price"];
    private static readonly string[] ExpectedProductJsonProperties = ["concurrencyToken", "id", "isActive", "name", "price", ];
    private static readonly string[] AllowedValidationProblemProperties = ["errors", "status", "title", "traceId", "type"];

    [Fact]
    public async Task CrudRoutesRequireAuthentication()
    {
        await using var factory = new TestApiFactory();
        using var client = factory.CreateClient();
        var response = await client.GetAsync("/products");
        Assert.Equal(HttpStatusCode.Unauthorized, response.StatusCode);
    }

    [Fact]
    public async Task AuthorizedListReturnsPaginationMetadata()
    {
        await using var factory = CreateFactory();
        using var client = factory.CreateAuthenticatedClient();
        var response = await client.GetAsync("/products?page=1&pageSize=20");
        Assert.Equal(HttpStatusCode.OK, response.StatusCode);
        var page = await response.Content.ReadFromJsonAsync<PagedResult<ProductDto>>();
        Assert.NotNull(page);
        Assert.Equal(1, page.Page);
        Assert.Equal(20, page.PageSize);
        Assert.Equal(1, page.TotalCount);
        var item = Assert.Single(page.Items);
        Assert.Equal(Guid.Parse("11111111-1111-1111-1111-111111111111"), item.Id);
    }

    [Theory]
    [InlineData("page=1&pageSize=1")]
    [InlineData("page=21474837&pageSize=100")]
    [InlineData("page=2147483647&pageSize=1")]
    public async Task AuthorizedListAcceptsSupportedPaginationOffsets(string query)
    {
        await using var factory = CreateFactory();
        using var client = factory.CreateAuthenticatedClient();
        var response = await client.GetAsync($"/products?{query}");
        Assert.Equal(HttpStatusCode.OK, response.StatusCode);
    }

    [Theory]
    [InlineData("page=0&pageSize=20")]
    [InlineData("page=-1&pageSize=20")]
    [InlineData("page=1&pageSize=0")]
    [InlineData("page=1&pageSize=-1")]
    [InlineData("page=1&pageSize=101")]
    [InlineData("page=21474838&pageSize=100")]
    [InlineData("page=2147483647&pageSize=100")]
    public async Task AuthorizedListRejectsInvalidPagination(string query)
    {
        await using var factory = CreateFactory();
        using var client = factory.CreateAuthenticatedClient();
        var response = await client.GetAsync($"/products?{query}");
        Assert.Equal(HttpStatusCode.BadRequest, response.StatusCode);
        var error = await response.Content.ReadFromJsonAsync<Dictionary<string, string>>();
        Assert.NotNull(error);
        Assert.True(error.ContainsKey("error"));
    }

    [Fact]
    public async Task AuthorizedGetMapsFoundAndMissing()
    {
        await using var factory = CreateFactory();
        using var client = factory.CreateAuthenticatedClient();
        var response = await client.GetAsync("/products/11111111-1111-1111-1111-111111111111");
        Assert.Equal(HttpStatusCode.OK, response.StatusCode);
        var item = await response.Content.ReadFromJsonAsync<ProductDto>();
        Assert.NotNull(item);
        Assert.Equal(Guid.Parse("11111111-1111-1111-1111-111111111111"), item.Id);
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
        Assert.NotNull(response.Headers.Location);
        Assert.Equal("https://localhost/products/11111111-1111-1111-1111-111111111111", response.Headers.Location!.ToString());
        var json = await response.Content.ReadAsStringAsync();
        using var document = JsonDocument.Parse(json);
        AssertProductJson(document.RootElement, true, "Product Prime", 0m, ValidToken);
        var created = await response.Content.ReadFromJsonAsync<ProductDto>();
        Assert.NotNull(created);
        Assert.Equal(Guid.Parse("11111111-1111-1111-1111-111111111111"), created.Id);
        Assert.True(EqualityComparer<bool>.Default.Equals(true, created.IsActive));
        Assert.True(EqualityComparer<string>.Default.Equals("Product Prime", created.Name));
        Assert.True(EqualityComparer<decimal>.Default.Equals(0m, created.Price));
        Assert.Equal(ValidToken, created.ConcurrencyToken);
        Assert.NotNull(FakeProductUseCases.LastCreateRequest);
        Assert.True(EqualityComparer<bool>.Default.Equals(request.IsActive, FakeProductUseCases.LastCreateRequest.IsActive));
        Assert.True(EqualityComparer<string>.Default.Equals(request.Name, FakeProductUseCases.LastCreateRequest.Name));
        Assert.True(EqualityComparer<decimal>.Default.Equals(request.Price, FakeProductUseCases.LastCreateRequest.Price));

    }

    [Fact]
    public async Task AuthorizedCreateMapsValidationOutcomeToProblemDetails()
    {
        await using var factory = CreateFactory();
        using var client = factory.CreateAuthenticatedClient();
        var response = await client.PostAsJsonAsync("/products", new CreateProductRequest { IsActive = true, Name = "", Price = 0m - 1m,  });

        Assert.Equal(HttpStatusCode.BadRequest, response.StatusCode);
        Assert.StartsWith("application/problem+json", response.Content.Headers.ContentType?.MediaType, StringComparison.Ordinal);
        var problemJson = await response.Content.ReadAsStringAsync();
        using var problemDocument = JsonDocument.Parse(problemJson);
        AssertValidationProblemJson(problemDocument.RootElement, ExpectedValidationErrorKeys);
        var problem = await response.Content.ReadFromJsonAsync<ValidationProblemDetails>();
        Assert.NotNull(problem);
        Assert.Equal("Validation failed", problem.Title);
        Assert.Equal(400, problem.Status);
        Assert.Equal(ExpectedValidationErrorKeys, problem.Errors.Keys.OrderBy(key => key, StringComparer.Ordinal).ToArray());
        Assert.Equal(["ProductName.Required: ProductName.Required message."], problem.Errors["name"]);
        Assert.Equal(["ProductPrice.Minimum: ProductPrice.Minimum message."], problem.Errors["price"]);
    }

    [Fact]
    public async Task AuthorizedUpdateMapsValidationOutcomeToProblemDetails()
    {
        await using var factory = CreateFactory();
        using var client = factory.CreateAuthenticatedClient();
        var response = await client.PutAsJsonAsync("/products/11111111-1111-1111-1111-111111111111", new UpdateProductRequest { IsActive = true, Name = "", Price = 0m - 1m, ConcurrencyToken = ValidToken });

        Assert.Equal(HttpStatusCode.BadRequest, response.StatusCode);
        Assert.StartsWith("application/problem+json", response.Content.Headers.ContentType?.MediaType, StringComparison.Ordinal);
        var problemJson = await response.Content.ReadAsStringAsync();
        using var problemDocument = JsonDocument.Parse(problemJson);
        AssertValidationProblemJson(problemDocument.RootElement, ExpectedValidationErrorKeys);
        var problem = await response.Content.ReadFromJsonAsync<ValidationProblemDetails>();
        Assert.NotNull(problem);
        Assert.Equal("Validation failed", problem.Title);
        Assert.Equal(400, problem.Status);
        Assert.Equal(ExpectedValidationErrorKeys, problem.Errors.Keys.OrderBy(key => key, StringComparer.Ordinal).ToArray());
        Assert.Equal(["ProductName.Required: ProductName.Required message."], problem.Errors["name"]);
        Assert.Equal(["ProductPrice.Minimum: ProductPrice.Minimum message."], problem.Errors["price"]);
    }



    [Fact]
    public async Task AuthorizedUpdateMapsOkNotFoundAndConflict()
    {
        await using var factory = CreateFactory();
        using var client = factory.CreateAuthenticatedClient();
        var request = new UpdateProductRequest { IsActive = false, Name = "Product Prime2", Price = 999999.99m, ConcurrencyToken = ValidToken };
        var response = await client.PutAsJsonAsync("/products/11111111-1111-1111-1111-111111111111", request);
        Assert.Equal(HttpStatusCode.OK, response.StatusCode);
        var json = await response.Content.ReadAsStringAsync();
        using var document = JsonDocument.Parse(json);
        AssertProductJson(document.RootElement, false, "Product Prime2", 999999.99m, ValidToken);
        var updated = await response.Content.ReadFromJsonAsync<ProductDto>();
        Assert.NotNull(updated);
        Assert.Equal(Guid.Parse("11111111-1111-1111-1111-111111111111"), updated.Id);
        Assert.True(EqualityComparer<bool>.Default.Equals(false, updated.IsActive));
        Assert.True(EqualityComparer<string>.Default.Equals("Product Prime2", updated.Name));
        Assert.True(EqualityComparer<decimal>.Default.Equals(999999.99m, updated.Price));
        Assert.Equal(ValidToken, updated.ConcurrencyToken);
        Assert.NotNull(FakeProductUseCases.LastUpdateRequest);
        Assert.True(EqualityComparer<bool>.Default.Equals(request.IsActive, FakeProductUseCases.LastUpdateRequest.IsActive));
        Assert.True(EqualityComparer<string>.Default.Equals(request.Name, FakeProductUseCases.LastUpdateRequest.Name));
        Assert.True(EqualityComparer<decimal>.Default.Equals(request.Price, FakeProductUseCases.LastUpdateRequest.Price));

        Assert.Equal(HttpStatusCode.NotFound, (await client.PutAsJsonAsync("/products/22222222-2222-2222-2222-222222222222", request)).StatusCode);
        Assert.Equal(HttpStatusCode.Conflict, (await client.PutAsJsonAsync("/products/33333333-3333-3333-3333-333333333333", request)).StatusCode);
    }

    [Fact]
    public async Task AuthorizedDeleteMapsNoContentNotFoundAndConflict()
    {
        await using var factory = CreateFactory();
        using var client = factory.CreateAuthenticatedClient();
        Assert.Equal(HttpStatusCode.NoContent, (await client.DeleteAsync($"/products/11111111-1111-1111-1111-111111111111?concurrencyToken={Uri.EscapeDataString(ValidToken)}")).StatusCode);
        Assert.Equal(HttpStatusCode.NotFound, (await client.DeleteAsync($"/products/22222222-2222-2222-2222-222222222222?concurrencyToken={Uri.EscapeDataString(ValidToken)}")).StatusCode);
        Assert.Equal(HttpStatusCode.Conflict, (await client.DeleteAsync($"/products/33333333-3333-3333-3333-333333333333?concurrencyToken={Uri.EscapeDataString(ValidToken)}")).StatusCode);
        var invalid = await client.DeleteAsync("/products/11111111-1111-1111-1111-111111111111?concurrencyToken=bad-token");
        Assert.Equal(HttpStatusCode.BadRequest, invalid.StatusCode);
        var error = await invalid.Content.ReadFromJsonAsync<Dictionary<string, string>>();
        Assert.NotNull(error);
        Assert.Equal("Invalid concurrency token.", error["error"]);
    }

    [Theory]
    [InlineData("", HttpStatusCode.BadRequest)]
    [InlineData("unknown-token", HttpStatusCode.BadRequest)]
    [InlineData("expired-token", HttpStatusCode.BadRequest)]
    [InlineData("stale-token", HttpStatusCode.Conflict)]
    [InlineData("token-v0", HttpStatusCode.BadRequest)]
    [InlineData("token-v2", HttpStatusCode.BadRequest)]
    public async Task AuthorizedUpdateMapsConcurrencyTokenOutcomes(string token, HttpStatusCode expectedStatus)
    {
        await using var factory = CreateFactory();
        using var client = factory.CreateAuthenticatedClient();
        var response = await client.PutAsJsonAsync("/products/11111111-1111-1111-1111-111111111111", new UpdateProductRequest { IsActive = false, Name = "Product Prime2", Price = 999999.99m, ConcurrencyToken = token });
        Assert.Equal(expectedStatus, response.StatusCode);
    }

    private static void AssertProductJson(JsonElement root, bool expectedIsActive, string expectedName, decimal expectedPrice, string expectedConcurrencyToken)
    {
        Assert.Equal(JsonValueKind.Object, root.ValueKind);
        Assert.Equal(ExpectedProductJsonProperties.OrderBy(name => name, StringComparer.Ordinal).ToArray(), root.EnumerateObject().Select(property => property.Name).OrderBy(name => name, StringComparer.Ordinal).ToArray());
        Assert.Equal(Guid.Parse("11111111-1111-1111-1111-111111111111"), root.GetProperty("id").GetGuid());
        Assert.Equal(expectedIsActive ? JsonValueKind.True : JsonValueKind.False, root.GetProperty("isActive").ValueKind);
        Assert.Equal(expectedIsActive, root.GetProperty("isActive").GetBoolean());
        Assert.Equal(JsonValueKind.String, root.GetProperty("name").ValueKind);
        Assert.Equal(expectedName, root.GetProperty("name").GetString());
        Assert.Equal(JsonValueKind.Number, root.GetProperty("price").ValueKind);
        Assert.Equal(expectedPrice, root.GetProperty("price").GetDecimal());
        Assert.Equal(JsonValueKind.String, root.GetProperty("concurrencyToken").ValueKind);
        Assert.Equal(expectedConcurrencyToken, root.GetProperty("concurrencyToken").GetString());
    }

    private static void AssertValidationProblemJson(JsonElement root, string[] expectedErrorKeys)
    {
        Assert.Empty(root.EnumerateObject().Select(property => property.Name).Except(AllowedValidationProblemProperties, StringComparer.Ordinal));
        Assert.Equal("Validation failed", root.GetProperty("title").GetString());
        Assert.Equal(400, root.GetProperty("status").GetInt32());
        Assert.Equal(JsonValueKind.Object, root.GetProperty("errors").ValueKind);
        Assert.Equal(expectedErrorKeys.OrderBy(key => key, StringComparer.Ordinal).ToArray(), root.GetProperty("errors").EnumerateObject().Select(property => property.Name).OrderBy(name => name, StringComparer.Ordinal).ToArray());
    }

    private const string ValidToken = "token-v1";

    private static TestApiFactory CreateFactory() => new(builder => builder.ConfigureTestServices(services =>
        services.AddScoped<IProductUseCases, FakeProductUseCases>()));

    private sealed class FakeProductUseCases : IProductUseCases
    {
        public static CreateProductRequest? LastCreateRequest { get; private set; }
        public static UpdateProductRequest? LastUpdateRequest { get; private set; }
        private static readonly ProductDto Item = new(
            Id: Guid.Parse("11111111-1111-1111-1111-111111111111"),
            IsActive: true,
            Name: "Product Prime",
            Price: 0m,
            ConcurrencyToken: ValidToken);
        public Task<PagedResult<ProductDto>> ListAsync(PageRequest request, CancellationToken cancellationToken)
        {
            var normalized = PaginationPolicy.Normalize(request.Page, request.PageSize);
            return Task.FromResult(new PagedResult<ProductDto>([Item], normalized.Page, normalized.PageSize, 1));
        }
        public Task<ProductDto?> GetByIdAsync(Guid id, CancellationToken cancellationToken) => Task.FromResult(id == Item.Id ? Item : null);
        public Task<MutationValidationResult<ProductDto>> CreateAsync(CreateProductRequest request, CancellationToken cancellationToken)
        {
            LastCreateRequest = request;
            List<ValidationIssue> issues = [];
            if (EqualityComparer<string>.Default.Equals(request.Name, "")) issues.Add(new ValidationIssue("name", "ProductName.Required", "ProductName.Required message."));
            if (EqualityComparer<decimal>.Default.Equals(request.Price, 0m - 1m)) issues.Add(new ValidationIssue("price", "ProductPrice.Minimum", "ProductPrice.Minimum message."));
            if (issues.Count > 0) return Task.FromResult(new MutationValidationResult<ProductDto>(MutationResultStatus.ValidationFailed, Validation: new ValidationOutcome(issues)));

            return Task.FromResult(new MutationValidationResult<ProductDto>(MutationResultStatus.Created, new ProductDto(
                Id: Item.Id,
                IsActive: request.IsActive,
                Name: request.Name,
                Price: request.Price,
                ConcurrencyToken: ValidToken)));
        }
        public Task<MutationValidationResult<ProductDto>> UpdateAsync(Guid id, UpdateProductRequest request, CancellationToken cancellationToken)
        {
            LastUpdateRequest = request;
            if (request.ConcurrencyToken == "stale-token")
            {
                return Task.FromResult(new MutationValidationResult<ProductDto>(MutationResultStatus.Conflict));
            }
            if (request.ConcurrencyToken != ValidToken)
            {
                return Task.FromResult(new MutationValidationResult<ProductDto>(MutationResultStatus.InvalidToken));
            }
            List<ValidationIssue> issues = [];
            if (EqualityComparer<string>.Default.Equals(request.Name, "")) issues.Add(new ValidationIssue("name", "ProductName.Required", "ProductName.Required message."));
            if (EqualityComparer<decimal>.Default.Equals(request.Price, 0m - 1m)) issues.Add(new ValidationIssue("price", "ProductPrice.Minimum", "ProductPrice.Minimum message."));
            if (issues.Count > 0) return Task.FromResult(new MutationValidationResult<ProductDto>(MutationResultStatus.ValidationFailed, Validation: new ValidationOutcome(issues)));

            return Task.FromResult(id.ToString().StartsWith("33333333", StringComparison.Ordinal) ? new MutationValidationResult<ProductDto>(MutationResultStatus.Conflict) : id == Item.Id ? new MutationValidationResult<ProductDto>(MutationResultStatus.Updated, new ProductDto(
                Id: Item.Id,
                IsActive: request.IsActive,
                Name: request.Name,
                Price: request.Price,
                ConcurrencyToken: ValidToken)) : new MutationValidationResult<ProductDto>(MutationResultStatus.NotFound));
        }
        public Task<MutationResult<bool>> DeleteAsync(Guid id, string concurrencyToken, CancellationToken cancellationToken) => Task.FromResult(concurrencyToken != ValidToken ? new MutationResult<bool>(MutationResultStatus.InvalidToken) : id.ToString().StartsWith("33333333", StringComparison.Ordinal) ? new MutationResult<bool>(MutationResultStatus.Conflict) : id == Item.Id ? new MutationResult<bool>(MutationResultStatus.Deleted, true) : new MutationResult<bool>(MutationResultStatus.NotFound));
    }
}
