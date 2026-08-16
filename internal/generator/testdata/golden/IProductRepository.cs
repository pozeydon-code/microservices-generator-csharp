using ProductService.Application.Common;
using ProductService.Domain.Entities;

namespace ProductService.Application.Features.Products;

public interface IProductRepository
{
    Task<(IReadOnlyList<EntitySnapshot<Product>> Items, int TotalCount)> ListAsync(int skip, int take, CancellationToken cancellationToken);
    Task<EntitySnapshot<Product>?> GetByIdAsync(Guid id, CancellationToken cancellationToken);
    Task AddAsync(Product entity, CancellationToken cancellationToken);
    Task<SaveResultStatus> UpdateAsync(Product entity, string concurrencyToken, CancellationToken cancellationToken);
    Task<SaveResultStatus> DeleteAsync(Product entity, string concurrencyToken, CancellationToken cancellationToken);
}
