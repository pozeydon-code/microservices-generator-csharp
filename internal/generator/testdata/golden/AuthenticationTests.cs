using System.Net;
using System.Net.Http.Json;
using Microsoft.AspNetCore.TestHost;
using Microsoft.Extensions.DependencyInjection;
using ProductService.Application.Common;
using ProductService.Application.Features.Products;
using Xunit;

[assembly: CollectionBehavior(DisableTestParallelization = true)]

namespace ProductService.Api.Tests;

public sealed class AuthenticationTests
{
    [Fact]
    public async Task ValidBearerGetsExpectedSuccessfulCrudBody()
    {
        await using var factory = new TestApiFactory(builder => builder.ConfigureTestServices(services =>
            services.AddScoped<IProductUseCases, AuthProductUseCases>()));
        using var client = factory.CreateAuthenticatedClient();
        var response = await client.GetAsync("/products");
        Assert.Equal(HttpStatusCode.OK, response.StatusCode);
        var body = await response.Content.ReadFromJsonAsync<PagedResult<ProductDto>>();
        Assert.NotNull(body);
        Assert.Equal(1, body.Page);
        Assert.Equal(20, body.PageSize);
        Assert.Equal(1, body.TotalCount);
        var item = Assert.Single(body.Items);
        Assert.Equal(Guid.Parse("11111111-1111-1111-1111-111111111111"), item.Id);
        Assert.True(EqualityComparer<bool>.Default.Equals(true, item.IsActive));
        Assert.True(EqualityComparer<string>.Default.Equals("Product Prime", item.Name));
        Assert.True(EqualityComparer<decimal>.Default.Equals(0m, item.Price));
        Assert.Equal("token-v1", item.ConcurrencyToken);
    }

    [Theory]
    [InlineData(null)]
    [InlineData("expired")]
    [InlineData("malformed")]
    [InlineData("wrong-signature")]
    [InlineData("wrong-issuer")]
    [InlineData("wrong-audience")]
    public async Task InvalidOrMissingBearerCannotReachCrudRoute(string? tokenKind)
    {
        await using var factory = new TestApiFactory();
        using var client = tokenKind switch
        {
            null => factory.CreateClient(new Microsoft.AspNetCore.Mvc.Testing.WebApplicationFactoryClientOptions { BaseAddress = new Uri("https://localhost") }),
            "expired" => factory.CreateBearerClient(TestJwtTokens.ExpiredToken()),
            "malformed" => factory.CreateBearerClient("not-a-jwt"),
            "wrong-signature" => factory.CreateBearerClient(TestJwtTokens.WrongSignatureToken()),
            "wrong-issuer" => factory.CreateBearerClient(TestJwtTokens.WrongIssuerToken()),
            "wrong-audience" => factory.CreateBearerClient(TestJwtTokens.WrongAudienceToken()),
            _ => throw new InvalidOperationException("Unknown token kind.")
        };
        var response = await client.GetAsync("/products");
        Assert.Equal(HttpStatusCode.Unauthorized, response.StatusCode);
    }

    [Fact]
    public async Task PlainHttpRedirectsToHttps()
    {
        await using var factory = new TestApiFactory();
        using var client = factory.CreateClient(new Microsoft.AspNetCore.Mvc.Testing.WebApplicationFactoryClientOptions
        {
            BaseAddress = new Uri("http://localhost"),
            AllowAutoRedirect = false
        });
        var response = await client.GetAsync("/health/live");
        Assert.Equal(HttpStatusCode.TemporaryRedirect, response.StatusCode);
        Assert.NotNull(response.Headers.Location);
        Assert.Equal("https://localhost/health/live", response.Headers.Location!.ToString());
    }

    [Fact]
    public async Task ProductionHttpsResponsesIncludeHsts()
    {
        await using var factory = TestApiFactory.Production();
        using var client = factory.CreateClient(new Microsoft.AspNetCore.Mvc.Testing.WebApplicationFactoryClientOptions { BaseAddress = new Uri("https://localhost") });
        var response = await client.GetAsync("/health/live");
        Assert.Equal(HttpStatusCode.OK, response.StatusCode);
        Assert.True(response.Headers.Contains("Strict-Transport-Security"));
    }

