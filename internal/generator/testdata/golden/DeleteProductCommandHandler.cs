using ErrorOr;
using MediatR;
using ProductService.Application.Common;
using ProductService.Domain.Features.Products;

namespace ProductService.Application.Features.Products.Delete;

public sealed class DeleteProductCommandHandler(IProductRepository repository) : IRequestHandler<DeleteProductCommand, ErrorOr<Deleted>>
{
    public async Task<ErrorOr<Deleted>> Handle(DeleteProductCommand request, CancellationToken cancellationToken)
    {
        var snapshot = await repository.GetByIdAsync(request.Id, cancellationToken);
        if (snapshot is null)
        {
            return Error.NotFound(code: "Product.NotFound", description: "Product was not found.");
        }
        if (string.IsNullOrWhiteSpace(request.ConcurrencyToken))
        {
            return Error.Validation(code: "ConcurrencyToken.Required", description: "Concurrency token is required.", metadata: new Dictionary<string, object> { ["field"] = "concurrencyToken" });
        }

        var status = await repository.DeleteAsync(snapshot.Entity, request.ConcurrencyToken, cancellationToken);
        if (status == SaveResultStatus.Conflict)
        {
            return Error.Conflict(code: "Product.ConcurrencyConflict", description: "Product was changed by another request.");
        }
        if (status == SaveResultStatus.InvalidToken)
        {
            return Error.Validation(code: "ConcurrencyToken.Invalid", description: "Invalid concurrency token.", metadata: new Dictionary<string, object> { ["field"] = "concurrencyToken" });
        }

        return Result.Deleted;
    }
}
