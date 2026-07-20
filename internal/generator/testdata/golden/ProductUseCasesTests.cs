using ProductService.Application.Common;
using ProductService.Application.Features.Products;
using ProductService.Domain.Features.Products;
using DomainProduct = ProductService.Domain.Features.Products.Product;
using ProductService.Domain.Shared.ValueObjects;

using Xunit;

namespace ProductService.Application.Tests.Features.Products;

public sealed class ProductUseCasesTests
{
    [Fact]
    public async Task CreateAssignsIdentityAndPreservesRequestValues()
    {
        var repository = new FakeProductRepository();
        var useCases = new ProductUseCases(repository);

        var result = await useCases.CreateAsync(new() { IsActive = true, Name = "Product Prime", Price = 0m,  }, CancellationToken.None);
        var created = result.Value!;

        Assert.Equal(MutationResultStatus.Created, result.Status);
        Assert.NotEqual(Guid.Empty, created.Id);
        Assert.True(created.IsActive);
        Assert.Equal("Product Prime", created.Name);
        Assert.Equal(0m, created.Price);

        Assert.NotNull(created.ConcurrencyToken);
    }

    [Fact]
    public async Task ListAppliesPaginationMetadata()
    {
        var repository = new FakeProductRepository();
        repository.Items.Add(DomainProduct.Create(new ProductState { IsActive = true, Name = ProductName.Create("Product Prime").Value!, Price = ProductPrice.Create(0m).Value!,  }));
        var useCases = new ProductUseCases(repository);

        var page = await useCases.ListAsync(new PageRequest(1, 20), CancellationToken.None);

        Assert.Single(page.Items);
        Assert.Equal(1, page.Page);
        Assert.Equal(20, page.PageSize);
        Assert.Equal(1, page.TotalCount);
    }

    [Fact]
    public async Task CreateAggregatesValidationErrorsAndDoesNotCallRepository()
    {
        var repository = new FakeProductRepository();
        var useCases = new ProductUseCases(repository);

        var result = await useCases.CreateAsync(new() { IsActive = true, Name = "", Price = 0m - 1m,  }, CancellationToken.None);

        Assert.Equal(MutationResultStatus.ValidationFailed, result.Status);
        Assert.Null(result.Value);
        Assert.Equal(0, repository.AddCalls);
        Assert.Collection(result.Validation!.Issues,
            issue =>
            {
                Assert.Equal("name", issue.Field);
                Assert.Equal("ProductName.Required", issue.Code);
                Assert.Equal("ProductName is required.", issue.Message);
            },
            issue =>
            {
                Assert.Equal("price", issue.Field);
                Assert.Equal("ProductPrice.Minimum", issue.Code);
                Assert.Equal("ProductPrice must be greater than or equal to 0m.", issue.Message);
            });
    }

    [Fact]
    public async Task UpdateAggregatesValidationErrorsAndDoesNotCallRepository()
    {
        var repository = new FakeProductRepository();
        var entity = DomainProduct.Create(new ProductState { IsActive = true, Name = ProductName.Create("Product Prime").Value!, Price = ProductPrice.Create(0m).Value!,  });
        repository.Items.Add(entity);
        var useCases = new ProductUseCases(repository);

        var result = await useCases.UpdateAsync(entity.Id, new() { IsActive = true, Name = "", Price = 0m - 1m, ConcurrencyToken = "token-v1" }, CancellationToken.None);

        Assert.Equal(MutationResultStatus.ValidationFailed, result.Status);
        Assert.Null(result.Value);
        Assert.Equal(0, repository.GetCalls + repository.AddCalls + repository.UpdateCalls + repository.DeleteCalls);
        Assert.Collection(result.Validation!.Issues,
            issue =>
            {
                Assert.Equal("name", issue.Field);
                Assert.Equal("ProductName.Required", issue.Code);
                Assert.Equal("ProductName is required.", issue.Message);
            },
            issue =>
            {
                Assert.Equal("price", issue.Field);
                Assert.Equal("ProductPrice.Minimum", issue.Code);
                Assert.Equal("ProductPrice must be greater than or equal to 0m.", issue.Message);
            });
    }

