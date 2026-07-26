using ErrorOr;
using MediatR;

namespace ProductService.Application.Features.Products.Delete;

public sealed record DeleteProductCommand(Guid Id, string ConcurrencyToken) : IRequest<ErrorOr<Deleted>>;
