using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.Logging;
using ProductService.Application.Common;
using ProductService.Application.Features.Products;
using ProductService.Domain.Features.Products;
using ProductService.Domain.Shared;
using ProductService.Infrastructure.Persistence;

namespace ProductService.Infrastructure.Persistence.Features.Products;

public sealed class ProductRepository(ProductServiceDbContext dbContext, ILogger<ProductRepository> logger) : IProductRepository
{
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

    public async Task<EntitySnapshot<Product>> AddAsync(Product entity, CancellationToken cancellationToken)
    {
        using var activity = ProductService.Infrastructure.DependencyInjection.ActivitySource.StartActivity("Product.add");
        await dbContext.Products.AddAsync(entity, cancellationToken);
        await dbContext.SaveChangesAsync(cancellationToken);
        return ToSnapshot(entity);
    }

    public async Task<SaveResultStatus> UpdateAsync(Product entity, string concurrencyToken, CancellationToken cancellationToken)
    {
        using var activity = ProductService.Infrastructure.DependencyInjection.ActivitySource.StartActivity("Product.update");
        try
        {
            if (!TryDecodeToken(concurrencyToken, out var rowVersion))
            {
                return SaveResultStatus.InvalidToken;
            }
            dbContext.Products.Update(entity);
            dbContext.Entry(entity).Property("RowVersion").OriginalValue = rowVersion;
            await dbContext.SaveChangesAsync(cancellationToken);
            return SaveResultStatus.Saved;
        }
        catch (DbUpdateConcurrencyException)
        {
            activity?.AddEvent(new("sql.concurrency_conflict"));
            return SaveResultStatus.Conflict;
        }
    }

    public async Task<SaveResultStatus> DeleteAsync(Product entity, string concurrencyToken, CancellationToken cancellationToken)
    {
        using var activity = ProductService.Infrastructure.DependencyInjection.ActivitySource.StartActivity("Product.delete");
        try
        {
            if (!TryDecodeToken(concurrencyToken, out var rowVersion))
            {
                return SaveResultStatus.InvalidToken;
            }
            dbContext.Products.Remove(entity);
            dbContext.Entry(entity).Property("RowVersion").OriginalValue = rowVersion;
            await dbContext.SaveChangesAsync(cancellationToken);
            return SaveResultStatus.Saved;
        }
        catch (DbUpdateConcurrencyException)
        {
            activity?.AddEvent(new("sql.concurrency_conflict"));
            return SaveResultStatus.Conflict;
        }
    }

    private EntitySnapshot<Product> ToSnapshot(Product entity) =>
        new(entity, Convert.ToBase64String((byte[])dbContext.Entry(entity).Property("RowVersion").CurrentValue!));

    private async Task<(string Field, IReadOnlyList<Guid> RecordIds)> FindReconstitutionDiagnosticsAsync(DomainReconstitutionException exception, Guid? id, CancellationToken cancellationToken)
    {
        await Task.CompletedTask;
        var code = exception.InvariantCodes.FirstOrDefault() ?? string.Empty;
        if (exception.ValueObjectType == "ProductName")
        {
            var predicate = code switch
            {
                "ProductName.Required" => "([Name] IS NULL OR LTRIM(RTRIM([Name])) = N'')",
                "ProductName.MaxLength" => "([Name] IS NOT NULL AND LEN([Name]) > 100)",
                _ => string.Empty
            };
            if (predicate.Length == 0)
            {
                return ("Name", []);
            }
            var sql = id.HasValue
                ? "SELECT TOP(5) [Id] AS [Value] FROM [Products] WHERE [Id] = {0} AND " + predicate + " ORDER BY [Id]"
                : "SELECT TOP(5) [Id] AS [Value] FROM [Products] WHERE " + predicate + " ORDER BY [Id]";
            var ids = id.HasValue
                ? await dbContext.Database.SqlQueryRaw<Guid>(sql, id.Value).ToListAsync(cancellationToken)
                : await dbContext.Database.SqlQueryRaw<Guid>(sql).ToListAsync(cancellationToken);
            return ("Name", ids);
        }
        if (exception.ValueObjectType == "ProductPrice")
        {
            var predicate = code switch
            {
                "ProductPrice.Minimum" => "[Price] < 0",
                "ProductPrice.Maximum" => "[Price] > 999999.99",
                _ => string.Empty
            };
            if (predicate.Length == 0)
            {
                return ("Price", []);
            }
            var sql = id.HasValue
                ? "SELECT TOP(5) [Id] AS [Value] FROM [Products] WHERE [Id] = {0} AND " + predicate + " ORDER BY [Id]"
                : "SELECT TOP(5) [Id] AS [Value] FROM [Products] WHERE " + predicate + " ORDER BY [Id]";
            var ids = id.HasValue
                ? await dbContext.Database.SqlQueryRaw<Guid>(sql, id.Value).ToListAsync(cancellationToken)
                : await dbContext.Database.SqlQueryRaw<Guid>(sql).ToListAsync(cancellationToken);
            return ("Price", ids);
        }
        return (string.Empty, []);
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
        logger.LogError(exception, "Invalid persisted value while materializing {Service}/{Entity}.{Field} during {Operation}. RecordIds={RecordIds}; ValueObject={ValueObject}; InvariantCodes={InvariantCodes}", "ProductService", "Product", field, operation, recordIdsText, exception.ValueObjectType, string.Join(",", exception.InvariantCodes));
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