    [Fact]
    public async Task ListAcceptsMinimumPaginationAndUsesZeroOffset()
    {
        var repository = new FakeProductRepository();
        repository.Items.Add(DomainProduct.Create(new ProductState { IsActive = true, Name = ProductName.Create("Product Prime").Value!, Price = ProductPrice.Create(0m).Value!,  }));
        var useCases = new ProductUseCases(repository);

        var page = await useCases.ListAsync(new PageRequest(1, 1), CancellationToken.None);

        Assert.Single(page.Items);
        Assert.Equal(0, repository.LastSkip);
        Assert.Equal(1, repository.LastTake);
    }

    [Fact]
    public void PaginationAcceptsMinimumMaximumSafeAndIntMaxValueWhenOffsetIsSupported()
    {
        Assert.Equal((1, 1, 0), PaginationPolicy.Normalize(1, 1));
        Assert.Equal((21474837, 100, 2147483600), PaginationPolicy.Normalize(21474837, 100));
        Assert.Equal((int.MaxValue, 1, int.MaxValue - 1), PaginationPolicy.Normalize(int.MaxValue, 1));
    }

    [Fact]
    public void PaginationRejectsInvalidBoundsAndFirstUnsupportedOffset()
    {
        Assert.Throws<ArgumentOutOfRangeException>(() => PaginationPolicy.Normalize(0, 20));
        Assert.Throws<ArgumentOutOfRangeException>(() => PaginationPolicy.Normalize(1, 101));
        Assert.Throws<ArgumentOutOfRangeException>(() => PaginationPolicy.Normalize(21474838, 100));
        Assert.Throws<ArgumentOutOfRangeException>(() => PaginationPolicy.Normalize(int.MaxValue, 100));
    }

    [Fact]
    public async Task GetReturnsNullForMissingEntity()
    {
        var useCases = new ProductUseCases(new FakeProductRepository());

        var found = await useCases.GetByIdAsync(Guid.NewGuid(), CancellationToken.None);

        Assert.Null(found);
    }

    [Fact]
    public async Task UpdateProtectsIdentityAndDetectsConcurrencyConflict()
    {
        var repository = new FakeProductRepository();
        var entity = DomainProduct.Create(new ProductState { IsActive = true, Name = ProductName.Create("Product Prime").Value!, Price = ProductPrice.Create(0m).Value!,  });
        repository.Items.Add(entity);
        var useCases = new ProductUseCases(repository);

        var staleToken = "stale-token";
        var validToken = "token-v1";
        var refreshedToken = "token-v2";
        var conflict = await useCases.UpdateAsync(entity.Id, new() { IsActive = false, Name = "Product Prime2", Price = 999999.99m, ConcurrencyToken = staleToken }, CancellationToken.None);
        var invalid = await useCases.UpdateAsync(entity.Id, new() { IsActive = false, Name = "Product Prime2", Price = 999999.99m, ConcurrencyToken = "unknown-token" }, CancellationToken.None);
        var empty = await useCases.UpdateAsync(entity.Id, new() { IsActive = false, Name = "Product Prime2", Price = 999999.99m, ConcurrencyToken = "" }, CancellationToken.None);
        var updated = await useCases.UpdateAsync(entity.Id, new() { IsActive = false, Name = "Product Prime2", Price = 999999.99m, ConcurrencyToken = validToken }, CancellationToken.None);

        Assert.Equal(MutationResultStatus.Conflict, conflict.Status);
        Assert.Equal(MutationResultStatus.InvalidToken, invalid.Status);
        Assert.Equal(MutationResultStatus.InvalidToken, empty.Status);
        Assert.Equal(MutationResultStatus.Updated, updated.Status);
        Assert.Equal(entity.Id, updated.Value!.Id);
        Assert.True(EqualityComparer<bool>.Default.Equals(false, updated.Value.IsActive));
        Assert.True(EqualityComparer<string>.Default.Equals("Product Prime2", updated.Value.Name));
        Assert.True(EqualityComparer<decimal>.Default.Equals(999999.99m, updated.Value.Price));
        Assert.Equal(refreshedToken, updated.Value.ConcurrencyToken);
    }

