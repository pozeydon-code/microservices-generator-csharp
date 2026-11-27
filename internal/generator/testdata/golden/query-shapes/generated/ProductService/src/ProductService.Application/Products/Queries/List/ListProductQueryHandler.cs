using ErrorOr;
using MediatR;
using ProductService.Application.Common;
using ProductService.Application.Products.Dtos;
using ProductService.Application.Products.Interfaces;
using ProductService.Domain.Entities;


namespace ProductService.Application.Products.Queries.List;

public sealed class ListProductQueryHandler(IProductRepository repository) : IRequestHandler<ListProductQuery, ErrorOr<PagedResult<ProductDto>>>
{
    public async Task<ErrorOr<PagedResult<ProductDto>>> Handle(ListProductQuery request, CancellationToken cancellationToken)
    {
        var normalized = PaginationPolicy.Normalize(request.Page, request.PageSize);
        if (request.Shape is not null)
        {
            var shapedResult = request.Shape switch
            {
                "summary" => await repository.ListShaped_summaryAsync(normalized.Offset, normalized.PageSize, cancellationToken),
                "detail" => await repository.ListShaped_detailAsync(normalized.Offset, normalized.PageSize, cancellationToken),
                _ => await repository.ListAsync(normalized.Offset, normalized.PageSize, cancellationToken)
            };
            return new PagedResult<ProductDto>(shapedResult.Items.Select(ToDtoFromEntity).ToList(), normalized.Page, normalized.PageSize, shapedResult.TotalCount);
        }
        var result = await repository.ListAsync(normalized.Offset, normalized.PageSize, cancellationToken);
        return new PagedResult<ProductDto>(result.Items.Select(ToDto).ToList(), normalized.Page, normalized.PageSize, result.TotalCount);
    }

    private static ProductDto ToDtoFromEntity(Product entity) => new(
        entity.Id,
        entity.Name,
        entity.Price,
        Convert.ToBase64String(entity.RowVersion));

    private static ProductDto ToDto(EntitySnapshot<Product> snapshot) => new(
        snapshot.Entity.Id,
        snapshot.Entity.Name,
        snapshot.Entity.Price,
        snapshot.ConcurrencyToken);
}
