using ErrorOr;
using MediatR;
using ProductService.Application.Common;
using ProductService.Application.Features.Products;
using ProductService.Domain.Features.Products;
using ProductService.Domain.Common.ValueObjects;


namespace ProductService.Application.Features.Products.List;

public sealed class ListProductQueryHandler(IProductRepository repository) : IRequestHandler<ListProductQuery, ErrorOr<PagedResult<ProductDto>>>
{
    public async Task<ErrorOr<PagedResult<ProductDto>>> Handle(ListProductQuery request, CancellationToken cancellationToken)
    {
        var normalized = PaginationPolicy.Normalize(request.Page, request.PageSize);
        var result = await repository.ListAsync(normalized.Offset, normalized.PageSize, cancellationToken);
        return new PagedResult<ProductDto>(result.Items.Select(ToDto).ToList(), normalized.Page, normalized.PageSize, result.TotalCount);
    }

    private static ProductDto ToDto(EntitySnapshot<Product> snapshot) => new(
        snapshot.Entity.Id,
        snapshot.Entity.IsActive,
        snapshot.Entity.Name.Value,
        snapshot.Entity.Price.Value,
        snapshot.ConcurrencyToken);
}
