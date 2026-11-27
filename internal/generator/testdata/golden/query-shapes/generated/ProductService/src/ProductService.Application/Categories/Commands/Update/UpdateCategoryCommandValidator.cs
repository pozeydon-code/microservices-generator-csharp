using FluentValidation;
using FluentValidation.Results;


namespace ProductService.Application.Categories.Commands.Update;

public sealed class UpdateCategoryCommandValidator : AbstractValidator<UpdateCategoryCommand>
{
    public UpdateCategoryCommandValidator()
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
