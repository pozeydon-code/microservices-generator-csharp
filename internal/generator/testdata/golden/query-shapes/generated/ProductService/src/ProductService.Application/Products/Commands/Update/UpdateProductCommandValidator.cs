using FluentValidation;
using FluentValidation.Results;


namespace ProductService.Application.Products.Commands.Update;

public sealed class UpdateProductCommandValidator : AbstractValidator<UpdateProductCommand>
{
    public UpdateProductCommandValidator()
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
