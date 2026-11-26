using ErrorOr;
using MediatR;
using ProductService.Application.Categories.Dtos;

namespace ProductService.Application.Categories.Commands.Update;

public sealed record UpdateCategoryCommand : IRequest<ErrorOr<CategoryDto>>
{
    public required Guid Id { get; init; }
    public string Name { get; init; } = string.Empty;
    public Guid CategoryId { get; init; }
    public string ConcurrencyToken { get; init; } = string.Empty;
}
