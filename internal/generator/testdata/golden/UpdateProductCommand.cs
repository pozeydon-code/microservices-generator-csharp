using ErrorOr;
using MediatR;

namespace ProductService.Application.Features.Products.Update;

public sealed record UpdateProductCommand : IRequest<ErrorOr<ProductDto>>
{
    public required Guid Id { get; init; }
    public bool IsActive { get; init; }
    public string Name { get; init; } = string.Empty;
    public decimal Price { get; init; }
    public string ConcurrencyToken { get; init; } = string.Empty;
}
