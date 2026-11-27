using ProductService.Application.Common;
using ProductService.Application.Categories.Commands.Create;
using ProductService.Application.Categories.Commands.Delete;
using ProductService.Application.Categories.Commands.Update;
using ProductService.Application.Categories.Dtos;
using ProductService.Application.Categories.Interfaces;
using ProductService.Application.Categories.Queries.GetById;
using ProductService.Application.Categories.Queries.List;
using ProductService.Domain.Entities;
using DomainCategory = ProductService.Domain.Entities.Category;

using FluentValidation;
using ErrorOr;
using Xunit;

namespace ProductService.Application.Tests.Features.Categories;

public sealed class CategoryApplicationTests
{
    [Fact]
    public async Task CreateCommandHandlerPersistsEntityAndMapsSnapshot()
    {
        var repository = new FakeCategoryRepository();
        var unitOfWork = new FakeUnitOfWork(repository);
        var handler = new CreateCategoryCommandHandler(repository, unitOfWork);

        var result = await handler.Handle(new CreateCategoryCommand
        {
            Name = "Name Value",
            CategoryId = Guid.Parse("00000000-0000-0000-0000-000000000001"),
        }, CancellationToken.None);

        Assert.False(result.IsError);
        Assert.Equal(1, repository.AddCalls);
        Assert.Equal(1, unitOfWork.SaveCalls);
        var created = result.Value;
        Assert.NotEqual(Guid.Empty, created.Id);
        Assert.Equal("Name Value", created.Name);
        Assert.Equal(Guid.Parse("00000000-0000-0000-0000-000000000001"), created.CategoryId);
        Assert.Equal("token-v1", created.ConcurrencyToken);
    }

    [Fact]
    public async Task CreateCommandValidatorAcceptsValidCommand()
    {
        var result = await new CreateCategoryCommandValidator().ValidateAsync(new CreateCategoryCommand
        {
            Name = "Name Value",
            CategoryId = Guid.Parse("00000000-0000-0000-0000-000000000001"),
        });

        Assert.True(result.IsValid);
    }

    [Fact]
    public async Task ValidationBehaviorCallsHandlerWhenNoValidatorsAreRegistered()
    {
        var behavior = new ValidationBehavior<CreateCategoryCommand, ErrorOr<CategoryDto>>([]);

        var result = await behavior.Handle(new CreateCategoryCommand(), _ =>
            Task.FromResult<ErrorOr<CategoryDto>>(new CategoryDto(Guid.NewGuid(), "Name Value", Guid.Parse("00000000-0000-0000-0000-000000000001"), "token-v1")), CancellationToken.None);

        Assert.False(result.IsError);
    }

    [Fact]
    public async Task UpdateCommandValidatorRequiresConcurrencyToken()
    {
        var result = await new UpdateCategoryCommandValidator().ValidateAsync(new UpdateCategoryCommand
        {
            Id = Guid.NewGuid(),
            Name = "Name Value",
            CategoryId = Guid.Parse("00000000-0000-0000-0000-000000000001"),
            ConcurrencyToken = "",
        });

        Assert.Contains(result.Errors, error => error.ErrorCode == "ConcurrencyToken.Required");
    }

    [Fact]
    public async Task UpdateCommandHandlerPreservesIdentityValuesAndConcurrencyToken()
    {
        var repository = new FakeCategoryRepository();
        var entity = DomainCategory.Create(new CategoryState { Name = "Name Value", CategoryId = Guid.Parse("00000000-0000-0000-0000-000000000001"),  });
        repository.Items.Add(entity);
        var unitOfWork = new FakeUnitOfWork(repository);
        var handler = new UpdateCategoryCommandHandler(repository, unitOfWork);

        var result = await handler.Handle(new UpdateCategoryCommand
        {
            Id = entity.Id,
            Name = "Updated Name",
            CategoryId = Guid.Parse("00000000-0000-0000-0000-000000000002"),
            ConcurrencyToken = "token-v1",
        }, CancellationToken.None);

        Assert.False(result.IsError);
        Assert.Equal(1, unitOfWork.SaveCalls);
        Assert.Equal(entity.Id, result.Value.Id);
        Assert.True(EqualityComparer<string>.Default.Equals("Updated Name", result.Value.Name));
        Assert.True(EqualityComparer<Guid>.Default.Equals(Guid.Parse("00000000-0000-0000-0000-000000000002"), result.Value.CategoryId));
        Assert.Equal("token-v2", result.Value.ConcurrencyToken);
    }

    [Fact]
    public async Task UpdateCommandHandlerMapsNotFoundConflictAndInvalidToken()
    {
        var repository = new FakeCategoryRepository();
        var entity = DomainCategory.Create(new CategoryState { Name = "Name Value", CategoryId = Guid.Parse("00000000-0000-0000-0000-000000000001"),  });
        repository.Items.Add(entity);
        var unitOfWork = new FakeUnitOfWork(repository);
        var handler = new UpdateCategoryCommandHandler(repository, unitOfWork);

        var missing = await handler.Handle(new UpdateCategoryCommand { Id = Guid.NewGuid(), Name = "Name Value", CategoryId = Guid.Parse("00000000-0000-0000-0000-000000000001"),  ConcurrencyToken = "token-v1" }, CancellationToken.None);
        unitOfWork.ConflictOnNextSave = true;
        var conflict = await handler.Handle(new UpdateCategoryCommand { Id = entity.Id, Name = "Updated Name", CategoryId = Guid.Parse("00000000-0000-0000-0000-000000000002"),  ConcurrencyToken = "token-v1" }, CancellationToken.None);
        var invalid = await handler.Handle(new UpdateCategoryCommand { Id = entity.Id, Name = "Updated Name", CategoryId = Guid.Parse("00000000-0000-0000-0000-000000000002"),  ConcurrencyToken = "unknown-token" }, CancellationToken.None);

        Assert.Equal(ErrorType.NotFound, missing.FirstError.Type);
        Assert.Equal(ErrorType.Conflict, conflict.FirstError.Type);
        Assert.Equal(ErrorType.Validation, invalid.FirstError.Type);
        Assert.Equal("ConcurrencyToken.Invalid", invalid.FirstError.Code);
    }

