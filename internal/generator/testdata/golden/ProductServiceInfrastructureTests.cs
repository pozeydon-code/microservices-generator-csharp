using Microsoft.Data.SqlClient;
using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.Logging.Abstractions;
using ProductService.Infrastructure;
using ProductService.Application.Common;
using ProductService.Application.Features.Products;
using ProductService.Domain.Features.Products;
using ProductService.Infrastructure.Persistence.Features.Products;
using DomainProduct = ProductService.Domain.Features.Products.Product;
using ProductService.Domain.Common.ValueObjects;

using ProductService.Infrastructure.Persistence;
using Xunit;

namespace ProductService.Infrastructure.Tests;

public sealed class ProductServiceInfrastructureTests
{
    [Fact]
    public async Task ReadinessReportsSchemaHealthTransitions()
    {
        if (!SqlTestDatabase.IsConfigured) { return; }
        var databaseName = $"ProductService_readiness_{Guid.NewGuid():N}";
        await using var database = new SqlTestDatabase(databaseName);

        await using (var beforeContext = database.CreateContext())
        {
            var before = await new SqlReadinessProbe(beforeContext, NullLogger<SqlReadinessProbe>.Instance).CheckAsync(CancellationToken.None);
            Assert.Equal(ReadinessStatus.NotReady, before.Status);
        }

        await database.InitializeAsync();
        await using (var healthyContext = database.CreateContext())
        {
            var healthy = await new SqlReadinessProbe(healthyContext, NullLogger<SqlReadinessProbe>.Instance).CheckAsync(CancellationToken.None);
            Assert.Equal(ReadinessStatus.Ready, healthy.Status);
        }

        await using (var incompatibleContext = database.CreateContext())
        {
            await incompatibleContext.Database.ExecuteSqlRawAsync("ALTER TABLE Products DROP COLUMN IsActive");
            var incompatible = await new SqlReadinessProbe(incompatibleContext, NullLogger<SqlReadinessProbe>.Instance).CheckAsync(CancellationToken.None);
            Assert.Equal(ReadinessStatus.NotReady, incompatible.Status);
        }
    }

    [Fact]
    public async Task ReadinessRejectsIncorrectStringValueObjectMaxLengthNullabilityAndPreflightFindsViolations()
    {
        if (!SqlTestDatabase.IsConfigured) { return; }
        var databaseName = $"ProductService_readiness_length_{Guid.NewGuid():N}";
        await using var database = new SqlTestDatabase(databaseName);
        await database.InitializeAsync();

        await using var context = database.CreateContext();
        var healthy = await new SqlReadinessProbe(context, NullLogger<SqlReadinessProbe>.Instance).CheckAsync(CancellationToken.None);
        Assert.Equal(ReadinessStatus.Ready, healthy.Status);

        await context.Database.ExecuteSqlRawAsync("ALTER TABLE Products ALTER COLUMN Name nvarchar(101) NOT NULL");
        var unhealthy = await new SqlReadinessProbe(context, NullLogger<SqlReadinessProbe>.Instance).CheckAsync(CancellationToken.None);
        Assert.Equal(ReadinessStatus.NotReady, unhealthy.Status);

        await context.Database.ExecuteSqlRawAsync("INSERT INTO [Products] ([Id], [Name], [IsActive], [Price]) VALUES ({0}, REPLICATE(N'x', 101), 1, 0)", Guid.NewGuid());
        var violatingRows = await context.Database.SqlQueryRaw<int>("SELECT COUNT(*) AS [Value] FROM [Products] WHERE LEN([Name]) > 100").SingleAsync();
        Assert.Equal(1, violatingRows);

        await context.Database.ExecuteSqlRawAsync("DELETE FROM [Products] WHERE LEN([Name]) > 100");
        await context.Database.ExecuteSqlRawAsync("ALTER TABLE Products ALTER COLUMN Name nvarchar(100) NULL");
        var nullable = await new SqlReadinessProbe(context, NullLogger<SqlReadinessProbe>.Instance).CheckAsync(CancellationToken.None);
        Assert.Equal(ReadinessStatus.NotReady, nullable.Status);
    }

