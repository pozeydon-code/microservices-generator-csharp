using ErrorOr;
using MediatR;
using ProductService.Application.Common;
using ProductService.Domain.Features.Products;
using ProductService.Domain.Common.ValueObjects;


namespace ProductService.Application.Features.Products.Update;

public sealed class UpdateProductCommandHandler(IProductRepository repository) : IRequestHandler<UpdateProductCommand, ErrorOr<ProductDto>>
{
    public async Task<ErrorOr<ProductDto>> Handle(UpdateProductCommand command, CancellationToken cancellationToken)
    {
        List<Error> errors = [];
        var nameResult = ProductName.Create(command.Name, "name");
        errors.AddRange(nameResult.Errors.Select(error => Error.Validation(
            code: error.Code,
            description: error.Message,
            metadata: new Dictionary<string, object> { ["field"] = error.Field ?? "name" })));
        var priceResult = ProductPrice.Create(command.Price, "price");
        errors.AddRange(priceResult.Errors.Select(error => Error.Validation(
            code: error.Code,
            description: error.Message,
            metadata: new Dictionary<string, object> { ["field"] = error.Field ?? "price" })));
        if (errors.Count > 0)
        {
            return errors;
        }
        if (string.IsNullOrWhiteSpace(command.ConcurrencyToken))
        {
            return Error.Validation(
                code: "ConcurrencyToken.Required",
                description: "Concurrency token is required.",
                metadata: new Dictionary<string, object> { ["field"] = "concurrencyToken" });
        }

        var snapshot = await repository.GetByIdAsync(command.Id, cancellationToken);
        if (snapshot is null)
        {
            return Error.NotFound(code: "Product.NotFound", description: "Product was not found.");
        }
        snapshot.Entity.Update(new ProductState
        {
            IsActive = command.IsActive,
            Name = nameResult.Value!,
            Price = priceResult.Value!,
        });

        var status = await repository.UpdateAsync(snapshot.Entity, command.ConcurrencyToken, cancellationToken);
        if (status == SaveResultStatus.Conflict)
        {
            return Error.Conflict(code: "Product.ConcurrencyConflict", description: "Product was changed by another request.");
        }
        if (status == SaveResultStatus.InvalidToken)
        {
            return Error.Validation(code: "ConcurrencyToken.Invalid", description: "Invalid concurrency token.", metadata: new Dictionary<string, object> { ["field"] = "concurrencyToken" });
        }

        var updated = await repository.GetByIdAsync(command.Id, cancellationToken);
        return ToDto(updated ?? snapshot);
    }

    private static ProductDto ToDto(EntitySnapshot<Product> snapshot) => new(
        snapshot.Entity.Id,
        snapshot.Entity.IsActive,
        snapshot.Entity.Name.Value,
        snapshot.Entity.Price.Value,
        snapshot.ConcurrencyToken);
}
