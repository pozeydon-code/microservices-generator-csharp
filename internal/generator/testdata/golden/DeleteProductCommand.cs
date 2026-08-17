using ErrorOr;
using MediatR;

namespace ProductService.Application.Products.Commands.Delete;

public sealed record DeleteProductCommand(Guid Id, string ConcurrencyToken) : IRequest<ErrorOr<Deleted>>;