    [Fact]
    public async Task ValueObjectPreflightDetectsRepairsAndClearsSqlSafeViolations()
    {
        if (!SqlTestDatabase.IsConfigured) { return; }
        var databaseName = $"ProductService_preflight_{Guid.NewGuid():N}";
        await using var database = new SqlTestDatabase(databaseName);
        await database.InitializeAsync();

        await using var context = database.CreateContext();
        Assert.Empty(await RunPreflightAsync(context));


        var id = Guid.NewGuid();
        await context.Database.ExecuteSqlRawAsync("ALTER TABLE Products ALTER COLUMN Name nvarchar(101) NOT NULL");
        await context.Database.ExecuteSqlRawAsync("INSERT INTO [Products] ([Id], [Name], [IsActive], [Price]) VALUES ({0}, REPLICATE(N'x', 101), 1, 0)", id);

        var violations = await RunPreflightAsync(context);
        var violation = Assert.Single(violations);
        Assert.Equal("Products", violation.TableName);
        Assert.Equal("Name", violation.ColumnName);
        Assert.Equal(id, violation.RecordId);

        await using var transaction = await context.Database.BeginTransactionAsync();
        await context.Database.ExecuteSqlRawAsync("UPDATE [Products] SET [Name] = {0} WHERE [Id] = {1}", "Repaired Value", id);
        await transaction.CommitAsync();

        await context.Database.ExecuteSqlRawAsync("ALTER TABLE Products ALTER COLUMN Name nvarchar(100) NOT NULL");

        Assert.Empty(await RunPreflightAsync(context));
        var healthy = await new SqlReadinessProbe(context, NullLogger<SqlReadinessProbe>.Instance).CheckAsync(CancellationToken.None);
        Assert.Equal(ReadinessStatus.Ready, healthy.Status);
    }



