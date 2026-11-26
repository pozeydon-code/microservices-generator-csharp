using ErrorOr;
using MediatR;
using ProductService.Application.Common;
using ProductService.Application.Categories.Dtos;
using ProductService.Application.Categories.Interfaces;
using ProductService.Domain.Entities;


namespace ProductService.Application.Categories.Commands.Update;

public sealed class UpdateCategoryCommandHandler(ICategoryRepository repository, IUnitOfWork unitOfWork) : IRequestHandler<UpdateCategoryCommand, ErrorOr<CategoryDto>>
{
    public async Task<ErrorOr<CategoryDto>> Handle(UpdateCategoryCommand request, CancellationToken cancellationToken)
    {
        List<Error> errors = [];
        if (errors.Count > 0)
        {
            return errors;
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
            return Error.NotFound(code: "Category.NotFound", description: "Category was not found.");
        }
        snapshot.Entity.Update(new CategoryState
        {
            Name = request.Name,
            CategoryId = request.CategoryId,
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
            return Error.Conflict(code: "Category.ConcurrencyConflict", description: "Category was changed by another request.");
        }

        var updated = await repository.GetByIdAsync(request.Id, cancellationToken);
        return ToDto(updated ?? snapshot);
    }

    private static CategoryDto ToDto(EntitySnapshot<Category> snapshot) => new(
        snapshot.Entity.Id,
        snapshot.Entity.Name,
        snapshot.Entity.CategoryId,
        snapshot.ConcurrencyToken);
}
