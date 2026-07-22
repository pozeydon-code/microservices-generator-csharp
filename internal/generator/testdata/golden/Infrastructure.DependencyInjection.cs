using System.Diagnostics;
using System.Globalization;
using Microsoft.Data.SqlClient;
using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Logging;
using ProductService.Application.Common;
using ProductService.Application.Features.Products;
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

        services.AddDbContext<ProductServiceDbContext>(options => options.UseSqlServer(
            connectionBuilder.ConnectionString,
            sql => sql.EnableRetryOnFailure(maxRetryCount: ResiliencePolicy.SqlRetryCount, maxRetryDelay: ResiliencePolicy.SqlRetryDelay, errorNumbersToAdd: null).CommandTimeout(ResiliencePolicy.SqlCommandTimeoutSeconds)));
        services.AddScoped<IProductRepository, ProductRepository>();
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

public sealed class SqlReadinessProbe(ProductServiceDbContext dbContext, ILogger<SqlReadinessProbe> logger) : IReadinessProbe
{
    private static readonly Action<ILogger, Exception?> ReadinessCheckFailed = LoggerMessage.Define(
        LogLevel.Warning,
        new EventId(1, nameof(ReadinessCheckFailed)),
        "Readiness check failed while validating SQL Server connectivity or generated schema.");

    public async Task<ReadinessResult> CheckAsync(CancellationToken cancellationToken)
    {
        using var readinessCts = CancellationTokenSource.CreateLinkedTokenSource(cancellationToken);
        readinessCts.CancelAfter(TimeSpan.FromSeconds(ResiliencePolicy.ReadinessTimeoutSeconds));
        try
        {
            using var activity = DependencyInjection.ActivitySource.StartActivity("sql.readiness");
            return await ExpectedSchemaExistsAsync(readinessCts.Token)
                ? new ReadinessResult(ReadinessStatus.Ready)
                : new ReadinessResult(ReadinessStatus.NotReady);
        }
        catch (Exception ex) when (ex is not OperationCanceledException || !cancellationToken.IsCancellationRequested)
        {
            ReadinessCheckFailed(logger, ex);
            return new ReadinessResult(ReadinessStatus.NotReady);
        }
    }

    private async Task<bool> ExpectedSchemaExistsAsync(CancellationToken cancellationToken)
    {
        var connection = dbContext.Database.GetDbConnection();
        if (connection.State != System.Data.ConnectionState.Open)
        {
            await connection.OpenAsync(cancellationToken);
        }

        await using var command = connection.CreateCommand();
        command.CommandText = @"
SELECT COUNT(*)
FROM (SELECT 'Products' AS table_name, 'Id' AS column_name, 'uniqueidentifier' AS data_type, CAST(0 AS bit) AS is_rowversion, NULL AS character_maximum_length, 'NO' AS is_nullable
UNION ALL
SELECT 'Products' AS table_name, 'IsActive' AS column_name, 'bit' AS data_type, CAST(0 AS bit) AS is_rowversion, NULL AS character_maximum_length, 'NO' AS is_nullable
UNION ALL
SELECT 'Products' AS table_name, 'Name' AS column_name, 'nvarchar' AS data_type, CAST(0 AS bit) AS is_rowversion, 100 AS character_maximum_length, 'NO' AS is_nullable
UNION ALL
SELECT 'Products' AS table_name, 'Price' AS column_name, 'decimal' AS data_type, CAST(0 AS bit) AS is_rowversion, NULL AS character_maximum_length, 'NO' AS is_nullable
UNION ALL
SELECT 'Products', 'RowVersion', 'timestamp', CAST(1 AS bit), NULL, 'YES'
) expected
JOIN INFORMATION_SCHEMA.COLUMNS columns
  ON columns.TABLE_SCHEMA = 'dbo'
 AND columns.TABLE_NAME = expected.table_name
 AND columns.COLUMN_NAME = expected.column_name
  AND columns.DATA_TYPE = expected.data_type
 AND columns.IS_NULLABLE = expected.is_nullable
 AND (expected.character_maximum_length IS NULL OR columns.CHARACTER_MAXIMUM_LENGTH = expected.character_maximum_length)
LEFT JOIN sys.tables tables ON tables.name = expected.table_name
LEFT JOIN sys.schemas schemas ON schemas.schema_id = tables.schema_id AND schemas.name = 'dbo'
LEFT JOIN sys.columns syscolumns ON syscolumns.object_id = tables.object_id AND syscolumns.name = expected.column_name
WHERE expected.is_rowversion = 0 OR syscolumns.system_type_id = 189";
        command.CommandTimeout = ResiliencePolicy.SqlCommandTimeoutSeconds;
        var result = await command.ExecuteScalarAsync(cancellationToken);
        return Convert.ToInt32(result, CultureInfo.InvariantCulture) == 5;
    }
}
