using ErrorOr;
using MediatR;
using ProductService.Application.Common;
using ProductService.Application.Categories.Dtos;
using ProductService.Application.Categories.Interfaces;
using ProductService.Domain.Entities;


namespace ProductService.Application.Categories.Commands.Create;

public sealed class CreateCategoryCommandHandler(ICategoryRepository repository, IUnitOfWork unitOfWork) : IRequestHandler<CreateCategoryCommand, ErrorOr<CategoryDto>>
{
    public async Task<ErrorOr<CategoryDto>> Handle(CreateCategoryCommand request, CancellationToken cancellationToken)
    {
        List<Error> errors = [];
        if (errors.Count > 0)
        {
            return errors;
        }

        var entity = Category.Create(new CategoryState
        {
            Name = request.Name,
            CategoryId = request.CategoryId,
        });
        await repository.AddAsync(entity, cancellationToken);
        try
        {
            await unitOfWork.SaveChangesAsync(cancellationToken);
        }
        catch (ConcurrencyConflictException)
        {
            return Error.Conflict(code: "Category.ConcurrencyConflict", description: "Category was changed by another request.");
        }

        var created = await repository.GetByIdAsync(entity.Id, cancellationToken);
        return ToDto(created ?? new EntitySnapshot<Category>(entity, string.Empty));
    }

    private static CategoryDto ToDto(EntitySnapshot<Category> snapshot) => new(
        snapshot.Entity.Id,
        snapshot.Entity.Name,
        snapshot.Entity.CategoryId,
        snapshot.ConcurrencyToken);
}
