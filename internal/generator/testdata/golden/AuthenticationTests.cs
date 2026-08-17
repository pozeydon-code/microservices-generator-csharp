using System.Net;
using System.Net.Http.Json;
using Microsoft.AspNetCore.TestHost;
using Microsoft.Extensions.DependencyInjection;
using ProductService.Application.Common;
using ProductService.Application.Products.Dtos;
using ProductService.Application.Products.Queries.List;
using ErrorOr;
using MediatR;
using Xunit;

[assembly: CollectionBehavior(DisableTestParallelization = true)]

namespace ProductService.WebApi.Tests;

public sealed class AuthenticationTests
{
    [Fact]
    public async Task ValidBearerGetsExpectedSuccessfulCrudBody()
    {
        await using var factory = new TestWebApiFactory(builder => builder.ConfigureTestServices(services =>
            services.AddScoped<IRequestHandler<ListProductQuery, ErrorOr<PagedResult<ProductDto>>>, AuthListProductQueryHandler>()));
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
        await using var factory = new TestWebApiFactory();
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
        await using var factory = new TestWebApiFactory();
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
        await using var factory = TestWebApiFactory.Production();
        using var client = factory.CreateClient(new Microsoft.AspNetCore.Mvc.Testing.WebApplicationFactoryClientOptions { BaseAddress = new Uri("https://localhost") });
        var response = await client.GetAsync("/health/live");
        Assert.Equal(HttpStatusCode.OK, response.StatusCode);
        Assert.True(response.Headers.Contains("Strict-Transport-Security"));
    }

    [Fact]
    public async Task SlowUseCaseTriggersRequestTimeoutAndCancellation()
    {
        SlowListProductQueryHandler.CancellationObserved = false;
        await using var factory = new TestWebApiFactory(builder => builder.ConfigureTestServices(services =>
            services.AddScoped<IRequestHandler<ListProductQuery, ErrorOr<PagedResult<ProductDto>>>, SlowListProductQueryHandler>()));
        using var client = factory.CreateAuthenticatedClient();
        var started = DateTimeOffset.UtcNow;
        var response = await client.GetAsync("/products");
        var elapsed = DateTimeOffset.UtcNow - started;
        Assert.Equal(HttpStatusCode.ServiceUnavailable, response.StatusCode);
        Assert.True(elapsed < TimeSpan.FromSeconds(20));
        Assert.True(SlowListProductQueryHandler.CancellationObserved);
        var problem = await response.Content.ReadFromJsonAsync<Dictionary<string, object>>();
        Assert.NotNull(problem);
        Assert.True(problem.ContainsKey("status"));
    }

    [Fact]
    public async Task RequestTimeoutBudgetValidationHasExactBoundary()
    {
        await using var rejectedFactory = new TestWebApiFactory(requestTimeoutSeconds: 9);
        var ex = await Assert.ThrowsAsync<InvalidOperationException>(async () =>
        {
            using var client = rejectedFactory.CreateClient();
            await client.GetAsync("/health/live");
        });
        Assert.Contains("at least 10 seconds", ex.ToString());

        await using var acceptedBoundaryFactory = new TestWebApiFactory(requestTimeoutSeconds: 10);
        using var acceptedBoundaryClient = acceptedBoundaryFactory.CreateClient(new Microsoft.AspNetCore.Mvc.Testing.WebApplicationFactoryClientOptions { BaseAddress = new Uri("https://localhost") });
        var boundaryResponse = await acceptedBoundaryClient.GetAsync("/health/live");
        Assert.Equal(HttpStatusCode.OK, boundaryResponse.StatusCode);

        await using var defaultFactory = new TestWebApiFactory();
        using var defaultClient = defaultFactory.CreateClient(new Microsoft.AspNetCore.Mvc.Testing.WebApplicationFactoryClientOptions { BaseAddress = new Uri("https://localhost") });
        var defaultResponse = await defaultClient.GetAsync("/health/live");
        Assert.Equal(HttpStatusCode.OK, defaultResponse.StatusCode);
    }

    private sealed class AuthListProductQueryHandler : IRequestHandler<ListProductQuery, ErrorOr<PagedResult<ProductDto>>>
    {
        private static readonly ProductDto Item = new(
            Id: Guid.Parse("11111111-1111-1111-1111-111111111111"),
            IsActive: true,
            Name: "Product Prime",
            Price: 0m,
            ConcurrencyToken: "token-v1");
        public Task<ErrorOr<PagedResult<ProductDto>>> Handle(ListProductQuery request, CancellationToken cancellationToken) => Task.FromResult<ErrorOr<PagedResult<ProductDto>>>(new PagedResult<ProductDto>([Item], 1, 20, 1));
    }

    private sealed class SlowListProductQueryHandler : IRequestHandler<ListProductQuery, ErrorOr<PagedResult<ProductDto>>>
    {
        public static bool CancellationObserved { get; set; }
        public async Task<ErrorOr<PagedResult<ProductDto>>> Handle(ListProductQuery request, CancellationToken cancellationToken)
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
    }
}
