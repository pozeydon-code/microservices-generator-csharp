using FluentValidation;
using FluentValidation.Results;

namespace ProductService.Application.Categories.Commands.Delete;

public sealed class DeleteCategoryCommandValidator : AbstractValidator<DeleteCategoryCommand>
{
    public DeleteCategoryCommandValidator()
    {
        RuleFor(command => command.ConcurrencyToken).Custom((value, context) =>
        {
            if (string.IsNullOrWhiteSpace(value))
            {
                context.AddFailure(new ValidationFailure(context.PropertyPath, "Concurrency token is required.") { ErrorCode = "ConcurrencyToken.Required" });
            }
        });
    }
}
