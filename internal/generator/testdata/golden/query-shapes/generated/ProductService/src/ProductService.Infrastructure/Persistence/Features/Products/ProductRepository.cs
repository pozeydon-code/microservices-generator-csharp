using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.Logging;
using ProductService.Application.Common;
using ProductService.Application.Products.Interfaces;
using ProductService.Domain.Entities;
using ProductService.Domain.Primitives;
using ProductService.Infrastructure.Persistence;

namespace ProductService.Infrastructure.Persistence.Features.Products;

public sealed class ProductRepository(ProductServiceDbContext dbContext, ILogger<ProductRepository> logger) : IProductRepository
{
    private static readonly Action<ILogger, string, string, string, string, string, Exception?> InvalidPersistedValue = LoggerMessage.Define<string, string, string, string, string>(
        LogLevel.Error,
        new EventId(1, nameof(InvalidPersistedValue)),
        "Invalid persisted value while materializing ProductService/Product.{Field} during {Operation}. RecordIds={RecordIds}; ValueObject={ValueObject}; InvariantCodes={InvariantCodes}");

    public async Task<(IReadOnlyList<EntitySnapshot<Product>> Items, int TotalCount)> ListAsync(int skip, int take, CancellationToken cancellationToken)
    {
        using var activity = ProductService.Infrastructure.DependencyInjection.ActivitySource.StartActivity("Product.list");
        var query = dbContext.Products.OrderBy(entity => entity.Id);
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

    public async Task<EntitySnapshot<Product>?> GetByIdAsync(Guid id, CancellationToken cancellationToken)
    {
        using var activity = ProductService.Infrastructure.DependencyInjection.ActivitySource.StartActivity("Product.get");
        try
        {
            var entity = await dbContext.Products.FindAsync([id], cancellationToken);
            return entity is null ? null : ToSnapshot(entity);
        }
        catch (DomainReconstitutionException ex)
        {
            var diagnostics = await FindReconstitutionDiagnosticsAsync(ex, id, cancellationToken);
            RecordReconstitutionFailure(activity, ex, "get", diagnostics.Field, diagnostics.RecordIds);
            throw;
        }
    }

    public async Task AddAsync(Product entity, CancellationToken cancellationToken)
    {
        using var activity = ProductService.Infrastructure.DependencyInjection.ActivitySource.StartActivity("Product.add");
        await dbContext.Products.AddAsync(entity, cancellationToken);
    }

    public Task<MutationPreparationStatus> UpdateAsync(Product entity, string concurrencyToken, CancellationToken cancellationToken)
    {
        using var activity = ProductService.Infrastructure.DependencyInjection.ActivitySource.StartActivity("Product.update");
        _ = cancellationToken;
        if (!TryDecodeToken(concurrencyToken, out var rowVersion))
        {
            return Task.FromResult(MutationPreparationStatus.InvalidToken);
        }
        dbContext.Products.Update(entity);
        dbContext.Entry(entity).Property(item => item.RowVersion).OriginalValue = rowVersion;
        return Task.FromResult(MutationPreparationStatus.Prepared);
    }

    public Task<MutationPreparationStatus> DeleteAsync(Product entity, string concurrencyToken, CancellationToken cancellationToken)
    {
        using var activity = ProductService.Infrastructure.DependencyInjection.ActivitySource.StartActivity("Product.delete");
        _ = cancellationToken;
        if (!TryDecodeToken(concurrencyToken, out var rowVersion))
        {
            return Task.FromResult(MutationPreparationStatus.InvalidToken);
        }
        dbContext.Products.Remove(entity);
        dbContext.Entry(entity).Property(item => item.RowVersion).OriginalValue = rowVersion;
        return Task.FromResult(MutationPreparationStatus.Prepared);
    }

    private EntitySnapshot<Product> ToSnapshot(Product entity) =>
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
            ["entity"] = "Product",
            ["field"] = field,
            ["operation"] = operation,
            ["value_object"] = exception.ValueObjectType,
            ["invariant_codes"] = string.Join(",", exception.InvariantCodes),
            ["record_ids"] = recordIdsText
        }));
        InvalidPersistedValue(logger, field, operation, recordIdsText, exception.ValueObjectType, string.Join(",", exception.InvariantCodes), exception);
    }

    public async Task<(IReadOnlyList<Product> Items, int TotalCount)> ListShaped_summaryAsync(int skip, int take, CancellationToken cancellationToken)
    {
        using var activity = ProductService.Infrastructure.DependencyInjection.ActivitySource.StartActivity("Product.list.summary");
        var query = dbContext.Products.AsNoTracking();
        query = ApplyShaping_summary(query);
        var orderedQuery = query.OrderBy(entity => entity.Id);
        var totalCount = await orderedQuery.CountAsync(cancellationToken);
        var entities = await orderedQuery.Skip(skip).Take(take).ToListAsync(cancellationToken);
        return (entities, totalCount);
    }

    public async Task<Product?> GetShaped_summaryAsync(Guid id, CancellationToken cancellationToken)
    {
        using var activity = ProductService.Infrastructure.DependencyInjection.ActivitySource.StartActivity("Product.get.summary");
        var query = dbContext.Products.AsNoTracking();
        query = ApplyShaping_summary(query);
        return await query.FirstOrDefaultAsync(e => e.Id == id, cancellationToken);
    }

    private static IQueryable<Product> ApplyShaping_summary(IQueryable<Product> query)
    {
        query = query.Include(e => e.Categorys);
        return query;
    }

    public async Task<(IReadOnlyList<Product> Items, int TotalCount)> ListShaped_detailAsync(int skip, int take, CancellationToken cancellationToken)
    {
        using var activity = ProductService.Infrastructure.DependencyInjection.ActivitySource.StartActivity("Product.list.detail");
        var query = dbContext.Products.AsNoTracking();
        query = ApplyShaping_detail(query);
        var orderedQuery = query.OrderBy(entity => entity.Id);
        var totalCount = await orderedQuery.CountAsync(cancellationToken);
        var entities = await orderedQuery.Skip(skip).Take(take).ToListAsync(cancellationToken);
        return (entities, totalCount);
    }

    public async Task<Product?> GetShaped_detailAsync(Guid id, CancellationToken cancellationToken)
    {
        using var activity = ProductService.Infrastructure.DependencyInjection.ActivitySource.StartActivity("Product.get.detail");
        var query = dbContext.Products.AsNoTracking();
        query = ApplyShaping_detail(query);
        return await query.FirstOrDefaultAsync(e => e.Id == id, cancellationToken);
    }

    private static IQueryable<Product> ApplyShaping_detail(IQueryable<Product> query)
    {
        query = query.Include(e => e.Categorys);
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
