using ErrorOr;
using MediatR;
using ProductService.Application.Products.Dtos;

namespace ProductService.Application.Products.Queries.GetById;

public sealed record GetProductByIdQuery(Guid Id) : IRequest<ErrorOr<ProductDto>>;
