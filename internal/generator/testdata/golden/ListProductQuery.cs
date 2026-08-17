using ErrorOr;
using MediatR;
using ProductService.Application.Common;
using ProductService.Application.Products.Dtos;

namespace ProductService.Application.Products.Queries.List;

public sealed record ListProductQuery(int? Page, int? PageSize) : IRequest<ErrorOr<PagedResult<ProductDto>>>;
