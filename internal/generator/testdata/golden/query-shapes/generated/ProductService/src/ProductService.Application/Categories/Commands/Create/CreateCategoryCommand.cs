using ErrorOr;
using MediatR;
using ProductService.Application.Categories.Dtos;

namespace ProductService.Application.Categories.Commands.Create;

public sealed record CreateCategoryCommand : IRequest<ErrorOr<CategoryDto>>
{
    public string Name { get; init; } = string.Empty;
    public Guid CategoryId { get; init; }
}
