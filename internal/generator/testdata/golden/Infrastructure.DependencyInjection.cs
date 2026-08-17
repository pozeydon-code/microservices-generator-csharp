using System.Diagnostics;
using Microsoft.Data.SqlClient;
using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.DependencyInjection;
using ProductService.Application.Common;
using ProductService.Application.Products.Interfaces;
using ProductService.Infrastructure.Health;
using ProductService.Infrastructure.Persistence;
using ProductService.Infrastructure.Persistence.Features.Products;

namespace ProductService.Infrastructure;

public static class DependencyInjection
{
    public static readonly ActivitySource ActivitySource = new("ProductService.Infrastructure");

    public static IServiceCollection AddInfrastructure(this IServiceCollection services, IConfiguration configuration)
    {
        var connectionString = configuration.GetConnectionString("DefaultConnection");
        if (string.IsNullOrWhiteSpace(connectionString))
        {
            throw new InvalidOperationException("Missing connection string 'ConnectionStrings:DefaultConnection'. Set it with configuration or the ConnectionStrings__DefaultConnection environment variable.");
        }

        var connectionBuilder = new SqlConnectionStringBuilder(connectionString)
        {
            ConnectTimeout = Math.Min(Math.Max(new SqlConnectionStringBuilder(connectionString).ConnectTimeout, 1), ResiliencePolicy.SqlConnectionTimeoutSeconds)
        };

        // SaveChangesAsync uses this bounded execution strategy. A lost response after commit is ambiguous,
        // so an exception or retry result must not be treated as proof that a mutation failed. Mutation
        // idempotency and operation identity belong at the API/Application boundary, not in this adapter.
        services.AddDbContext<ProductServiceDbContext>(options => options.UseSqlServer(
            connectionBuilder.ConnectionString,
            sql => sql.EnableRetryOnFailure(maxRetryCount: ResiliencePolicy.SqlRetryCount, maxRetryDelay: ResiliencePolicy.SqlRetryDelay, errorNumbersToAdd: null).CommandTimeout(ResiliencePolicy.SqlCommandTimeoutSeconds)));
        services.AddScoped<IProductRepository, ProductRepository>();
        services.AddScoped<IUnitOfWork, UnitOfWork>();
        services.AddScoped<IReadinessProbe, SqlReadinessProbe>();
        return services;
    }
}

internal static class ResiliencePolicy
{
    public const int SqlConnectionTimeoutSeconds = 2;
    public const int SqlCommandTimeoutSeconds = 2;
    public const int SqlRetryCount = 1;
    public static readonly TimeSpan SqlRetryDelay = TimeSpan.FromMilliseconds(250);
    public const int ReadinessTimeoutSeconds = 2;
}
