using ErrorOr;
using MediatR;
using ProductService.Application.Features.Products;

namespace ProductService.Application.Features.Products.GetById;

public sealed record GetProductByIdQuery(Guid Id) : IRequest<ErrorOr<ProductDto>>;
