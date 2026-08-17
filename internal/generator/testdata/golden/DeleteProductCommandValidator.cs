using FluentValidation;
using FluentValidation.Results;

namespace ProductService.Application.Products.Commands.Delete;

public sealed class DeleteProductCommandValidator : AbstractValidator<DeleteProductCommand>
{
    public DeleteProductCommandValidator()
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
