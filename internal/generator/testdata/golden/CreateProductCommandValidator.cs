using FluentValidation;
using FluentValidation.Results;
using ProductService.Domain.ValueObjects;


namespace ProductService.Application.Products.Commands.Create;

public sealed class CreateProductCommandValidator : AbstractValidator<CreateProductCommand>
{
    public CreateProductCommandValidator()
    {
        RuleFor(command => command.Name).Custom((value, context) =>
        {
            var result = ProductName.Create(value, "name");
            if (!result.IsError) return;

            foreach (var error in result.Errors)
            {
                context.AddFailure(new ValidationFailure(context.PropertyPath, error.Description) { ErrorCode = error.Code });
            }
        });
        RuleFor(command => command.Price).Custom((value, context) =>
        {
            var result = ProductPrice.Create(value, "price");
            if (!result.IsError) return;

            foreach (var error in result.Errors)
            {
                context.AddFailure(new ValidationFailure(context.PropertyPath, error.Description) { ErrorCode = error.Code });
            }
        });
    }
}