    [Fact]
    public async Task DeleteReturnsNotFoundAndConflictPredictably()
    {
        var repository = new FakeProductRepository();
        var entity = DomainProduct.Create(new ProductState { IsActive = true, Name = ProductName.Create("Product Prime").Value!, Price = ProductPrice.Create(0m).Value!,  });
        repository.Items.Add(entity);
        var useCases = new ProductUseCases(repository);

        var validToken = "token-v1";
        var staleToken = "stale-token";
        var missing = await useCases.DeleteAsync(Guid.NewGuid(), validToken, CancellationToken.None);
        var conflict = await useCases.DeleteAsync(entity.Id, staleToken, CancellationToken.None);
        var invalid = await useCases.DeleteAsync(entity.Id, "unknown-token", CancellationToken.None);
        var empty = await useCases.DeleteAsync(entity.Id, "", CancellationToken.None);
        var deleted = await useCases.DeleteAsync(entity.Id, validToken, CancellationToken.None);

        Assert.Equal(MutationResultStatus.NotFound, missing.Status);
        Assert.Equal(MutationResultStatus.Conflict, conflict.Status);
        Assert.Equal(MutationResultStatus.InvalidToken, invalid.Status);
        Assert.Equal(MutationResultStatus.InvalidToken, empty.Status);
        Assert.Equal(MutationResultStatus.Deleted, deleted.Status);
    }

    private sealed class FakeProductRepository : IProductRepository
    {
        public List<DomainProduct> Items { get; } = [];
        public int LastSkip { get; private set; }
        public int LastTake { get; private set; }
        public int GetCalls { get; private set; }
        public int AddCalls { get; private set; }
        public int UpdateCalls { get; private set; }
        public int DeleteCalls { get; private set; }

        public Task<(IReadOnlyList<EntitySnapshot<DomainProduct>> Items, int TotalCount)> ListAsync(int skip, int take, CancellationToken cancellationToken)
        {
            LastSkip = skip;
            LastTake = take;
            return Task.FromResult(((IReadOnlyList<EntitySnapshot<DomainProduct>>)Items.Skip(skip).Take(take).Select(ToSnapshot).ToList(), Items.Count));
        }

        public Task<EntitySnapshot<DomainProduct>?> GetByIdAsync(Guid id, CancellationToken cancellationToken)
        {
            GetCalls++;
            var entity = Items.SingleOrDefault(item => item.Id == id);
            return Task.FromResult(entity is null ? null : ToSnapshot(entity));
        }

        public Task<EntitySnapshot<DomainProduct>> AddAsync(DomainProduct entity, CancellationToken cancellationToken)
        {
            AddCalls++;
            Items.Add(entity);
            return Task.FromResult(ToSnapshot(entity));
        }

        public Task<SaveResultStatus> UpdateAsync(DomainProduct entity, string concurrencyToken, CancellationToken cancellationToken)
        {
            UpdateCalls++;
            if (concurrencyToken == "stale-token")
            {
                return Task.FromResult(SaveResultStatus.Conflict);
            }
            if (concurrencyToken != CurrentToken)
            {
                return Task.FromResult(SaveResultStatus.InvalidToken);
            }
            CurrentToken = "token-v2";
            return Task.FromResult(SaveResultStatus.Saved);
        }

        public Task<SaveResultStatus> DeleteAsync(DomainProduct entity, string concurrencyToken, CancellationToken cancellationToken)
        {
            DeleteCalls++;
            if (concurrencyToken == "stale-token")
            {
                return Task.FromResult(SaveResultStatus.Conflict);
            }
            if (concurrencyToken != CurrentToken)
            {
                return Task.FromResult(SaveResultStatus.InvalidToken);
            }
            Items.Remove(entity);
            return Task.FromResult(SaveResultStatus.Saved);
        }

        private string CurrentToken { get; set; } = "token-v1";
        private EntitySnapshot<DomainProduct> ToSnapshot(DomainProduct entity) => new(entity, CurrentToken);
    }
}
