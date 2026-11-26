using ErrorOr;
using MediatR;
using ProductService.Application.Common;
using ProductService.Application.Products.Dtos;
using ProductService.Application.Products.Interfaces;
using ProductService.Domain.Entities;

namespace ProductService.Application.Products.Queries.GetById;

public sealed class GetProductByIdQueryHandler(IProductRepository repository) : IRequestHandler<GetProductByIdQuery, ErrorOr<ProductDto>>
{
    public async Task<ErrorOr<ProductDto>> Handle(GetProductByIdQuery request, CancellationToken cancellationToken)
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
                ? Error.NotFound(code: "Product.NotFound", description: "Product was not found.")
                : ToDtoFromEntity(entity);
        }
        var snapshot = await repository.GetByIdAsync(request.Id, cancellationToken);
        return snapshot is null
            ? Error.NotFound(code: "Product.NotFound", description: "Product was not found.")
            : ToDto(snapshot);
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
