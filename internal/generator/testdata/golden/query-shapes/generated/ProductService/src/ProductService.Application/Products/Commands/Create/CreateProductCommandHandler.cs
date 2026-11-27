using ErrorOr;
using MediatR;
using ProductService.Application.Common;
using ProductService.Application.Products.Dtos;
using ProductService.Application.Products.Interfaces;
using ProductService.Domain.Entities;


namespace ProductService.Application.Products.Commands.Create;

public sealed class CreateProductCommandHandler(IProductRepository repository, IUnitOfWork unitOfWork) : IRequestHandler<CreateProductCommand, ErrorOr<ProductDto>>
{
    public async Task<ErrorOr<ProductDto>> Handle(CreateProductCommand request, CancellationToken cancellationToken)
    {
        List<Error> errors = [];
        if (errors.Count > 0)
        {
            return errors;
        }

        var entity = Product.Create(new ProductState
        {
            Name = request.Name,
            Price = request.Price,
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
        snapshot.Entity.Name,
        snapshot.Entity.Price,
        snapshot.ConcurrencyToken);
}
