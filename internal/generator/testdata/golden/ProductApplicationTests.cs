using ProductService.Application.Common;
using ProductService.Application.Features.Products;
using ProductService.Application.Features.Products.Create;
using ProductService.Application.Features.Products.Delete;
using ProductService.Application.Features.Products.GetById;
using ProductService.Application.Features.Products.List;
using ProductService.Application.Features.Products.Update;
using ProductService.Domain.Features.Products;
using DomainProduct = ProductService.Domain.Features.Products.Product;
using ProductService.Domain.Common.ValueObjects;

using ErrorOr;
using Xunit;

namespace ProductService.Application.Tests.Features.Products;

public sealed class ProductApplicationTests
{
    [Fact]
    public async Task CreateCommandHandlerPersistsEntityAndMapsSnapshot()
    {
        var repository = new FakeProductRepository();
        var handler = new CreateProductCommandHandler(repository);

        var result = await handler.Handle(new CreateProductCommand
        {
            IsActive = true,
            Name = "Product Prime",
            Price = 0m,
        }, CancellationToken.None);

        Assert.False(result.IsError);
        Assert.Equal(1, repository.AddCalls);
        var created = result.Value;
        Assert.NotEqual(Guid.Empty, created.Id);
        Assert.True(created.IsActive);
        Assert.Equal("Product Prime", created.Name);
        Assert.Equal(0m, created.Price);
        Assert.Equal("token-v1", created.ConcurrencyToken);
    }

    [Fact]
    public async Task CreateCommandValidatorAcceptsValidCommand()
    {
        var result = await new CreateProductCommandValidator().ValidateAsync(new CreateProductCommand
        {
            IsActive = true,
            Name = "Product Prime",
            Price = 0m,
        });

        Assert.True(result.IsValid);
    }

    [Fact]
    public async Task CreateAndUpdateValidatorsReportDomainInvariantCodes()
    {
        var create = await new CreateProductCommandValidator().ValidateAsync(new CreateProductCommand
        {
            IsActive = true,
            Name = "",
            Price = 0m - 1m,
        });
        var update = await new UpdateProductCommandValidator().ValidateAsync(new UpdateProductCommand
        {
            Id = Guid.NewGuid(),
            IsActive = true,
            Name = "",
            Price = 0m - 1m,
            ConcurrencyToken = "token-v1",
        });

        Assert.Contains(create.Errors, error => error.ErrorCode == "ProductName.Required");
        Assert.Contains(update.Errors, error => error.ErrorCode == "ProductName.Required");
        Assert.Contains(create.Errors, error => error.ErrorCode == "ProductPrice.Minimum");
        Assert.Contains(update.Errors, error => error.ErrorCode == "ProductPrice.Minimum");
    }

    [Fact]
    public async Task UpdateCommandValidatorRequiresConcurrencyToken()
    {
        var result = await new UpdateProductCommandValidator().ValidateAsync(new UpdateProductCommand
        {
            Id = Guid.NewGuid(),
            IsActive = true,
            Name = "Product Prime",
            Price = 0m,
            ConcurrencyToken = "",
        });

        Assert.Contains(result.Errors, error => error.ErrorCode == "ConcurrencyToken.Required");
    }

    [Fact]
    public async Task UpdateCommandHandlerPreservesIdentityValuesAndConcurrencyToken()
    {
        var repository = new FakeProductRepository();
        var entity = DomainProduct.Create(new ProductState { IsActive = true, Name = ProductName.Create("Product Prime").Value!, Price = ProductPrice.Create(0m).Value!,  });
        repository.Items.Add(entity);
        var handler = new UpdateProductCommandHandler(repository);

        var result = await handler.Handle(new UpdateProductCommand
        {
            Id = entity.Id,
            IsActive = false,
            Name = "Product Prime2",
            Price = 999999.99m,
            ConcurrencyToken = "token-v1",
        }, CancellationToken.None);

        Assert.False(result.IsError);
        Assert.Equal(entity.Id, result.Value.Id);
        Assert.True(EqualityComparer<bool>.Default.Equals(false, result.Value.IsActive));
        Assert.True(EqualityComparer<string>.Default.Equals("Product Prime2", result.Value.Name));
        Assert.True(EqualityComparer<decimal>.Default.Equals(999999.99m, result.Value.Price));
        Assert.Equal("token-v2", result.Value.ConcurrencyToken);
    }

    [Fact]
    public async Task UpdateCommandHandlerMapsNotFoundConflictAndInvalidToken()
    {
        var repository = new FakeProductRepository();
        var entity = DomainProduct.Create(new ProductState { IsActive = true, Name = ProductName.Create("Product Prime").Value!, Price = ProductPrice.Create(0m).Value!,  });
        repository.Items.Add(entity);
        var handler = new UpdateProductCommandHandler(repository);

        var missing = await handler.Handle(new UpdateProductCommand { Id = Guid.NewGuid(), IsActive = true, Name = "Product Prime", Price = 0m,  ConcurrencyToken = "token-v1" }, CancellationToken.None);
        var conflict = await handler.Handle(new UpdateProductCommand { Id = entity.Id, IsActive = false, Name = "Product Prime2", Price = 999999.99m,  ConcurrencyToken = "stale-token" }, CancellationToken.None);
        var invalid = await handler.Handle(new UpdateProductCommand { Id = entity.Id, IsActive = false, Name = "Product Prime2", Price = 999999.99m,  ConcurrencyToken = "unknown-token" }, CancellationToken.None);

        Assert.Equal(ErrorType.NotFound, missing.FirstError.Type);
        Assert.Equal(ErrorType.Conflict, conflict.FirstError.Type);
        Assert.Equal(ErrorType.Validation, invalid.FirstError.Type);
        Assert.Equal("ConcurrencyToken.Invalid", invalid.FirstError.Code);
    }

