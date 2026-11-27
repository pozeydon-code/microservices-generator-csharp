using ProductService.Application.Common;
using ProductService.Application.Products.Commands.Create;
using ProductService.Application.Products.Commands.Delete;
using ProductService.Application.Products.Commands.Update;
using ProductService.Application.Products.Dtos;
using ProductService.Application.Products.Interfaces;
using ProductService.Application.Products.Queries.GetById;
using ProductService.Application.Products.Queries.List;
using ProductService.Domain.Entities;
using DomainProduct = ProductService.Domain.Entities.Product;

using FluentValidation;
using ErrorOr;
using Xunit;

namespace ProductService.Application.Tests.Features.Products;

public sealed class ProductApplicationTests
{
    [Fact]
    public async Task CreateCommandHandlerPersistsEntityAndMapsSnapshot()
    {
        var repository = new FakeProductRepository();
        var unitOfWork = new FakeUnitOfWork(repository);
        var handler = new CreateProductCommandHandler(repository, unitOfWork);

        var result = await handler.Handle(new CreateProductCommand
        {
            Name = "Name Value",
            Price = 12.34m,
        }, CancellationToken.None);

        Assert.False(result.IsError);
        Assert.Equal(1, repository.AddCalls);
        Assert.Equal(1, unitOfWork.SaveCalls);
        var created = result.Value;
        Assert.NotEqual(Guid.Empty, created.Id);
        Assert.Equal("Name Value", created.Name);
        Assert.Equal(12.34m, created.Price);
        Assert.Equal("token-v1", created.ConcurrencyToken);
    }

    [Fact]
    public async Task CreateCommandValidatorAcceptsValidCommand()
    {
        var result = await new CreateProductCommandValidator().ValidateAsync(new CreateProductCommand
        {
            Name = "Name Value",
            Price = 12.34m,
        });

        Assert.True(result.IsValid);
    }

    [Fact]
    public async Task ValidationBehaviorCallsHandlerWhenNoValidatorsAreRegistered()
    {
        var behavior = new ValidationBehavior<CreateProductCommand, ErrorOr<ProductDto>>([]);

        var result = await behavior.Handle(new CreateProductCommand(), _ =>
            Task.FromResult<ErrorOr<ProductDto>>(new ProductDto(Guid.NewGuid(), "Name Value", 12.34m, "token-v1")), CancellationToken.None);

        Assert.False(result.IsError);
    }

    [Fact]
    public async Task UpdateCommandValidatorRequiresConcurrencyToken()
    {
        var result = await new UpdateProductCommandValidator().ValidateAsync(new UpdateProductCommand
        {
            Id = Guid.NewGuid(),
            Name = "Name Value",
            Price = 12.34m,
            ConcurrencyToken = "",
        });

        Assert.Contains(result.Errors, error => error.ErrorCode == "ConcurrencyToken.Required");
    }

    [Fact]
    public async Task UpdateCommandHandlerPreservesIdentityValuesAndConcurrencyToken()
    {
        var repository = new FakeProductRepository();
        var entity = DomainProduct.Create(new ProductState { Name = "Name Value", Price = 12.34m,  });
        repository.Items.Add(entity);
        var unitOfWork = new FakeUnitOfWork(repository);
        var handler = new UpdateProductCommandHandler(repository, unitOfWork);

        var result = await handler.Handle(new UpdateProductCommand
        {
            Id = entity.Id,
            Name = "Updated Name",
            Price = 56.78m,
            ConcurrencyToken = "token-v1",
        }, CancellationToken.None);

        Assert.False(result.IsError);
        Assert.Equal(1, unitOfWork.SaveCalls);
        Assert.Equal(entity.Id, result.Value.Id);
        Assert.True(EqualityComparer<string>.Default.Equals("Updated Name", result.Value.Name));
        Assert.True(EqualityComparer<decimal>.Default.Equals(56.78m, result.Value.Price));
        Assert.Equal("token-v2", result.Value.ConcurrencyToken);
    }