    [Fact]
    public async Task DeleteCommandValidatorRequiresConcurrencyToken()
    {
        var result = await new DeleteCategoryCommandValidator().ValidateAsync(new DeleteCategoryCommand(Guid.NewGuid(), ""));

        Assert.Contains(result.Errors, error => error.ErrorCode == "ConcurrencyToken.Required");
    }

    [Fact]
    public async Task DeleteCommandHandlerMapsNotFoundConflictInvalidTokenAndSuccess()
    {
        var repository = new FakeCategoryRepository();
        var entity = DomainCategory.Create(new CategoryState { Name = "Name Value", CategoryId = Guid.Parse("00000000-0000-0000-0000-000000000001"),  });
        repository.Items.Add(entity);
        var unitOfWork = new FakeUnitOfWork(repository);
        var handler = new DeleteCategoryCommandHandler(repository, unitOfWork);

        var missing = await handler.Handle(new DeleteCategoryCommand(Guid.NewGuid(), "token-v1"), CancellationToken.None);
        var missingMalformed = await handler.Handle(new DeleteCategoryCommand(Guid.NewGuid(), "unknown-token"), CancellationToken.None);
        unitOfWork.ConflictOnNextSave = true;
        var conflict = await handler.Handle(new DeleteCategoryCommand(entity.Id, "token-v1"), CancellationToken.None);
        var invalid = await handler.Handle(new DeleteCategoryCommand(entity.Id, "unknown-token"), CancellationToken.None);
        var deleted = await handler.Handle(new DeleteCategoryCommand(entity.Id, "token-v1"), CancellationToken.None);

        Assert.Equal(ErrorType.NotFound, missing.FirstError.Type);
        Assert.Equal(ErrorType.NotFound, missingMalformed.FirstError.Type);
        Assert.Equal("Category.NotFound", missingMalformed.FirstError.Code);
        Assert.Equal(ErrorType.Conflict, conflict.FirstError.Type);
        Assert.Equal(ErrorType.Validation, invalid.FirstError.Type);
        Assert.False(deleted.IsError);
        Assert.Equal(Result.Deleted, deleted.Value);
    }

    [Fact]
    public async Task ListAndGetByIdHandlersMapRepositorySnapshots()
    {
        var repository = new FakeCategoryRepository();
        var entity = DomainCategory.Create(new CategoryState { Name = "Name Value", CategoryId = Guid.Parse("00000000-0000-0000-0000-000000000001"),  });
        repository.Items.Add(entity);

        var page = await new ListCategoryQueryHandler(repository).Handle(new ListCategoryQuery(1, 20), CancellationToken.None);
        var item = await new GetCategoryByIdQueryHandler(repository).Handle(new GetCategoryByIdQuery(entity.Id), CancellationToken.None);

        Assert.False(page.IsError);
        Assert.Single(page.Value.Items);
        Assert.False(item.IsError);
        Assert.Equal(entity.Id, item.Value.Id);
        Assert.Equal("token-v1", item.Value.ConcurrencyToken);
    }

    private sealed class FakeCategoryRepository : ICategoryRepository
    {
        public List<DomainCategory> Items { get; } = [];
        public int AddCalls { get; private set; }
        private string CurrentToken { get; set; } = "token-v1";
        private bool PendingTokenUpdate { get; set; }
        private DomainCategory? PendingDelete { get; set; }

        public Task<(IReadOnlyList<EntitySnapshot<DomainCategory>> Items, int TotalCount)> ListAsync(int skip, int take, CancellationToken cancellationToken) =>
            Task.FromResult(((IReadOnlyList<EntitySnapshot<DomainCategory>>)Items.Skip(skip).Take(take).Select(ToSnapshot).ToList(), Items.Count));

        public Task<EntitySnapshot<DomainCategory>?> GetByIdAsync(Guid id, CancellationToken cancellationToken) =>
            Task.FromResult(Items.SingleOrDefault(item => item.Id == id) is { } entity ? ToSnapshot(entity) : null);

        public Task AddAsync(DomainCategory entity, CancellationToken cancellationToken)
        {
            AddCalls++;
            Items.Add(entity);
            return Task.CompletedTask;
        }

        public Task<MutationPreparationStatus> UpdateAsync(DomainCategory entity, string concurrencyToken, CancellationToken cancellationToken)
        {
            if (concurrencyToken != CurrentToken) return Task.FromResult(MutationPreparationStatus.InvalidToken);
            PendingTokenUpdate = true;
            return Task.FromResult(MutationPreparationStatus.Prepared);
        }

        public Task<MutationPreparationStatus> DeleteAsync(DomainCategory entity, string concurrencyToken, CancellationToken cancellationToken)
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

        private EntitySnapshot<DomainCategory> ToSnapshot(DomainCategory entity) => new(entity, CurrentToken);
    }

    private sealed class FakeUnitOfWork(FakeCategoryRepository repository) : IUnitOfWork
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
