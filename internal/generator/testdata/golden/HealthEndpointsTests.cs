using System.Net;
using Microsoft.AspNetCore.TestHost;
using Microsoft.Extensions.DependencyInjection;
using ProductService.Application.Common;
using Xunit;

namespace ProductService.Api.Tests;

public sealed class HealthEndpointsTests
{
    [Fact]
    public async Task HealthLiveIsAnonymous()
    {
        await using var factory = new TestApiFactory();
        using var client = factory.CreateClient();
        Assert.Equal(HttpStatusCode.OK, (await client.GetAsync("/health/live")).StatusCode);
    }

    [Fact]
    public async Task HealthReadyIsAnonymousAndReturnsGenericFailureWhenNotReady()
    {
        await using var factory = new TestApiFactory(builder => builder.ConfigureTestServices(services =>
            services.AddScoped<IReadinessProbe>(_ => new FakeReadinessProbe(ReadinessStatus.NotReady))));
        using var client = factory.CreateClient();
        var response = await client.GetAsync("/health/ready");
        Assert.Equal(HttpStatusCode.ServiceUnavailable, response.StatusCode);
        var body = await response.Content.ReadAsStringAsync();
        Assert.Contains("Service is not ready.", body);
        Assert.DoesNotContain("SQL", body, StringComparison.OrdinalIgnoreCase);
        Assert.DoesNotContain("migration", body, StringComparison.OrdinalIgnoreCase);
        Assert.DoesNotContain("table", body, StringComparison.OrdinalIgnoreCase);
    }

    private sealed class FakeReadinessProbe(ReadinessStatus status) : IReadinessProbe
    {
        public Task<ReadinessResult> CheckAsync(CancellationToken cancellationToken) => Task.FromResult(new ReadinessResult(status));
    }
}
