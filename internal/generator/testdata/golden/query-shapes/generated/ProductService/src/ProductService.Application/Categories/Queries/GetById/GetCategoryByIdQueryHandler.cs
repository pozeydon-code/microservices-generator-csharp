using ErrorOr;
using MediatR;
using ProductService.Application.Common;
using ProductService.Application.Categories.Dtos;
using ProductService.Application.Categories.Interfaces;
using ProductService.Domain.Entities;

namespace ProductService.Application.Categories.Queries.GetById;

public sealed class GetCategoryByIdQueryHandler(ICategoryRepository repository) : IRequestHandler<GetCategoryByIdQuery, ErrorOr<CategoryDto>>
{
    public async Task<ErrorOr<CategoryDto>> Handle(GetCategoryByIdQuery request, CancellationToken cancellationToken)
    {
        if (request.Shape is not null)
        {
            var entity = request.Shape switch
            {
                "summary" => await repository.GetShaped_summaryAsync(request.Id, cancellationToken),
                "detail" => await repository.GetShaped_detailAsync(request.Id, cancellationToken),
                _ => null
            };
            return entity is null
                ? Error.NotFound(code: "Category.NotFound", description: "Category was not found.")
                : ToDtoFromEntity(entity);
        }
        var snapshot = await repository.GetByIdAsync(request.Id, cancellationToken);
        return snapshot is null
            ? Error.NotFound(code: "Category.NotFound", description: "Category was not found.")
            : ToDto(snapshot);
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
