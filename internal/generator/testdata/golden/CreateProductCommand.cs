using ErrorOr;
using MediatR;

namespace ProductService.Application.Features.Products.Create;

public sealed record CreateProductCommand : IRequest<ErrorOr<ProductDto>>
{
    public bool IsActive { get; init; }
    public string Name { get; init; } = string.Empty;
    public decimal Price { get; init; }
}