    [Fact]
    public async Task DeleteCommandValidatorRequiresConcurrencyToken()
    {
        var result = await new DeleteProductCommandValidator().ValidateAsync(new DeleteProductCommand(Guid.NewGuid(), ""));

        Assert.Contains(result.Errors, error => error.ErrorCode == "ConcurrencyToken.Required");
    }

    [Fact]
    public async Task DeleteCommandHandlerMapsNotFoundConflictInvalidTokenAndSuccess()
    {
        var repository = new FakeProductRepository();
        var entity = DomainProduct.Create(new ProductState { IsActive = true, Name = ProductName.Create("Product Prime").Value!, Price = ProductPrice.Create(0m).Value!,  });
        repository.Items.Add(entity);
        var handler = new DeleteProductCommandHandler(repository);

        var missing = await handler.Handle(new DeleteProductCommand(Guid.NewGuid(), "token-v1"), CancellationToken.None);
        var conflict = await handler.Handle(new DeleteProductCommand(entity.Id, "stale-token"), CancellationToken.None);
        var invalid = await handler.Handle(new DeleteProductCommand(entity.Id, "unknown-token"), CancellationToken.None);
        var deleted = await handler.Handle(new DeleteProductCommand(entity.Id, "token-v1"), CancellationToken.None);

        Assert.Equal(ErrorType.NotFound, missing.FirstError.Type);
        Assert.Equal(ErrorType.Conflict, conflict.FirstError.Type);
        Assert.Equal(ErrorType.Validation, invalid.FirstError.Type);
        Assert.False(deleted.IsError);
        Assert.Equal(Result.Deleted, deleted.Value);
    }

    [Fact]
    public async Task ListAndGetByIdHandlersMapRepositorySnapshots()
    {
        var repository = new FakeProductRepository();
        var entity = DomainProduct.Create(new ProductState { IsActive = true, Name = ProductName.Create("Product Prime").Value!, Price = ProductPrice.Create(0m).Value!,  });
        repository.Items.Add(entity);

        var page = await new ListProductQueryHandler(repository).Handle(new ListProductQuery(1, 20), CancellationToken.None);
        var item = await new GetProductByIdQueryHandler(repository).Handle(new GetProductByIdQuery(entity.Id), CancellationToken.None);

        Assert.False(page.IsError);
        Assert.Single(page.Value.Items);
        Assert.False(item.IsError);
        Assert.Equal(entity.Id, item.Value.Id);
        Assert.Equal("token-v1", item.Value.ConcurrencyToken);
    }

    private sealed class FakeProductRepository : IProductRepository
    {
        public List<DomainProduct> Items { get; } = [];
        public int AddCalls { get; private set; }
        private string CurrentToken { get; set; } = "token-v1";

        public Task<(IReadOnlyList<EntitySnapshot<DomainProduct>> Items, int TotalCount)> ListAsync(int skip, int take, CancellationToken cancellationToken) =>
            Task.FromResult(((IReadOnlyList<EntitySnapshot<DomainProduct>>)Items.Skip(skip).Take(take).Select(ToSnapshot).ToList(), Items.Count));

        public Task<EntitySnapshot<DomainProduct>?> GetByIdAsync(Guid id, CancellationToken cancellationToken) =>
            Task.FromResult(Items.SingleOrDefault(item => item.Id == id) is { } entity ? ToSnapshot(entity) : null);

        public Task<EntitySnapshot<DomainProduct>> AddAsync(DomainProduct entity, CancellationToken cancellationToken)
        {
            AddCalls++;
            Items.Add(entity);
            return Task.FromResult(ToSnapshot(entity));
        }

        public Task<SaveResultStatus> UpdateAsync(DomainProduct entity, string concurrencyToken, CancellationToken cancellationToken)
        {
            if (concurrencyToken == "stale-token") return Task.FromResult(SaveResultStatus.Conflict);
            if (concurrencyToken != CurrentToken) return Task.FromResult(SaveResultStatus.InvalidToken);
            CurrentToken = "token-v2";
            return Task.FromResult(SaveResultStatus.Saved);
        }

        public Task<SaveResultStatus> DeleteAsync(DomainProduct entity, string concurrencyToken, CancellationToken cancellationToken)
        {
            if (concurrencyToken == "stale-token") return Task.FromResult(SaveResultStatus.Conflict);
            if (concurrencyToken != CurrentToken) return Task.FromResult(SaveResultStatus.InvalidToken);
            Items.Remove(entity);
            return Task.FromResult(SaveResultStatus.Saved);
        }

        private EntitySnapshot<DomainProduct> ToSnapshot(DomainProduct entity) => new(entity, CurrentToken);
    }
}