    [Fact]
    public async Task ProductRepositoryPersistsReloadsConcurrencyAndConflicts()
    {
        if (!SqlTestDatabase.IsConfigured) { return; }
        var databaseName = $"ProductService_Product_{Guid.NewGuid():N}";
        await using var database = new SqlTestDatabase(databaseName);
        await database.InitializeAsync();

        await using var createContext = database.CreateContext();
        var repository = new ProductRepository(createContext, NullLogger<ProductRepository>.Instance);
        var created = await repository.AddAsync(DomainProduct.Create(new ProductState { IsActive = true, Name = ProductName.Create("Product Prime").Value!, Price = ProductPrice.Create(0m).Value!,  }), CancellationToken.None);
        Assert.False(string.IsNullOrWhiteSpace(created.ConcurrencyToken));
        Assert.True(EqualityComparer<bool>.Default.Equals(true, created.Entity.IsActive));
        Assert.True(EqualityComparer<string>.Default.Equals("Product Prime", created.Entity.Name.Value));
        Assert.True(EqualityComparer<decimal>.Default.Equals(0m, created.Entity.Price.Value));

        await using var reloadContext = database.CreateContext();
        var reloadRepository = new ProductRepository(reloadContext, NullLogger<ProductRepository>.Instance);
        var reloaded = await reloadRepository.GetByIdAsync(created.Entity.Id, CancellationToken.None);
        Assert.NotNull(reloaded);
        Assert.True(EqualityComparer<bool>.Default.Equals(true, reloaded.Entity.IsActive));
        Assert.True(EqualityComparer<string>.Default.Equals("Product Prime", reloaded.Entity.Name.Value));
        Assert.True(EqualityComparer<decimal>.Default.Equals(0m, reloaded.Entity.Price.Value));

        Assert.Equal(created.ConcurrencyToken, reloaded.ConcurrencyToken);

        reloaded.Entity.Update(new ProductState { IsActive = false, Name = ProductName.Create("Product Prime2").Value!, Price = ProductPrice.Create(999999.99m).Value!,  });
        Assert.Equal(SaveResultStatus.Saved, await reloadRepository.UpdateAsync(reloaded.Entity, reloaded.ConcurrencyToken, CancellationToken.None));
        var updated = await reloadRepository.GetByIdAsync(created.Entity.Id, CancellationToken.None);
        Assert.NotNull(updated);
        Assert.NotEqual(reloaded.ConcurrencyToken, updated.ConcurrencyToken);
        Assert.True(EqualityComparer<bool>.Default.Equals(false, updated.Entity.IsActive));
        Assert.True(EqualityComparer<string>.Default.Equals("Product Prime2", updated.Entity.Name.Value));
        Assert.True(EqualityComparer<decimal>.Default.Equals(999999.99m, updated.Entity.Price.Value));

        updated.Entity.Update(new ProductState { IsActive = true, Name = ProductName.Create("Product Prime").Value!, Price = ProductPrice.Create(0m).Value!,  });
        Assert.Equal(SaveResultStatus.Conflict, await reloadRepository.UpdateAsync(updated.Entity, reloaded.ConcurrencyToken, CancellationToken.None));
        Assert.Equal(SaveResultStatus.InvalidToken, await reloadRepository.UpdateAsync(updated.Entity, "not-base64", CancellationToken.None));
        Assert.Equal(SaveResultStatus.InvalidToken, await reloadRepository.UpdateAsync(updated.Entity, Convert.ToBase64String([]), CancellationToken.None));
        Assert.Equal(SaveResultStatus.InvalidToken, await reloadRepository.UpdateAsync(updated.Entity, Convert.ToBase64String([1]), CancellationToken.None));
        Assert.Equal(SaveResultStatus.InvalidToken, await reloadRepository.UpdateAsync(updated.Entity, Convert.ToBase64String(new byte[7]), CancellationToken.None));
        Assert.Equal(SaveResultStatus.Conflict, await reloadRepository.UpdateAsync(updated.Entity, Convert.ToBase64String(new byte[8]), CancellationToken.None));
        Assert.Equal(SaveResultStatus.InvalidToken, await reloadRepository.UpdateAsync(updated.Entity, Convert.ToBase64String(new byte[9]), CancellationToken.None));

        Assert.Equal(SaveResultStatus.Conflict, await reloadRepository.DeleteAsync(updated.Entity, reloaded.ConcurrencyToken, CancellationToken.None));
        Assert.Equal(SaveResultStatus.InvalidToken, await reloadRepository.DeleteAsync(updated.Entity, "not-base64", CancellationToken.None));
        Assert.Equal(SaveResultStatus.InvalidToken, await reloadRepository.DeleteAsync(updated.Entity, Convert.ToBase64String([]), CancellationToken.None));
        Assert.Equal(SaveResultStatus.InvalidToken, await reloadRepository.DeleteAsync(updated.Entity, Convert.ToBase64String([1]), CancellationToken.None));
        Assert.Equal(SaveResultStatus.InvalidToken, await reloadRepository.DeleteAsync(updated.Entity, Convert.ToBase64String(new byte[7]), CancellationToken.None));
        Assert.Equal(SaveResultStatus.Conflict, await reloadRepository.DeleteAsync(updated.Entity, Convert.ToBase64String(new byte[8]), CancellationToken.None));
        Assert.Equal(SaveResultStatus.InvalidToken, await reloadRepository.DeleteAsync(updated.Entity, Convert.ToBase64String(new byte[9]), CancellationToken.None));
        Assert.Equal(SaveResultStatus.Saved, await reloadRepository.DeleteAsync(updated.Entity, updated.ConcurrencyToken, CancellationToken.None));
        Assert.Null(await reloadRepository.GetByIdAsync(created.Entity.Id, CancellationToken.None));
    }

