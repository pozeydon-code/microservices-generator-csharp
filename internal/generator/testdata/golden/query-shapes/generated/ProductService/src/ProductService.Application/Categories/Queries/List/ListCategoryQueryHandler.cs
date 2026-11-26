using ErrorOr;
using MediatR;
using ProductService.Application.Common;
using ProductService.Application.Categories.Dtos;
using ProductService.Application.Categories.Interfaces;
using ProductService.Domain.Entities;


namespace ProductService.Application.Categories.Queries.List;

public sealed class ListCategoryQueryHandler(ICategoryRepository repository) : IRequestHandler<ListCategoryQuery, ErrorOr<PagedResult<CategoryDto>>>
{
    public async Task<ErrorOr<PagedResult<CategoryDto>>> Handle(ListCategoryQuery request, CancellationToken cancellationToken)
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
            return new PagedResult<CategoryDto>(shapedResult.Items.Select(ToDtoFromEntity).ToList(), normalized.Page, normalized.PageSize, shapedResult.TotalCount);
        }
        var result = await repository.ListAsync(normalized.Offset, normalized.PageSize, cancellationToken);
        return new PagedResult<CategoryDto>(result.Items.Select(ToDto).ToList(), normalized.Page, normalized.PageSize, result.TotalCount);
    }

    private static CategoryDto ToDtoFromEntity(Category entity) => new(
        entity.Id,
        entity.Name,
        entity.CategoryId,
        Convert.ToBase64String(entity.RowVersion));

    private static CategoryDto ToDto(EntitySnapshot<Category> snapshot) => new(
        snapshot.Entity.Id,
        snapshot.Entity.Name,
        snapshot.Entity.CategoryId,
        snapshot.ConcurrencyToken);
}
