using ErrorOr;
using MediatR;
using ProductService.Application.Common;
using ProductService.Application.Categories.Dtos;

namespace ProductService.Application.Categories.Queries.List;

public sealed record ListCategoryQuery(int? Page, int? PageSize, string? Shape = null) : IRequest<ErrorOr<PagedResult<CategoryDto>>>;
