using ErrorOr;
using MediatR;
using ProductService.Application.Categories.Dtos;

namespace ProductService.Application.Categories.Queries.GetById;

public sealed record GetCategoryByIdQuery(Guid Id, string? Shape = null) : IRequest<ErrorOr<CategoryDto>>;