    [Fact]
    public async Task ProductRepositoryMapsTwoContextConcurrencyRaceToConflict()
    {
        if (!SqlTestDatabase.IsConfigured) { return; }
        var databaseName = $"ProductService_Product_race_{Guid.NewGuid():N}";
        await using var database = new SqlTestDatabase(databaseName);
        await database.InitializeAsync();

        await using var seedContext = database.CreateContext();
        var seedRepository = new ProductRepository(seedContext, NullLogger<ProductRepository>.Instance);
        var created = await seedRepository.AddAsync(DomainProduct.Create(new ProductState { IsActive = true, Name = ProductName.Create("Product Prime").Value!, Price = ProductPrice.Create(0m).Value!,  }), CancellationToken.None);

        await using var firstContext = database.CreateContext();
        await using var secondContext = database.CreateContext();
        var firstRepository = new ProductRepository(firstContext, NullLogger<ProductRepository>.Instance);
        var secondRepository = new ProductRepository(secondContext, NullLogger<ProductRepository>.Instance);
        var first = await firstRepository.GetByIdAsync(created.Entity.Id, CancellationToken.None);
        var second = await secondRepository.GetByIdAsync(created.Entity.Id, CancellationToken.None);
        Assert.NotNull(first);
        Assert.NotNull(second);

        second.Entity.Update(new ProductState { IsActive = false, Name = ProductName.Create("Product Prime2").Value!, Price = ProductPrice.Create(999999.99m).Value!,  });
        Assert.Equal(SaveResultStatus.Saved, await secondRepository.UpdateAsync(second.Entity, second.ConcurrencyToken, CancellationToken.None));
        first.Entity.Update(new ProductState { IsActive = true, Name = ProductName.Create("Product Prime").Value!, Price = ProductPrice.Create(0m).Value!,  });
        Assert.Equal(SaveResultStatus.Conflict, await firstRepository.UpdateAsync(first.Entity, first.ConcurrencyToken, CancellationToken.None));
    }

    [Fact]
    public async Task ProductRepositoryRejectsCorruptPersistedNameValue()
    {
        if (!SqlTestDatabase.IsConfigured) { return; }
        var databaseName = $"ProductService_Product_corrupt_Name_{Guid.NewGuid():N}";
        await using var database = new SqlTestDatabase(databaseName);
        await database.InitializeAsync();

        var id = Guid.NewGuid();
        await using (var corruptContext = database.CreateContext())
        {
            await corruptContext.Database.ExecuteSqlRawAsync("ALTER TABLE Products ALTER COLUMN Name nvarchar(max) NOT NULL");

            await corruptContext.Database.ExecuteSqlRawAsync("INSERT INTO [Products] ([Id], [IsActive], [Name], [Price]) VALUES ({0}, 1, N'', 0)", id);
        }

        await using var reloadContext = database.CreateContext();
        var logger = new CaptureLogger<ProductRepository>();
        var repository = new ProductRepository(reloadContext, logger);
        var exception = await Assert.ThrowsAsync<ProductService.Domain.Common.DomainReconstitutionException>(() => repository.GetByIdAsync(id, CancellationToken.None));
        Assert.Equal("ProductName", exception.ValueObjectType);
        Assert.Contains("ProductName.Required", exception.InvariantCodes);
        Assert.Contains("Product", logger.LastMessage);
        Assert.Contains("Name", logger.LastMessage);
        Assert.Contains("ProductName.Required", logger.LastMessage);
        Assert.Contains(id.ToString(), logger.LastMessage);
    }

    [Fact]
    public async Task ProductRepositoryRejectsCorruptPersistedPriceValue()
    {
        if (!SqlTestDatabase.IsConfigured) { return; }
        var databaseName = $"ProductService_Product_corrupt_Price_{Guid.NewGuid():N}";
        await using var database = new SqlTestDatabase(databaseName);
        await database.InitializeAsync();

        var id = Guid.NewGuid();
        await using (var corruptContext = database.CreateContext())
        {

            await corruptContext.Database.ExecuteSqlRawAsync("INSERT INTO [Products] ([Id], [IsActive], [Name], [Price]) VALUES ({0}, 1, N'Product Prime', 0 - 1)", id);
        }

        await using var reloadContext = database.CreateContext();
        var logger = new CaptureLogger<ProductRepository>();
        var repository = new ProductRepository(reloadContext, logger);
        var exception = await Assert.ThrowsAsync<ProductService.Domain.Common.DomainReconstitutionException>(() => repository.GetByIdAsync(id, CancellationToken.None));
        Assert.Equal("ProductPrice", exception.ValueObjectType);
        Assert.Contains("ProductPrice.Minimum", exception.InvariantCodes);
        Assert.Contains("Product", logger.LastMessage);
        Assert.Contains("Price", logger.LastMessage);
        Assert.Contains("ProductPrice.Minimum", logger.LastMessage);
        Assert.Contains(id.ToString(), logger.LastMessage);
    }


