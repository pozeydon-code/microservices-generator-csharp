using ErrorOr;
using MediatR;
using ProductService.Application.Common;
using ProductService.Application.Features.Products;

namespace ProductService.Application.Features.Products.List;

public sealed record ListProductQuery(int? Page, int? PageSize) : IRequest<ErrorOr<PagedResult<ProductDto>>>;
