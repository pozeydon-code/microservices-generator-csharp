using ErrorOr;
using MediatR;
using ProductService.Application.Common;
using ProductService.Domain.Features.Products;
using ProductService.Domain.Common.ValueObjects;


namespace ProductService.Application.Features.Products.Create;

public sealed class CreateProductCommandHandler(IProductRepository repository) : IRequestHandler<CreateProductCommand, ErrorOr<ProductDto>>
{
    public async Task<ErrorOr<ProductDto>> Handle(CreateProductCommand command, CancellationToken cancellationToken)
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

        var entity = Product.Create(new ProductState
        {
            IsActive = command.IsActive,
            Name = nameResult.Value!,
            Price = priceResult.Value!,
        });
        var snapshot = await repository.AddAsync(entity, cancellationToken);
        return ToDto(snapshot);
    }

    private static ProductDto ToDto(EntitySnapshot<Product> snapshot) => new(
        snapshot.Entity.Id,
        snapshot.Entity.IsActive,
        snapshot.Entity.Name.Value,
        snapshot.Entity.Price.Value,
        snapshot.ConcurrencyToken);
}