    private static async Task<IReadOnlyList<PreflightViolation>> RunPreflightAsync(ProductServiceDbContext context)
    {
        var infrastructureDirectory = Path.GetDirectoryName(typeof(ProductServiceDbContext).Assembly.Location)!;
        var scriptPath = Path.Combine(infrastructureDirectory, "ValueObjectPreflight.sql");
        if (!File.Exists(scriptPath))
        {
            scriptPath = Path.GetFullPath(Path.Combine(AppContext.BaseDirectory, "../../../../../../src/ProductService/ProductService.Infrastructure/Persistence/ValueObjectPreflight.sql"));
        }
        var script = await File.ReadAllTextAsync(scriptPath);
        var connection = (SqlConnection)context.Database.GetDbConnection();
        if (connection.State != System.Data.ConnectionState.Open)
        {
            await connection.OpenAsync();
        }
        await using var command = connection.CreateCommand();
        command.CommandText = script;
        command.CommandTimeout = 5;
        await using var reader = await command.ExecuteReaderAsync();
        var violations = new List<PreflightViolation>();
        while (await reader.ReadAsync())
        {
            violations.Add(new PreflightViolation(reader.GetString(0), reader.GetString(1), reader.GetString(2), reader.GetGuid(3)));
        }
        return violations;
    }

    private sealed record PreflightViolation(string TableName, string ColumnName, string RuleCode, Guid RecordId);

    private sealed class CaptureLogger<T> : Microsoft.Extensions.Logging.ILogger<T>
    {
        public string LastMessage { get; private set; } = string.Empty;
        public IDisposable? BeginScope<TState>(TState state) where TState : notnull => null;
        public bool IsEnabled(Microsoft.Extensions.Logging.LogLevel logLevel) => true;
        public void Log<TState>(Microsoft.Extensions.Logging.LogLevel logLevel, Microsoft.Extensions.Logging.EventId eventId, TState state, Exception? exception, Func<TState, Exception?, string> formatter) => LastMessage = formatter(state, exception);
    }

    private sealed class SqlTestDatabase(string databaseName) : IAsyncDisposable
    {
        public static bool IsConfigured => !string.IsNullOrWhiteSpace(Environment.GetEnvironmentVariable("MICROGEN_TEST_SQLSERVER"));

        private readonly string connectionString = BuildConnectionString(databaseName);

        public async Task InitializeAsync()
        {
            await using var context = CreateContext();
            await context.Database.EnsureDeletedAsync();
            await context.Database.EnsureCreatedAsync();
        }

        public ProductServiceDbContext CreateContext()
        {
            var options = new DbContextOptionsBuilder<ProductServiceDbContext>()
                .UseSqlServer(connectionString, sql => sql.CommandTimeout(5))
                .Options;
            return new ProductServiceDbContext(options);
        }

        public async ValueTask DisposeAsync()
        {
            await using var context = CreateContext();
            await context.Database.EnsureDeletedAsync();
        }

        private static string BuildConnectionString(string databaseName)
        {
            var server = Environment.GetEnvironmentVariable("MICROGEN_TEST_SQLSERVER");
            if (string.IsNullOrWhiteSpace(server))
            {
                throw new InvalidOperationException("Set MICROGEN_TEST_SQLSERVER to run generated Infrastructure SQL tests.");
            }
            var builder = new SqlConnectionStringBuilder(server) { InitialCatalog = databaseName, ConnectTimeout = 5, TrustServerCertificate = true };
            return builder.ConnectionString;
        }
    }
}
