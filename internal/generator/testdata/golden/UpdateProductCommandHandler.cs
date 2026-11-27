using ErrorOr;
using MediatR;
using ProductService.Application.Common;
using ProductService.Application.Products.Dtos;
using ProductService.Application.Products.Interfaces;
using ProductService.Domain.Entities;
using ProductService.Domain.ValueObjects;


namespace ProductService.Application.Products.Commands.Update;

public sealed class UpdateProductCommandHandler(IProductRepository repository, IUnitOfWork unitOfWork) : IRequestHandler<UpdateProductCommand, ErrorOr<ProductDto>>
{
    public async Task<ErrorOr<ProductDto>> Handle(UpdateProductCommand request, CancellationToken cancellationToken)
    {
        var nameResult = ProductName.Create(request.Name, "name");
        if (nameResult.IsError)
        {
            return nameResult.Errors;
        }

        var priceResult = ProductPrice.Create(request.Price, "price");
        if (priceResult.IsError)
        {
            return priceResult.Errors;
        }

        if (string.IsNullOrWhiteSpace(request.ConcurrencyToken))
        {
            return Error.Validation(
                code: "ConcurrencyToken.Required",
                description: "Concurrency token is required.",
                metadata: new Dictionary<string, object> { ["field"] = "concurrencyToken" });
        }

        var snapshot = await repository.GetByIdAsync(request.Id, cancellationToken);
        if (snapshot is null)
        {
            return Error.NotFound(code: "Product.NotFound", description: "Product was not found.");
        }
        snapshot.Entity.Update(new ProductState
        {
            IsActive = request.IsActive,
            Name = nameResult.Value!,
            Price = priceResult.Value!,
        });

        var status = await repository.UpdateAsync(snapshot.Entity, request.ConcurrencyToken, cancellationToken);
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

        var updated = await repository.GetByIdAsync(request.Id, cancellationToken);
        return ToDto(updated ?? snapshot);
    }

    private static ProductDto ToDto(EntitySnapshot<Product> snapshot) => new(
        snapshot.Entity.Id,
        snapshot.Entity.IsActive,
        snapshot.Entity.Name.Value,
        snapshot.Entity.Price.Value,
        snapshot.ConcurrencyToken);
}
