using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.Logging;
using ProductService.Application.Common;
using ProductService.Application.Categories.Interfaces;
using ProductService.Domain.Entities;
using ProductService.Domain.Primitives;
using ProductService.Infrastructure.Persistence;

namespace ProductService.Infrastructure.Persistence.Features.Categories;

public sealed class CategoryRepository(ProductServiceDbContext dbContext, ILogger<CategoryRepository> logger) : ICategoryRepository
{
    private static readonly Action<ILogger, string, string, string, string, string, Exception?> InvalidPersistedValue = LoggerMessage.Define<string, string, string, string, string>(
        LogLevel.Error,
        new EventId(1, nameof(InvalidPersistedValue)),
        "Invalid persisted value while materializing ProductService/Category.{Field} during {Operation}. RecordIds={RecordIds}; ValueObject={ValueObject}; InvariantCodes={InvariantCodes}");

    public async Task<(IReadOnlyList<EntitySnapshot<Category>> Items, int TotalCount)> ListAsync(int skip, int take, CancellationToken cancellationToken)
    {
        using var activity = ProductService.Infrastructure.DependencyInjection.ActivitySource.StartActivity("Category.list");
        var query = dbContext.Categorys.OrderBy(entity => entity.Id);
        var totalCount = await query.CountAsync(cancellationToken);
        try
        {
            var entities = await query.Skip(skip).Take(take).ToListAsync(cancellationToken);
            var items = entities.Select(ToSnapshot).ToList();
            return (items, totalCount);
        }
        catch (DomainReconstitutionException ex)
        {
            var diagnostics = await FindReconstitutionDiagnosticsAsync(ex, null, cancellationToken);
            RecordReconstitutionFailure(activity, ex, "list", diagnostics.Field, diagnostics.RecordIds);
            throw;
        }
    }

    public async Task<EntitySnapshot<Category>?> GetByIdAsync(Guid id, CancellationToken cancellationToken)
    {
        using var activity = ProductService.Infrastructure.DependencyInjection.ActivitySource.StartActivity("Category.get");
        try
        {
            var entity = await dbContext.Categorys.FindAsync([id], cancellationToken);
            return entity is null ? null : ToSnapshot(entity);
        }
        catch (DomainReconstitutionException ex)
        {
            var diagnostics = await FindReconstitutionDiagnosticsAsync(ex, id, cancellationToken);
            RecordReconstitutionFailure(activity, ex, "get", diagnostics.Field, diagnostics.RecordIds);
            throw;
        }
    }

    public async Task AddAsync(Category entity, CancellationToken cancellationToken)
    {
        using var activity = ProductService.Infrastructure.DependencyInjection.ActivitySource.StartActivity("Category.add");
        await dbContext.Categorys.AddAsync(entity, cancellationToken);
    }

    public Task<MutationPreparationStatus> UpdateAsync(Category entity, string concurrencyToken, CancellationToken cancellationToken)
    {
        using var activity = ProductService.Infrastructure.DependencyInjection.ActivitySource.StartActivity("Category.update");
        _ = cancellationToken;
        if (!TryDecodeToken(concurrencyToken, out var rowVersion))
        {
            return Task.FromResult(MutationPreparationStatus.InvalidToken);
        }
        dbContext.Categorys.Update(entity);
        dbContext.Entry(entity).Property(item => item.RowVersion).OriginalValue = rowVersion;
        return Task.FromResult(MutationPreparationStatus.Prepared);
    }

    public Task<MutationPreparationStatus> DeleteAsync(Category entity, string concurrencyToken, CancellationToken cancellationToken)
    {
        using var activity = ProductService.Infrastructure.DependencyInjection.ActivitySource.StartActivity("Category.delete");
        _ = cancellationToken;
        if (!TryDecodeToken(concurrencyToken, out var rowVersion))
        {
            return Task.FromResult(MutationPreparationStatus.InvalidToken);
        }
        dbContext.Categorys.Remove(entity);
        dbContext.Entry(entity).Property(item => item.RowVersion).OriginalValue = rowVersion;
        return Task.FromResult(MutationPreparationStatus.Prepared);
    }