    [Fact]
    public async Task SlowUseCaseTriggersRequestTimeoutAndCancellation()
    {
        SlowProductUseCases.CancellationObserved = false;
        await using var factory = new TestApiFactory(builder => builder.ConfigureTestServices(services =>
            services.AddScoped<IProductUseCases, SlowProductUseCases>()));
        using var client = factory.CreateAuthenticatedClient();
        var started = DateTimeOffset.UtcNow;
        var response = await client.GetAsync("/products");
        var elapsed = DateTimeOffset.UtcNow - started;
        Assert.Equal(HttpStatusCode.ServiceUnavailable, response.StatusCode);
        Assert.True(elapsed < TimeSpan.FromSeconds(20));
        Assert.True(SlowProductUseCases.CancellationObserved);
        var problem = await response.Content.ReadFromJsonAsync<Dictionary<string, object>>();
        Assert.NotNull(problem);
        Assert.True(problem.ContainsKey("status"));
    }

    [Fact]
    public async Task RequestTimeoutBudgetValidationHasExactBoundary()
    {
        await using var rejectedFactory = new TestApiFactory(requestTimeoutSeconds: 9);
        var ex = await Assert.ThrowsAsync<InvalidOperationException>(async () =>
        {
            using var client = rejectedFactory.CreateClient();
            await client.GetAsync("/health/live");
        });
        Assert.Contains("at least 10 seconds", ex.ToString());

        await using var acceptedBoundaryFactory = new TestApiFactory(requestTimeoutSeconds: 10);
        using var acceptedBoundaryClient = acceptedBoundaryFactory.CreateClient(new Microsoft.AspNetCore.Mvc.Testing.WebApplicationFactoryClientOptions { BaseAddress = new Uri("https://localhost") });
        var boundaryResponse = await acceptedBoundaryClient.GetAsync("/health/live");
        Assert.Equal(HttpStatusCode.OK, boundaryResponse.StatusCode);

        await using var defaultFactory = new TestApiFactory();
        using var defaultClient = defaultFactory.CreateClient(new Microsoft.AspNetCore.Mvc.Testing.WebApplicationFactoryClientOptions { BaseAddress = new Uri("https://localhost") });
        var defaultResponse = await defaultClient.GetAsync("/health/live");
        Assert.Equal(HttpStatusCode.OK, defaultResponse.StatusCode);
    }

    private sealed class AuthProductUseCases : IProductUseCases
    {
        private static readonly ProductDto Item = new(
            Id: Guid.Parse("11111111-1111-1111-1111-111111111111"),
            IsActive: true,
            Name: "Product Prime",
            Price: 0m,
            ConcurrencyToken: "token-v1");
        public Task<PagedResult<ProductDto>> ListAsync(PageRequest request, CancellationToken cancellationToken) => Task.FromResult(new PagedResult<ProductDto>([Item], 1, 20, 1));
        public Task<ProductDto?> GetByIdAsync(Guid id, CancellationToken cancellationToken) => Task.FromResult<ProductDto?>(Item);
        public Task<MutationValidationResult<ProductDto>> CreateAsync(CreateProductRequest request, CancellationToken cancellationToken) => Task.FromResult(new MutationValidationResult<ProductDto>(MutationResultStatus.Created, Item));
        public Task<MutationValidationResult<ProductDto>> UpdateAsync(Guid id, UpdateProductRequest request, CancellationToken cancellationToken) => Task.FromResult(new MutationValidationResult<ProductDto>(MutationResultStatus.Updated, Item));
        public Task<MutationResult<bool>> DeleteAsync(Guid id, string concurrencyToken, CancellationToken cancellationToken) => Task.FromResult(new MutationResult<bool>(MutationResultStatus.Deleted, true));
    }

    private sealed class SlowProductUseCases : IProductUseCases
    {
        public static bool CancellationObserved { get; set; }
        public async Task<PagedResult<ProductDto>> ListAsync(PageRequest request, CancellationToken cancellationToken)
        {
            using var registration = cancellationToken.UnsafeRegister(_ => CancellationObserved = true, null);
            try
            {
                await Task.Delay(TimeSpan.FromSeconds(30), cancellationToken);
            }
            catch (OperationCanceledException)
            {
                CancellationObserved = true;
                throw;
            }
            throw new InvalidOperationException("Timeout test should cancel before completion.");
        }
        public Task<ProductDto?> GetByIdAsync(Guid id, CancellationToken cancellationToken) => throw new NotSupportedException();
        public Task<MutationValidationResult<ProductDto>> CreateAsync(CreateProductRequest request, CancellationToken cancellationToken) => throw new NotSupportedException();
        public Task<MutationValidationResult<ProductDto>> UpdateAsync(Guid id, UpdateProductRequest request, CancellationToken cancellationToken) => throw new NotSupportedException();
        public Task<MutationResult<bool>> DeleteAsync(Guid id, string concurrencyToken, CancellationToken cancellationToken) => throw new NotSupportedException();
    }
}
