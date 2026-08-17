using ErrorOr;
using MediatR;
using ProductService.Application.Products.Dtos;

namespace ProductService.Application.Products.Commands.Create;

public sealed record CreateProductCommand : IRequest<ErrorOr<ProductDto>>
{
    public bool IsActive { get; init; }
    public string Name { get; init; } = string.Empty;
    public decimal Price { get; init; }
}
