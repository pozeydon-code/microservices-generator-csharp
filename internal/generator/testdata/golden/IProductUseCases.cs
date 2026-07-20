using ProductService.Application.Common;

namespace ProductService.Application.Features.Products;

public interface IProductUseCases
{
    Task<PagedResult<ProductDto>> ListAsync(PageRequest request, CancellationToken cancellationToken);
    Task<ProductDto?> GetByIdAsync(Guid id, CancellationToken cancellationToken);
    Task<MutationValidationResult<ProductDto>> CreateAsync(CreateProductRequest request, CancellationToken cancellationToken);
    Task<MutationValidationResult<ProductDto>> UpdateAsync(Guid id, UpdateProductRequest request, CancellationToken cancellationToken);
    Task<MutationResult<bool>> DeleteAsync(Guid id, string concurrencyToken, CancellationToken cancellationToken);
}