    private EntitySnapshot<Category> ToSnapshot(Category entity) =>
        new(entity, Convert.ToBase64String(entity.RowVersion));

    private static Task<(string Field, IReadOnlyList<Guid> RecordIds)> FindReconstitutionDiagnosticsAsync(DomainReconstitutionException exception, Guid? id, CancellationToken cancellationToken)
    {
        _ = exception;
        _ = id;
        _ = cancellationToken;
        return Task.FromResult<(string Field, IReadOnlyList<Guid> RecordIds)>((string.Empty, []));
    }

    private void RecordReconstitutionFailure(System.Diagnostics.Activity? activity, DomainReconstitutionException exception, string operation, string field, IReadOnlyList<Guid> recordIds)
    {
        var recordIdsText = string.Join(",", recordIds);
        activity?.AddEvent(new("domain.reconstitution_failed", tags: new System.Diagnostics.ActivityTagsCollection
        {
            ["service"] = "ProductService",
            ["entity"] = "Category",
            ["field"] = field,
            ["operation"] = operation,
            ["value_object"] = exception.ValueObjectType,
            ["invariant_codes"] = string.Join(",", exception.InvariantCodes),
            ["record_ids"] = recordIdsText
        }));
        InvalidPersistedValue(logger, field, operation, recordIdsText, exception.ValueObjectType, string.Join(",", exception.InvariantCodes), exception);
    }

    public async Task<(IReadOnlyList<Category> Items, int TotalCount)> ListShaped_summaryAsync(int skip, int take, CancellationToken cancellationToken)
    {
        using var activity = ProductService.Infrastructure.DependencyInjection.ActivitySource.StartActivity("Category.list.summary");
        var query = dbContext.Categorys.AsNoTracking();
        query = ApplyShaping_summary(query);
        var orderedQuery = query.OrderBy(entity => entity.Id);
        var totalCount = await orderedQuery.CountAsync(cancellationToken);
        var entities = await orderedQuery.Skip(skip).Take(take).ToListAsync(cancellationToken);
        return (entities, totalCount);
    }

    public async Task<Category?> GetShaped_summaryAsync(Guid id, CancellationToken cancellationToken)
    {
        using var activity = ProductService.Infrastructure.DependencyInjection.ActivitySource.StartActivity("Category.get.summary");
        var query = dbContext.Categorys.AsNoTracking();
        query = ApplyShaping_summary(query);
        return await query.FirstOrDefaultAsync(e => e.Id == id, cancellationToken);
    }

    private static IQueryable<Category> ApplyShaping_summary(IQueryable<Category> query)
    {
        query = query.Include(e => e.Product);
        return query;
    }

    public async Task<(IReadOnlyList<Category> Items, int TotalCount)> ListShaped_detailAsync(int skip, int take, CancellationToken cancellationToken)
    {
        using var activity = ProductService.Infrastructure.DependencyInjection.ActivitySource.StartActivity("Category.list.detail");
        var query = dbContext.Categorys.AsNoTracking();
        query = ApplyShaping_detail(query);
        var orderedQuery = query.OrderBy(entity => entity.Id);
        var totalCount = await orderedQuery.CountAsync(cancellationToken);
        var entities = await orderedQuery.Skip(skip).Take(take).ToListAsync(cancellationToken);
        return (entities, totalCount);
    }

    public async Task<Category?> GetShaped_detailAsync(Guid id, CancellationToken cancellationToken)
    {
        using var activity = ProductService.Infrastructure.DependencyInjection.ActivitySource.StartActivity("Category.get.detail");
        var query = dbContext.Categorys.AsNoTracking();
        query = ApplyShaping_detail(query);
        return await query.FirstOrDefaultAsync(e => e.Id == id, cancellationToken);
    }

    private static IQueryable<Category> ApplyShaping_detail(IQueryable<Category> query)
    {
        query = query.Include(e => e.Product);
        return query;
    }

    private static bool TryDecodeToken(string concurrencyToken, out byte[] rowVersion)
    {
        try
        {
            rowVersion = Convert.FromBase64String(concurrencyToken);
            return rowVersion.Length == 8;
        }
        catch (FormatException)
        {
            rowVersion = [];
            return false;
        }
    }
}