    [Fact]
    public async Task UpdateCommandHandlerMapsNotFoundConflictAndInvalidToken()
    {
        var repository = new FakeProductRepository();
        var entity = DomainProduct.Create(new ProductState { Name = "Name Value", Price = 12.34m,  });
        repository.Items.Add(entity);
        var unitOfWork = new FakeUnitOfWork(repository);
        var handler = new UpdateProductCommandHandler(repository, unitOfWork);

        var missing = await handler.Handle(new UpdateProductCommand { Id = Guid.NewGuid(), Name = "Name Value", Price = 12.34m,  ConcurrencyToken = "token-v1" }, CancellationToken.None);
        unitOfWork.ConflictOnNextSave = true;
        var conflict = await handler.Handle(new UpdateProductCommand { Id = entity.Id, Name = "Updated Name", Price = 56.78m,  ConcurrencyToken = "token-v1" }, CancellationToken.None);
        var invalid = await handler.Handle(new UpdateProductCommand { Id = entity.Id, Name = "Updated Name", Price = 56.78m,  ConcurrencyToken = "unknown-token" }, CancellationToken.None);

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
        var entity = DomainProduct.Create(new ProductState { Name = "Name Value", Price = 12.34m,  });
        repository.Items.Add(entity);
        var unitOfWork = new FakeUnitOfWork(repository);
        var handler = new DeleteProductCommandHandler(repository, unitOfWork);

        var missing = await handler.Handle(new DeleteProductCommand(Guid.NewGuid(), "token-v1"), CancellationToken.None);
        var missingMalformed = await handler.Handle(new DeleteProductCommand(Guid.NewGuid(), "unknown-token"), CancellationToken.None);
        unitOfWork.ConflictOnNextSave = true;
        var conflict = await handler.Handle(new DeleteProductCommand(entity.Id, "token-v1"), CancellationToken.None);
        var invalid = await handler.Handle(new DeleteProductCommand(entity.Id, "unknown-token"), CancellationToken.None);
        var deleted = await handler.Handle(new DeleteProductCommand(entity.Id, "token-v1"), CancellationToken.None);

        Assert.Equal(ErrorType.NotFound, missing.FirstError.Type);
        Assert.Equal(ErrorType.NotFound, missingMalformed.FirstError.Type);
        Assert.Equal("Product.NotFound", missingMalformed.FirstError.Code);
        Assert.Equal(ErrorType.Conflict, conflict.FirstError.Type);
        Assert.Equal(ErrorType.Validation, invalid.FirstError.Type);
        Assert.False(deleted.IsError);
        Assert.Equal(Result.Deleted, deleted.Value);
    }

    [Fact]
    public async Task ListAndGetByIdHandlersMapRepositorySnapshots()
    {
        var repository = new FakeProductRepository();
        var entity = DomainProduct.Create(new ProductState { Name = "Name Value", Price = 12.34m,  });
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
        private bool PendingTokenUpdate { get; set; }
        private DomainProduct? PendingDelete { get; set; }

        public Task<(IReadOnlyList<EntitySnapshot<DomainProduct>> Items, int TotalCount)> ListAsync(int skip, int take, CancellationToken cancellationToken) =>
            Task.FromResult(((IReadOnlyList<EntitySnapshot<DomainProduct>>)Items.Skip(skip).Take(take).Select(ToSnapshot).ToList(), Items.Count));

        public Task<EntitySnapshot<DomainProduct>?> GetByIdAsync(Guid id, CancellationToken cancellationToken) =>
            Task.FromResult(Items.SingleOrDefault(item => item.Id == id) is { } entity ? ToSnapshot(entity) : null);

        public Task AddAsync(DomainProduct entity, CancellationToken cancellationToken)
        {
            AddCalls++;
            Items.Add(entity);
            return Task.CompletedTask;
        }

        public Task<MutationPreparationStatus> UpdateAsync(DomainProduct entity, string concurrencyToken, CancellationToken cancellationToken)
        {
            if (concurrencyToken != CurrentToken) return Task.FromResult(MutationPreparationStatus.InvalidToken);
            PendingTokenUpdate = true;
            return Task.FromResult(MutationPreparationStatus.Prepared);
        }

        public Task<MutationPreparationStatus> DeleteAsync(DomainProduct entity, string concurrencyToken, CancellationToken cancellationToken)
        {
            if (concurrencyToken != CurrentToken) return Task.FromResult(MutationPreparationStatus.InvalidToken);
            PendingDelete = entity;
            return Task.FromResult(MutationPreparationStatus.Prepared);
        }

        public void Commit()
        {
            if (PendingDelete is not null)
            {
                Items.Remove(PendingDelete);
                PendingDelete = null;
            }
            if (PendingTokenUpdate)
            {
                CurrentToken = "token-v2";
                PendingTokenUpdate = false;
            }
        }

        private EntitySnapshot<DomainProduct> ToSnapshot(DomainProduct entity) => new(entity, CurrentToken);
    }

    private sealed class FakeUnitOfWork(FakeProductRepository repository) : IUnitOfWork
    {
        public int SaveCalls { get; private set; }
        public bool ConflictOnNextSave { get; set; }

        public Task<int> SaveChangesAsync(CancellationToken cancellationToken)
        {
            SaveCalls++;
            if (ConflictOnNextSave)
            {
                ConflictOnNextSave = false;
                throw new ConcurrencyConflictException();
            }

            repository.Commit();
            return Task.FromResult(1);
        }
    }
}
