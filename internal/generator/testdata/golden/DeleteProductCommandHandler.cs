using ErrorOr;
using MediatR;
using ProductService.Application.Common;
using ProductService.Application.Products.Interfaces;
using ProductService.Domain.Entities;

namespace ProductService.Application.Products.Commands.Delete;

public sealed class DeleteProductCommandHandler(IProductRepository repository, IUnitOfWork unitOfWork) : IRequestHandler<DeleteProductCommand, ErrorOr<Deleted>>
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
        if (status == MutationPreparationStatus.InvalidToken)
        {
            return Error.Validation(code: "ConcurrencyToken.Invalid", description: "Invalid concurrency token.", metadata: new Dictionary<string, object> { ["field"] = "concurrencyToken" });
        }

        try
        {
            await unitOfWork.SaveChangesAsync(cancellationToken);
        }
        catch (ConcurrencyConflictException)
        {
            return Error.Conflict(code: "Product.ConcurrencyConflict", description: "Product was changed by another request.");
        }

        return Result.Deleted;
    }
}
