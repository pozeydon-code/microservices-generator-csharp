using ErrorOr;
using MediatR;
using ProductService.Application.Common;
using ProductService.Domain.Entities;
using ProductService.Domain.ValueObjects;


namespace ProductService.Application.Features.Products.Create;

public sealed class CreateProductCommandHandler(IProductRepository repository, IUnitOfWork unitOfWork) : IRequestHandler<CreateProductCommand, ErrorOr<ProductDto>>
{
    public async Task<ErrorOr<ProductDto>> Handle(CreateProductCommand request, CancellationToken cancellationToken)
    {
        List<Error> errors = [];
        var nameResult = ProductName.Create(request.Name, "name");
        errors.AddRange(nameResult.Errors.Select(error => Error.Validation(
            code: error.Code,
            description: error.Message,
            metadata: new Dictionary<string, object> { ["field"] = error.Field ?? "name" })));
        var priceResult = ProductPrice.Create(request.Price, "price");
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
            IsActive = request.IsActive,
            Name = nameResult.Value!,
            Price = priceResult.Value!,
        });
        await repository.AddAsync(entity, cancellationToken);
        try
        {
            await unitOfWork.SaveChangesAsync(cancellationToken);
        }
        catch (ConcurrencyConflictException)
        {
            return Error.Conflict(code: "Product.ConcurrencyConflict", description: "Product was changed by another request.");
        }

        var created = await repository.GetByIdAsync(entity.Id, cancellationToken);
        return ToDto(created ?? new EntitySnapshot<Product>(entity, string.Empty));
    }

    private static ProductDto ToDto(EntitySnapshot<Product> snapshot) => new(
        snapshot.Entity.Id,
        snapshot.Entity.IsActive,
        snapshot.Entity.Name.Value,
        snapshot.Entity.Price.Value,
        snapshot.ConcurrencyToken);
}
