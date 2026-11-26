using ProductService.Application.Common;
using ProductService.Domain.Entities;

namespace ProductService.Application.Categories.Interfaces;

public interface ICategoryRepository
{
    Task<(IReadOnlyList<EntitySnapshot<Category>> Items, int TotalCount)> ListAsync(int skip, int take, CancellationToken cancellationToken);
    Task<EntitySnapshot<Category>?> GetByIdAsync(Guid id, CancellationToken cancellationToken);
    Task AddAsync(Category entity, CancellationToken cancellationToken);
    Task<MutationPreparationStatus> UpdateAsync(Category entity, string concurrencyToken, CancellationToken cancellationToken);
    Task<MutationPreparationStatus> DeleteAsync(Category entity, string concurrencyToken, CancellationToken cancellationToken);
}
