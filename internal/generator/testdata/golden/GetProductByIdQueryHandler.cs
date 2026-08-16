using ErrorOr;
using MediatR;
using ProductService.Application.Common;
using ProductService.Application.Features.Products;
using ProductService.Domain.Entities;

namespace ProductService.Application.Features.Products.GetById;

public sealed class GetProductByIdQueryHandler(IProductRepository repository) : IRequestHandler<GetProductByIdQuery, ErrorOr<ProductDto>>
{
    public async Task<ErrorOr<ProductDto>> Handle(GetProductByIdQuery request, CancellationToken cancellationToken)
    {
        var snapshot = await repository.GetByIdAsync(request.Id, cancellationToken);
        return snapshot is null
            ? Error.NotFound(code: "Product.NotFound", description: "Product was not found.")
            : ToDto(snapshot);
    }

    private static ProductDto ToDto(EntitySnapshot<Product> snapshot) => new(
        snapshot.Entity.Id,
        snapshot.Entity.IsActive,
        snapshot.Entity.Name.Value,
        snapshot.Entity.Price.Value,
        snapshot.ConcurrencyToken);
}
