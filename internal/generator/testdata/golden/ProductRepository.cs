using Microsoft.EntityFrameworkCore;
using ProductService.Application.Common;
using ProductService.Application.Products.Interfaces;
using ProductService.Domain.Entities;
using ProductService.Infrastructure.Persistence;

namespace ProductService.Infrastructure.Persistence.Features.Products;

public sealed class ProductRepository(ProductServiceDbContext dbContext) : IProductRepository
{
    public async Task<(IReadOnlyList<EntitySnapshot<Product>> Items, int TotalCount)> ListAsync(int skip, int take, CancellationToken cancellationToken)
    {
        using var activity = ProductService.Infrastructure.DependencyInjection.ActivitySource.StartActivity("Product.list");
        var query = dbContext.Products.OrderBy(entity => entity.Id);
        var totalCount = await query.CountAsync(cancellationToken);
        var entities = await query.Skip(skip).Take(take).ToListAsync(cancellationToken);
        var items = entities.Select(ToSnapshot).ToList();
        return (items, totalCount);
    }

    public async Task<EntitySnapshot<Product>?> GetByIdAsync(Guid id, CancellationToken cancellationToken)
    {
        using var activity = ProductService.Infrastructure.DependencyInjection.ActivitySource.StartActivity("Product.get");
        var entity = await dbContext.Products.FindAsync([id], cancellationToken);
        return entity is null ? null : ToSnapshot(entity);
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
