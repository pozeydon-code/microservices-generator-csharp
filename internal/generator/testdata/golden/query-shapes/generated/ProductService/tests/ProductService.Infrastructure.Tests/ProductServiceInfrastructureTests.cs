using Microsoft.Data.SqlClient;
using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.Logging.Abstractions;
using ProductService.Infrastructure;
using ProductService.Infrastructure.Health;
using ProductService.Application.Common;
using ProductService.Domain.Entities;
using ProductService.Application.Categories.Interfaces;
using ProductService.Infrastructure.Persistence.Features.Categories;
using DomainCategory = ProductService.Domain.Entities.Category;
using ProductService.Application.Products.Interfaces;
using ProductService.Infrastructure.Persistence.Features.Products;
using DomainProduct = ProductService.Domain.Entities.Product;

using ProductService.Infrastructure.Persistence;
using Xunit;

namespace ProductService.Infrastructure.Tests;

public sealed class ProductServiceInfrastructureTests
{
    [Fact]
    public async Task ReadinessValidatesDatabaseConnectivityAndMappedColumns()
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

        await using (var schemaChangedContext = database.CreateContext())
        {
            await schemaChangedContext.Database.ExecuteSqlRawAsync("ALTER TABLE Categorys DROP COLUMN Name");
            var schemaChanged = await new SqlReadinessProbe(schemaChangedContext, NullLogger<SqlReadinessProbe>.Instance).CheckAsync(CancellationToken.None);
            Assert.Equal(ReadinessStatus.NotReady, schemaChanged.Status);
        }
    }

    [Fact]
    public async Task CategoryRepositoryPersistsReloadsConcurrencyAndConflicts()
    {
        if (!SqlTestDatabase.IsConfigured) { return; }
        var databaseName = $"ProductService_Category_{Guid.NewGuid():N}";
        await using var database = new SqlTestDatabase(databaseName);
        await database.InitializeAsync();

        await using var createContext = database.CreateContext();
        await SeedRequiredRelationshipsForCategoryAsync(createContext);

        var repository = new CategoryRepository(createContext, NullLogger<CategoryRepository>.Instance);
        var unitOfWork = new UnitOfWork(createContext);
        var entity = DomainCategory.Create(new CategoryState { Name = "Name Value", CategoryId = Guid.Parse("00000000-0000-0000-0000-000000000001"),  });
        await repository.AddAsync(entity, CancellationToken.None);
        await unitOfWork.SaveChangesAsync(CancellationToken.None);
        var created = await repository.GetByIdAsync(entity.Id, CancellationToken.None);
        Assert.NotNull(created);
        Assert.False(string.IsNullOrWhiteSpace(created.ConcurrencyToken));
        Assert.True(EqualityComparer<string>.Default.Equals("Name Value", created.Entity.Name));

        await using var reloadContext = database.CreateContext();
        var reloadRepository = new CategoryRepository(reloadContext, NullLogger<CategoryRepository>.Instance);
        var reloadUnitOfWork = new UnitOfWork(reloadContext);
        var reloaded = await reloadRepository.GetByIdAsync(created.Entity.Id, CancellationToken.None);
        Assert.NotNull(reloaded);
        Assert.True(EqualityComparer<string>.Default.Equals("Name Value", reloaded.Entity.Name));

        Assert.Equal(created.ConcurrencyToken, reloaded.ConcurrencyToken);

        reloaded.Entity.Update(new CategoryState { Name = "Updated Name", CategoryId = Guid.Parse("00000000-0000-0000-0000-000000000002"),  });
        Assert.Equal(MutationPreparationStatus.Prepared, await reloadRepository.UpdateAsync(reloaded.Entity, reloaded.ConcurrencyToken, CancellationToken.None));
        await reloadUnitOfWork.SaveChangesAsync(CancellationToken.None);
        var updated = await reloadRepository.GetByIdAsync(created.Entity.Id, CancellationToken.None);
        Assert.NotNull(updated);
        Assert.NotEqual(reloaded.ConcurrencyToken, updated.ConcurrencyToken);
        Assert.True(EqualityComparer<string>.Default.Equals("Updated Name", updated.Entity.Name));

        updated.Entity.Update(new CategoryState { Name = "Name Value", CategoryId = Guid.Parse("00000000-0000-0000-0000-000000000001"),  });
        Assert.Equal(MutationPreparationStatus.Prepared, await reloadRepository.UpdateAsync(updated.Entity, reloaded.ConcurrencyToken, CancellationToken.None));
        await Assert.ThrowsAsync<ConcurrencyConflictException>(() => reloadUnitOfWork.SaveChangesAsync(CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.UpdateAsync(updated.Entity, "not-base64", CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.UpdateAsync(updated.Entity, Convert.ToBase64String([]), CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.UpdateAsync(updated.Entity, Convert.ToBase64String([1]), CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.UpdateAsync(updated.Entity, Convert.ToBase64String(new byte[7]), CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.Prepared, await reloadRepository.UpdateAsync(updated.Entity, Convert.ToBase64String(new byte[8]), CancellationToken.None));
        await Assert.ThrowsAsync<ConcurrencyConflictException>(() => reloadUnitOfWork.SaveChangesAsync(CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.UpdateAsync(updated.Entity, Convert.ToBase64String(new byte[9]), CancellationToken.None));

        Assert.Equal(MutationPreparationStatus.Prepared, await reloadRepository.DeleteAsync(updated.Entity, reloaded.ConcurrencyToken, CancellationToken.None));
        await Assert.ThrowsAsync<ConcurrencyConflictException>(() => reloadUnitOfWork.SaveChangesAsync(CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.DeleteAsync(updated.Entity, "not-base64", CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.DeleteAsync(updated.Entity, Convert.ToBase64String([]), CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.DeleteAsync(updated.Entity, Convert.ToBase64String([1]), CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.DeleteAsync(updated.Entity, Convert.ToBase64String(new byte[7]), CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.Prepared, await reloadRepository.DeleteAsync(updated.Entity, Convert.ToBase64String(new byte[8]), CancellationToken.None));
        await Assert.ThrowsAsync<ConcurrencyConflictException>(() => reloadUnitOfWork.SaveChangesAsync(CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.DeleteAsync(updated.Entity, Convert.ToBase64String(new byte[9]), CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.Prepared, await reloadRepository.DeleteAsync(updated.Entity, updated.ConcurrencyToken, CancellationToken.None));
        await reloadUnitOfWork.SaveChangesAsync(CancellationToken.None);
        Assert.Null(await reloadRepository.GetByIdAsync(created.Entity.Id, CancellationToken.None));
    }

    [Fact]
    public async Task CategoryRepositoryMapsTwoContextConcurrencyRaceToConflict()
    {
        if (!SqlTestDatabase.IsConfigured) { return; }
        var databaseName = $"ProductService_Category_race_{Guid.NewGuid():N}";
        await using var database = new SqlTestDatabase(databaseName);
        await database.InitializeAsync();

        await using var seedContext = database.CreateContext();
        await SeedRequiredRelationshipsForCategoryAsync(seedContext);

        var seedRepository = new CategoryRepository(seedContext, NullLogger<CategoryRepository>.Instance);
        var seedUnitOfWork = new UnitOfWork(seedContext);
        var seed = DomainCategory.Create(new CategoryState { Name = "Name Value", CategoryId = Guid.Parse("00000000-0000-0000-0000-000000000001"),  });
        await seedRepository.AddAsync(seed, CancellationToken.None);
        await seedUnitOfWork.SaveChangesAsync(CancellationToken.None);
        var created = await seedRepository.GetByIdAsync(seed.Id, CancellationToken.None);
        Assert.NotNull(created);

        await using var firstContext = database.CreateContext();
        await using var secondContext = database.CreateContext();
        var firstRepository = new CategoryRepository(firstContext, NullLogger<CategoryRepository>.Instance);
        var secondRepository = new CategoryRepository(secondContext, NullLogger<CategoryRepository>.Instance);
        var firstUnitOfWork = new UnitOfWork(firstContext);
        var secondUnitOfWork = new UnitOfWork(secondContext);
        var first = await firstRepository.GetByIdAsync(created.Entity.Id, CancellationToken.None);
        var second = await secondRepository.GetByIdAsync(created.Entity.Id, CancellationToken.None);
        Assert.NotNull(first);
        Assert.NotNull(second);

        second.Entity.Update(new CategoryState { Name = "Updated Name", CategoryId = Guid.Parse("00000000-0000-0000-0000-000000000002"),  });
        Assert.Equal(MutationPreparationStatus.Prepared, await secondRepository.UpdateAsync(second.Entity, second.ConcurrencyToken, CancellationToken.None));
        await secondUnitOfWork.SaveChangesAsync(CancellationToken.None);
        first.Entity.Update(new CategoryState { Name = "Name Value", CategoryId = Guid.Parse("00000000-0000-0000-0000-000000000001"),  });
        Assert.Equal(MutationPreparationStatus.Prepared, await firstRepository.UpdateAsync(first.Entity, first.ConcurrencyToken, CancellationToken.None));
        await Assert.ThrowsAsync<ConcurrencyConflictException>(() => firstUnitOfWork.SaveChangesAsync(CancellationToken.None));
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
        var unitOfWork = new UnitOfWork(createContext);
        var entity = DomainProduct.Create(new ProductState { Name = "Name Value", Price = 12.34m,  });
        await repository.AddAsync(entity, CancellationToken.None);
        await unitOfWork.SaveChangesAsync(CancellationToken.None);
        var created = await repository.GetByIdAsync(entity.Id, CancellationToken.None);
        Assert.NotNull(created);
        Assert.False(string.IsNullOrWhiteSpace(created.ConcurrencyToken));
        Assert.True(EqualityComparer<string>.Default.Equals("Name Value", created.Entity.Name));
        Assert.True(EqualityComparer<decimal>.Default.Equals(12.34m, created.Entity.Price));

        await using var reloadContext = database.CreateContext();
        var reloadRepository = new ProductRepository(reloadContext, NullLogger<ProductRepository>.Instance);
        var reloadUnitOfWork = new UnitOfWork(reloadContext);
        var reloaded = await reloadRepository.GetByIdAsync(created.Entity.Id, CancellationToken.None);
        Assert.NotNull(reloaded);
        Assert.True(EqualityComparer<string>.Default.Equals("Name Value", reloaded.Entity.Name));
        Assert.True(EqualityComparer<decimal>.Default.Equals(12.34m, reloaded.Entity.Price));

        Assert.Equal(created.ConcurrencyToken, reloaded.ConcurrencyToken);

        reloaded.Entity.Update(new ProductState { Name = "Updated Name", Price = 56.78m,  });
        Assert.Equal(MutationPreparationStatus.Prepared, await reloadRepository.UpdateAsync(reloaded.Entity, reloaded.ConcurrencyToken, CancellationToken.None));
        await reloadUnitOfWork.SaveChangesAsync(CancellationToken.None);
        var updated = await reloadRepository.GetByIdAsync(created.Entity.Id, CancellationToken.None);
        Assert.NotNull(updated);
        Assert.NotEqual(reloaded.ConcurrencyToken, updated.ConcurrencyToken);
        Assert.True(EqualityComparer<string>.Default.Equals("Updated Name", updated.Entity.Name));
        Assert.True(EqualityComparer<decimal>.Default.Equals(56.78m, updated.Entity.Price));

        updated.Entity.Update(new ProductState { Name = "Name Value", Price = 12.34m,  });
        Assert.Equal(MutationPreparationStatus.Prepared, await reloadRepository.UpdateAsync(updated.Entity, reloaded.ConcurrencyToken, CancellationToken.None));
        await Assert.ThrowsAsync<ConcurrencyConflictException>(() => reloadUnitOfWork.SaveChangesAsync(CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.UpdateAsync(updated.Entity, "not-base64", CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.UpdateAsync(updated.Entity, Convert.ToBase64String([]), CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.UpdateAsync(updated.Entity, Convert.ToBase64String([1]), CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.UpdateAsync(updated.Entity, Convert.ToBase64String(new byte[7]), CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.Prepared, await reloadRepository.UpdateAsync(updated.Entity, Convert.ToBase64String(new byte[8]), CancellationToken.None));
        await Assert.ThrowsAsync<ConcurrencyConflictException>(() => reloadUnitOfWork.SaveChangesAsync(CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.UpdateAsync(updated.Entity, Convert.ToBase64String(new byte[9]), CancellationToken.None));

        Assert.Equal(MutationPreparationStatus.Prepared, await reloadRepository.DeleteAsync(updated.Entity, reloaded.ConcurrencyToken, CancellationToken.None));
        await Assert.ThrowsAsync<ConcurrencyConflictException>(() => reloadUnitOfWork.SaveChangesAsync(CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.DeleteAsync(updated.Entity, "not-base64", CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.DeleteAsync(updated.Entity, Convert.ToBase64String([]), CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.DeleteAsync(updated.Entity, Convert.ToBase64String([1]), CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.DeleteAsync(updated.Entity, Convert.ToBase64String(new byte[7]), CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.Prepared, await reloadRepository.DeleteAsync(updated.Entity, Convert.ToBase64String(new byte[8]), CancellationToken.None));
        await Assert.ThrowsAsync<ConcurrencyConflictException>(() => reloadUnitOfWork.SaveChangesAsync(CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.DeleteAsync(updated.Entity, Convert.ToBase64String(new byte[9]), CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.Prepared, await reloadRepository.DeleteAsync(updated.Entity, updated.ConcurrencyToken, CancellationToken.None));
        await reloadUnitOfWork.SaveChangesAsync(CancellationToken.None);
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
        var seedUnitOfWork = new UnitOfWork(seedContext);
        var seed = DomainProduct.Create(new ProductState { Name = "Name Value", Price = 12.34m,  });
        await seedRepository.AddAsync(seed, CancellationToken.None);
        await seedUnitOfWork.SaveChangesAsync(CancellationToken.None);
        var created = await seedRepository.GetByIdAsync(seed.Id, CancellationToken.None);
        Assert.NotNull(created);

        await using var firstContext = database.CreateContext();
        await using var secondContext = database.CreateContext();
        var firstRepository = new ProductRepository(firstContext, NullLogger<ProductRepository>.Instance);
        var secondRepository = new ProductRepository(secondContext, NullLogger<ProductRepository>.Instance);
        var firstUnitOfWork = new UnitOfWork(firstContext);
        var secondUnitOfWork = new UnitOfWork(secondContext);
        var first = await firstRepository.GetByIdAsync(created.Entity.Id, CancellationToken.None);
        var second = await secondRepository.GetByIdAsync(created.Entity.Id, CancellationToken.None);
        Assert.NotNull(first);
        Assert.NotNull(second);

        second.Entity.Update(new ProductState { Name = "Updated Name", Price = 56.78m,  });
        Assert.Equal(MutationPreparationStatus.Prepared, await secondRepository.UpdateAsync(second.Entity, second.ConcurrencyToken, CancellationToken.None));
        await secondUnitOfWork.SaveChangesAsync(CancellationToken.None);
        first.Entity.Update(new ProductState { Name = "Name Value", Price = 12.34m,  });
        Assert.Equal(MutationPreparationStatus.Prepared, await firstRepository.UpdateAsync(first.Entity, first.ConcurrencyToken, CancellationToken.None));
        await Assert.ThrowsAsync<ConcurrencyConflictException>(() => firstUnitOfWork.SaveChangesAsync(CancellationToken.None));
    }



    private static async Task SeedRequiredRelationshipsForCategoryAsync(ProductServiceDbContext context)
    {
        await context.Database.ExecuteSqlRawAsync("INSERT INTO [Products] ([Id], [Name], [Price]) VALUES ({0}, N'Name Value', 12.34)", Guid.Parse("00000000-0000-0000-0000-000000000001"));
        await context.Database.ExecuteSqlRawAsync("INSERT INTO [Products] ([Id], [Name], [Price]) VALUES ({0}, N'Name Value', 12.34)", Guid.Parse("00000000-0000-0000-0000-000000000002"));
        await context.SaveChangesAsync(CancellationToken.None);
    }


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
