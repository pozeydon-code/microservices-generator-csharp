using ErrorOr;
using MediatR;
using ProductService.Application.Common;
using ProductService.Application.Products.Dtos;
using ProductService.Application.Products.Interfaces;
using ProductService.Domain.Entities;
using ProductService.Domain.ValueObjects;


namespace ProductService.Application.Products.Commands.Create;

public sealed class CreateProductCommandHandler(IProductRepository repository, IUnitOfWork unitOfWork) : IRequestHandler<CreateProductCommand, ErrorOr<ProductDto>>
{
    public async Task<ErrorOr<ProductDto>> Handle(CreateProductCommand request, CancellationToken cancellationToken)
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
