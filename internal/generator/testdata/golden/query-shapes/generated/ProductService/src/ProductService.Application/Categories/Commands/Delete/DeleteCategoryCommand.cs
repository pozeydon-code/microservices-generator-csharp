using ErrorOr;
using MediatR;

namespace ProductService.Application.Categories.Commands.Delete;

public sealed record DeleteCategoryCommand(Guid Id, string ConcurrencyToken) : IRequest<ErrorOr<Deleted>>;
